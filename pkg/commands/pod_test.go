package commands

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
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

// fakePodClient lets the pod-state tests run without a podman binary: each field
// is the canned answer for one call, and the counters show batching.
type fakePodClient struct {
	enabled    bool
	podList    []*Pod
	podListErr error
	psMap      map[string]string
	psErr      error

	listCalls int
	psCalls   int
}

func (f *fakePodClient) Enabled() bool   { return f.enabled }
func (f *fakePodClient) Reason() string  { return "fake" }
func (f *fakePodClient) Version() string { return "5.0.0" }
func (f *fakePodClient) PodInspect(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}
func (f *fakePodClient) PodAction(_ context.Context, _, _ string, _ ...string) error { return nil }
func (f *fakePodClient) PodPrune(_ context.Context) error                            { return nil }
func (f *fakePodClient) PodLogs(_ context.Context, _ *Pod, _ []*Container, _ int) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (f *fakePodClient) PodList(_ context.Context) ([]*Pod, error) {
	f.listCalls++

	return f.podList, f.podListErr
}

func (f *fakePodClient) ContainerPodMap(_ context.Context) (map[string]string, error) {
	f.psCalls++

	return f.psMap, f.psErr
}

func TestJoinPods(t *testing.T) {
	infraID := "infra1"
	pods := []*Pod{
		{Name: "web", InfraID: infraID},
		{Name: "noinfra", InfraID: ""},
		{Name: "gone", InfraID: "missing-infra"},
	}
	infra := map[string]container.Summary{
		infraID: {Ports: []container.Port{
			{PublicPort: 8080, PrivatePort: 80, Type: "tcp", IP: "0.0.0.0"},
			{PublicPort: 8080, PrivatePort: 80, Type: "tcp", IP: "0.0.0.0"}, // duplicate, must collapse
			{PrivatePort: 443, Type: "tcp"},
		}},
	}
	podByContainer := map[string]string{"ctr-web": "web", "ctr-orphan": "gone", "": "ignored", "ctr-nopod": ""}

	state := joinPods(pods, podByContainer, infra)

	assert.Equal(t, "443/tcp, 8080->80/tcp", pods[0].Ports)
	assert.Empty(t, pods[1].Ports, "pod without infra has no ports")
	assert.Empty(t, pods[2].Ports, "missing infra summary is tolerated")
	assert.Equal(t, map[string]string{"ctr-web": "web", "ctr-orphan": "gone"}, state.PodByContainer)
}

func TestFormatPorts(t *testing.T) {
	assert.Empty(t, formatPorts(nil))
	assert.Equal(t, "127.0.0.1:9000->9000/udp", formatPorts([]container.Port{
		{PublicPort: 9000, PrivatePort: 9000, Type: "udp", IP: "127.0.0.1"},
	}))
}

func TestRefreshPodStateBatchesAndCaches(t *testing.T) {
	infraID := "infra1"
	client := &fakePodClient{
		enabled: true,
		podList: []*Pod{{Name: "web", InfraID: infraID}},
		psMap:   map[string]string{"ctr-web": "web"},
	}
	podmanCommand := &PodmanCommand{
		Log:             NewDummyLog(),
		infraContainers: map[string]container.Summary{infraID: {Ports: []container.Port{{PublicPort: 80, PrivatePort: 80, Type: "tcp"}}}},
	}
	podmanCommand.podClient = client

	ctx := context.Background()
	state, err := podmanCommand.RefreshPodState(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, client.listCalls)
	assert.Equal(t, 1, client.psCalls)
	assert.Equal(t, "80->80/tcp", state.Pods[0].Ports)
	assert.Equal(t, map[string]string{"ctr-web": "web"}, state.PodByContainer)

	// A failing refresh returns the last snapshot instead of an empty panel.
	client.podListErr = errors.New("podman pod ls: exit status 125")
	state, err = podmanCommand.RefreshPodState(ctx)
	assert.Error(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, "web", state.Pods[0].Name)
}

func TestRefreshPodStateDisabledIsNoop(t *testing.T) {
	client := &fakePodClient{enabled: false}
	podmanCommand := &PodmanCommand{Log: NewDummyLog()}
	podmanCommand.podClient = client

	state, err := podmanCommand.RefreshPodState(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, state)
	assert.Zero(t, client.listCalls)
}

func TestAggregatePodStats(t *testing.T) {
	newStats := func(cpu float64, usage int, limit int64) *RecordedStats {
		stats := &RecordedStats{DerivedStats: DerivedStats{CPUPercentage: cpu}}
		stats.ClientStats.MemoryStats.Usage = usage
		stats.ClientStats.MemoryStats.Limit = limit

		return stats
	}

	web := &Container{}
	web.StatHistory = []*RecordedStats{newStats(0.5, 1, 1), newStats(1.234, 100, 1000)}

	side := &Container{}
	side.StatHistory = []*RecordedStats{newStats(2.5, 200, 1000)}

	noStats := &Container{}

	row := AggregatePodStats([]*Container{web, side, noStats})

	assert.InDelta(t, 3.73, row.CPU, 0.0001)
	assert.Equal(t, int64(300), row.MemUsage)
	assert.Equal(t, int64(2000), row.MemLimit)

	empty := AggregatePodStats(nil)
	assert.Zero(t, empty.CPU)
	assert.Zero(t, empty.MemUsage)
}
