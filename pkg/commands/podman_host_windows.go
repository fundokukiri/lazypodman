//go:build windows

package commands

// defaultPodmanHost is the named pipe of the default Podman machine.
const defaultPodmanHost = "npipe:////./pipe/podman-machine-default"

// discoverPodmanSocket is a no-op on Windows: Podman on Windows always runs
// inside a `podman machine` and is reached over its named pipe.
func discoverPodmanSocket() string {
	return ""
}
