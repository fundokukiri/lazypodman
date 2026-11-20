//go:build !windows

package commands

import (
	"os"
	"path/filepath"
	"strconv"
)

// rootfulPodmanSocket is the socket used by the rootful Podman service.
const rootfulPodmanSocket = "/run/podman/podman.sock"

// defaultPodmanHost is used when no socket could be discovered. Rootless
// Podman is by far the most common setup, so prefer the user's socket unless
// we are running as root.
var defaultPodmanHost = func() string {
	if uid := os.Getuid(); uid > 0 {
		if xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR"); xdgRuntimeDir != "" {
			return "unix://" + filepath.Join(xdgRuntimeDir, "podman", "podman.sock")
		}
		return "unix://" + filepath.Join("/run", "user", strconv.Itoa(uid), "podman", "podman.sock")
	}

	return "unix://" + rootfulPodmanSocket
}()

// discoverPodmanSocket returns a `unix://` URL for the first Podman service
// socket that exists on this machine, or an empty string if there is none.
//
// Lazypodman does not care whether the socket belongs to a rootless or rootful
// service: whichever one is running is the one we want to show.
func discoverPodmanSocket() string {
	for _, path := range rootlessPodmanSocketCandidates() {
		if socketExists(path) {
			return "unix://" + path
		}
	}

	if socketExists(rootfulPodmanSocket) {
		return "unix://" + rootfulPodmanSocket
	}

	return ""
}

// rootlessPodmanSocketCandidates lists the places where a rootless Podman
// service puts its socket. Podman uses $XDG_RUNTIME_DIR/podman/podman.sock;
// when that variable is not set the runtime dir is the usual /run/user/<uid>.
func rootlessPodmanSocketCandidates() []string {
	dirs := make([]string, 0, 3)

	if xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR"); xdgRuntimeDir != "" {
		dirs = append(dirs, xdgRuntimeDir)
	}

	if uid := os.Getuid(); uid >= 0 {
		dirs = append(dirs, filepath.Join("/run/user", strconv.Itoa(uid)))
		// Podman falls back to this location when XDG_RUNTIME_DIR is unset.
		dirs = append(dirs, filepath.Join(os.TempDir(), "podman-run-"+strconv.Itoa(uid)))
	}

	paths := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		paths = append(paths, filepath.Join(dir, "podman", "podman.sock"))
	}

	return paths
}
