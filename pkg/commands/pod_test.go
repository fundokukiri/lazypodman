package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// readPodFixture loads a SYNTHETIC fixture from testdata/pods. See that
// directory's README: the JSON shapes are not captured from a real host.
func readPodFixture(t *testing.T, name string) []byte {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("testdata", "pods", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	return content
}

func TestParsePodsFixtures(t *testing.T) {
	t.Run("v5.2", func(t *testing.T) {
		pods, err := parsePods(readPodFixture(t, "v5.2.json"))
		assert.NoError(t, err)
		assert.Len(t, pods, 3)

		web := pods[0]
		assert.Equal(t, "web", web.Name)
		assert.Equal(t, "Running", web.Status)
		assert.Equal(t, 2, web.NumberOfContainers)
		assert.Equal(t, "1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f809", web.InfraID)
		assert.True(t, web.HasInfra())
		assert.Equal(t, "web", web.Labels["app"])

		batch := pods[1]
		assert.Equal(t, "batch", batch.Name)
		assert.Equal(t, "Exited", batch.Status)
		assert.Equal(t, 0, batch.NumberOfContainers)
		assert.True(t, batch.HasInfra())

		noinfra := pods[2]
		assert.Equal(t, "noinfra", noinfra.Name)
		assert.Empty(t, noinfra.InfraID)
		assert.False(t, noinfra.HasInfra())
		assert.Equal(t, 1, noinfra.NumberOfContainers)
	})

	t.Run("v4.9", func(t *testing.T) {
		pods, err := parsePods(readPodFixture(t, "v4.9.json"))
		assert.NoError(t, err)
		assert.Len(t, pods, 2)

		// Created stays a raw string: some versions print RFC3339, others
		// human text.
		assert.Equal(t, "2 hours ago", pods[0].Created)
		assert.Empty(t, pods[0].Labels)
		assert.Equal(t, "Stopped", pods[1].Status)
		assert.Equal(t, 3, pods[1].NumberOfContainers)
	})
}

func TestParsePodsTolerance(t *testing.T) {
	for _, in := range []string{"", "   \n", "null", "[]", readFixtureString(t, "null.json"), readFixtureString(t, "empty.json")} {
		pods, err := parsePods([]byte(in))
		assert.NoError(t, err, "input %q", in)
		assert.Empty(t, pods, "input %q", in)
	}
}

func TestParsePodsMalformed(t *testing.T) {
	_, err := parsePods(readPodFixture(t, "malformed.json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pod ls")
}

func TestParseContainerPodMap(t *testing.T) {
	podByContainer, err := parseContainerPodMap(readPodFixture(t, "ps_v5.json"))
	assert.NoError(t, err)
	assert.Len(t, podByContainer, 3)

	assert.Equal(t, "web", podByContainer["1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f809"])
	assert.Equal(t, "web", podByContainer["a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"])
	assert.Equal(t, "web", podByContainer["b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90a1"])

	// The standalone container has no pod and must not appear.
	_, ok := podByContainer["c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2"]
	assert.False(t, ok)
}

func TestParseContainerPodMapTolerance(t *testing.T) {
	for _, in := range []string{"", "null", "[]"} {
		podByContainer, err := parseContainerPodMap([]byte(in))
		assert.NoError(t, err, "input %q", in)
		assert.Empty(t, podByContainer, "input %q", in)
	}

	_, err := parseContainerPodMap([]byte("not json"))
	assert.Error(t, err)
}

func readFixtureString(t *testing.T, name string) string {
	t.Helper()

	return string(readPodFixture(t, name))
}
