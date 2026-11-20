package commands

import (
	"os"
	"strings"
)

// podmanHostEnvKeys are the environment variables that tell us which Podman
// service endpoint to talk to, checked in order of decreasing precedence.
//
// CONTAINER_HOST is how the Podman CLI (and `podman system service`) advertises
// a remote/socket endpoint. DOCKER_HOST is still honored because podman-docker,
// docker-compose and various CI setups export it for a Podman socket.
// PODMAN_HOST is a lazypodman-specific override that wins over both.
var podmanHostEnvKeys = []string{"PODMAN_HOST", "CONTAINER_HOST", "DOCKER_HOST"}

// determinePodmanHost determines the Podman service endpoint we should connect
// to, in the following order of decreasing precedence:
//   - PODMAN_HOST, CONTAINER_HOST or DOCKER_HOST environment variables
//   - an existing rootless Podman socket (e.g. $XDG_RUNTIME_DIR/podman/podman.sock)
//   - an existing rootful Podman socket (/run/podman/podman.sock)
//   - the default endpoint for the host operating system
func determinePodmanHost() string {
	for _, key := range podmanHostEnvKeys {
		if host := os.Getenv(key); host != "" {
			return normalizePodmanHost(host)
		}
	}

	if host := discoverPodmanSocket(); host != "" {
		return host
	}

	return defaultPodmanHost
}

// normalizePodmanHost accepts both URL style endpoints (unix://, tcp://,
// ssh://) and a bare socket path, which is what people often put in
// CONTAINER_HOST by mistake. The API client requires a scheme, so we add one
// for absolute paths.
func normalizePodmanHost(host string) string {
	if strings.Contains(host, "://") {
		return host
	}

	if strings.HasPrefix(host, "/") {
		return "unix://" + host
	}

	return host
}

func socketExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeSocket != 0
}
