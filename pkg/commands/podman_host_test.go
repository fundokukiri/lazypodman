package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
)

func TestDeterminePodmanHostPrecedence(t *testing.T) {
	scenarios := []struct {
		testName  string
		podmanEnv string
		contEnv   string
		dockerEnv string
		expected  string
	}{
		{
			testName:  "podman host wins over everything else",
			podmanEnv: "unix:///tmp/podman.sock",
			contEnv:   "unix:///tmp/container.sock",
			dockerEnv: "unix:///tmp/docker.sock",
			expected:  "unix:///tmp/podman.sock",
		},
		{
			testName:  "container host wins over docker host",
			contEnv:   "unix:///tmp/container.sock",
			dockerEnv: "unix:///tmp/docker.sock",
			expected:  "unix:///tmp/container.sock",
		},
		{
			testName:  "docker host is still honored",
			dockerEnv: "unix:///tmp/docker.sock",
			expected:  "unix:///tmp/docker.sock",
		},
		{
			testName: "bare socket path gets a unix scheme",
			contEnv:  "/run/user/1000/podman/podman.sock",
			expected: "unix:///run/user/1000/podman/podman.sock",
		},
		{
			testName: "falls back to the platform default",
			expected: defaultPodmanHost,
		},
	}

	for _, s := range scenarios {
		s := s
		t.Run(s.testName, func(t *testing.T) {
			t.Setenv("PODMAN_HOST", s.podmanEnv)
			t.Setenv("CONTAINER_HOST", s.contEnv)
			t.Setenv("DOCKER_HOST", s.dockerEnv)

			if s.testName == "falls back to the platform default" {
				// Make sure discovery does not pick up a real socket from the machine.
				t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
			}

			actual := determinePodmanHost()
			assert.Equal(t, s.expected, actual)
		})
	}
}

func TestDiscoverRootlessPodmanSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("rootless unix socket discovery is unix-only")
	}

	xdgRuntimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", xdgRuntimeDir)

	socketPath := filepath.Join(xdgRuntimeDir, "podman", "podman.sock")
	assert.NoError(t, os.MkdirAll(filepath.Dir(socketPath), 0o755))
	assert.NoError(t, os.WriteFile(socketPath, []byte{}, 0o600))

	t.Setenv("PODMAN_HOST", "")
	t.Setenv("CONTAINER_HOST", "")
	t.Setenv("DOCKER_HOST", "")

	host := determinePodmanHost()
	assert.Equal(t, "unix://"+socketPath, host)
}

func TestIsPodmanInfraContainer(t *testing.T) {
	scenarios := []struct {
		testName  string
		summary   container.Summary
		expected  bool
		reasoning string
	}{
		{
			testName: "regular container",
			summary: container.Summary{
				Names:   []string{"/web"},
				Image:   "docker.io/library/nginx",
				Command: "nginx -g 'daemon off;'",
			},
			expected: false,
		},
		{
			testName: "container named like an infra container but not a pause container",
			summary: container.Summary{
				Names:   []string{"/my-infra"},
				Image:   "docker.io/library/alpine",
				Command: "sh",
			},
			expected: false,
		},
		{
			testName: "infra container detected by name, image and command",
			summary: container.Summary{
				Names:   []string{"/mypod-infra"},
				Image:   "k8s.gcr.io/pause:3.9",
				Command: "/pause",
			},
			expected: true,
		},
		{
			testName: "infra container detected by label",
			summary: container.Summary{
				Names:  []string{"/mypod-infra"},
				Labels: map[string]string{"io.podman.annotations.infra.name": "mypod"},
			},
			expected: true,
		},
	}

	for _, s := range scenarios {
		s := s
		t.Run(s.testName, func(t *testing.T) {
			assert.Equal(t, s.expected, isPodmanInfraContainer(s.summary))
		})
	}
}

func TestLabelValueChoosesFirstNonEmpty(t *testing.T) {
	labels := map[string]string{
		"io.podman.compose.project": "podman-project",
		"io.podman.compose.service": "",
	}

	assert.Equal(t, "podman-project", labelValue(labels, "com.docker.compose.project", "io.podman.compose.project"))
	assert.Equal(t, "", labelValue(labels, "com.docker.compose.service", "io.podman.compose.service"))
	assert.Equal(t, "", labelValue(labels, "com.docker.compose.oneoff"))
}
