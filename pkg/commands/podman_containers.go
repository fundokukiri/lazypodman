package commands

import (
	"strings"

	"github.com/docker/docker/api/types/container"
)

// labelValue returns the first non-empty value found for the given label keys.
// Podman labels compose-managed containers with both the Docker-compatible
// `com.docker.compose.*` labels and its own `io.podman.compose.*` labels, so
// callers pass the Docker key first and the Podman key as a fallback.
func labelValue(labels map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := labels[key]; ok && value != "" {
			return value
		}
	}

	return ""
}

// infraContainerLabels are set on the hidden per-pod "pause" containers that
// Podman creates. They show up in the Docker-compatible container list but are
// not something a user wants to manage from lazypodman.
var infraContainerLabels = []string{
	"io.podman.annotations.infra.name",
	"io.podman.compose.infra",
}

// isPodmanInfraContainer reports whether the container is a Podman pod infra
// (pause) container, which we hide from the containers panel. Detection is
// label-first because that is unambiguous; the name/image check is a fallback
// for Podman releases/API responses that omit the annotation labels.
func isPodmanInfraContainer(ctr container.Summary) bool {
	for _, label := range infraContainerLabels {
		if _, ok := ctr.Labels[label]; ok {
			return true
		}
	}

	if !strings.HasSuffix(containerName(ctr), "-infra") {
		return false
	}

	return strings.Contains(ctr.Image, "pause") || strings.HasPrefix(ctr.Command, "/pause")
}

// containerName returns the name of a container as reported by the API,
// without the leading slash.
func containerName(ctr container.Summary) string {
	if len(ctr.Names) == 0 {
		return ""
	}

	return strings.TrimPrefix(ctr.Names[0], "/")
}
