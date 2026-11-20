package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePodmanConnections(t *testing.T) {
	connections, err := parsePodmanConnections([]byte(`[{"Name":"local","URI":"unix:///run/user/1000/podman/podman.sock","Default":true},{"Name":"vps","URI":"ssh://user@example/run/user/1000/podman/podman.sock"}]`))

	assert.NoError(t, err)
	assert.Len(t, connections, 2)
	assert.Equal(t, "local", connections[0].Name)
	assert.True(t, connections[0].Default)
	assert.Equal(t, "vps", connections[1].Name)
}

func TestParsePodmanConnectionsAcceptsEmptyOutput(t *testing.T) {
	connections, err := parsePodmanConnections([]byte("[]\n"))

	assert.NoError(t, err)
	assert.Empty(t, connections)
}

func TestWithPodmanHostEnvReplacesEveryEndpoint(t *testing.T) {
	env := withPodmanHostEnv([]string{
		"PATH=/bin",
		"PODMAN_HOST=old",
		"CONTAINER_HOST=old",
		"DOCKER_HOST=old",
	}, "ssh://user@host/run/user/1000/podman/podman.sock")

	assert.Contains(t, env, "PATH=/bin")
	assert.Contains(t, env, "CONTAINER_HOST=ssh://user@host/run/user/1000/podman/podman.sock")
	assert.Contains(t, env, "DOCKER_HOST=ssh://user@host/run/user/1000/podman/podman.sock")
	assert.NotContains(t, env, "PODMAN_HOST=old")
	assert.NotContains(t, env, "CONTAINER_HOST=old")
	assert.NotContains(t, env, "DOCKER_HOST=old")
}
