package gui

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/go-connections/nat"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
)

func TestApplyImageDefaultsUsesImageCommandAndWorkdir(t *testing.T) {
	state := newRunContainerState("alpine")
	inspect := imagetypes.InspectResponse{Config: &dockerspec.DockerOCIImageConfig{
		ImageConfig: ocispec.ImageConfig{
			Cmd:        []string{"sh", "-c", "echo hello world"},
			Entrypoint: []string{"/entrypoint"},
			WorkingDir: "/srv",
			User:       "1000",
			ExposedPorts: map[string]struct{}{
				"443/tcp": {},
				"80/tcp":  {},
			},
			Env:    []string{"A=1", "B=2"},
			Labels: map[string]string{"role": "web"},
		},
	}}

	applyImageDefaults(state, inspect)

	assert.Equal(t, "sh -c 'echo hello world'", state.text["command"])
	assert.Equal(t, "/entrypoint", state.text["entrypoint"])
	assert.Equal(t, "/srv", state.text["workdir"])
	assert.Equal(t, "1000", state.text["user"])
	assert.Equal(t, []string{"443/tcp", "80/tcp"}, state.imageInfo.Ports)
	assert.Equal(t, 2, state.imageInfo.EnvCount)
	assert.Equal(t, 1, state.imageInfo.Labels)
	assert.Equal(t, "ports 443/tcp, 80/tcp · env 2 · labels 1", state.imageInfo.summary())
}

func TestApplyImageDefaultsSkipsMultiArgumentEntrypoint(t *testing.T) {
	state := newRunContainerState("alpine")
	inspect := imagetypes.InspectResponse{Config: &dockerspec.DockerOCIImageConfig{
		ImageConfig: ocispec.ImageConfig{Entrypoint: []string{"/bin/sh", "-c"}},
	}}

	applyImageDefaults(state, inspect)

	assert.Equal(t, "", state.text["entrypoint"])
}

func TestApplyImageDefaultsExtractsHealthCommand(t *testing.T) {
	state := newRunContainerState("alpine")
	inspect := imagetypes.InspectResponse{Config: &dockerspec.DockerOCIImageConfig{
		DockerOCIImageConfigExt: dockerspec.DockerOCIImageConfigExt{
			Healthcheck: &dockerspec.HealthcheckConfig{Test: []string{"CMD-SHELL", "curl -f localhost"}},
		},
	}}

	applyImageDefaults(state, inspect)

	assert.Equal(t, "curl -f localhost", state.text["health_cmd"])
}

func TestApplyRunLikeContainerCopiesVisibleSettings(t *testing.T) {
	state := newRunContainerState("newimage")
	ctr := &commands.Container{
		Name:    "source",
		Details: container.InspectResponse{ContainerJSONBase: &container.ContainerJSONBase{}},
	}
	ctr.Details.Config = &container.Config{
		Image:      "alpine",
		Cmd:        []string{"sleep", "10"},
		Env:        []string{"A=1"},
		Hostname:   "web",
		Tty:        true,
		OpenStdin:  true,
		WorkingDir: "/app",
		Labels:     map[string]string{"role": "web"},
	}
	ctr.Details.HostConfig = &container.HostConfig{
		AutoRemove:    true,
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		PortBindings: nat.PortMap{
			"80/tcp": []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "8080"}},
		},
	}
	ctr.Details.Mounts = []container.MountPoint{
		{Type: "bind", Source: "/tmp/data", Destination: "/data", RW: true},
		{Type: "volume", Name: "cache", Destination: "/cache", RW: true},
	}

	applyRunLikeContainer(state, ctr)

	assert.Equal(t, "sleep 10", state.text["command"])
	assert.Equal(t, "/app", state.text["workdir"])
	assert.Equal(t, "web", state.text["hostname"])
	assert.Equal(t, "unless-stopped", state.text["restart"])
	assert.True(t, state.toggles["tty"])
	assert.True(t, state.toggles["interactive"])
	assert.True(t, state.toggles["rm"])
	assert.Equal(t, []string{"A=1"}, state.lists["env"])
	assert.Equal(t, []string{"role=web"}, state.lists["labels"])
	assert.Equal(t, []string{"127.0.0.1:8080:80/tcp"}, state.lists["ports"])
	assert.Equal(t, []string{"/tmp/data:/data", "cache:/cache"}, state.lists["volumes"])
}

func TestValidateRunStateBlocksNameAndPortConflicts(t *testing.T) {
	state := newRunContainerState("alpine")
	state.text["name"] = "web"
	state.lists["ports"] = []string{"8080:80"}
	containers := []*commands.Container{{
		Name: "web",
		Container: container.Summary{
			Ports: []container.Port{{PublicPort: 8080, Type: "tcp"}},
		},
	}}

	issues := validateRunState(state, containers)

	assert.Len(t, issues, 2)
	assert.True(t, issues[0].Blocking)
	assert.True(t, issues[1].Blocking)
}

func TestValidateRunStateAcceptsReplaceForNameConflict(t *testing.T) {
	state := newRunContainerState("alpine")
	state.text["name"] = "web"
	state.toggles["replace"] = true
	containers := []*commands.Container{{Name: "web"}}

	for _, issue := range validateRunState(state, containers) {
		assert.False(t, issue.Blocking)
	}
}

func TestValidateRunStateRejectsInvalidPorts(t *testing.T) {
	state := newRunContainerState("alpine")
	state.lists["ports"] = []string{"99999:80", "8080:0", "8080:80/qx"}

	issues := validateRunState(state, nil)

	assert.Len(t, issues, 3)
	for _, issue := range issues {
		assert.True(t, issue.Blocking)
	}
}

func TestValidateRunStateWarnsAboutMissingBindAndPrivileges(t *testing.T) {
	state := newRunContainerState("alpine")
	state.lists["volumes"] = []string{"/definitely/missing:/data"}
	state.toggles["privileged"] = true

	issues := validateRunState(state, nil)

	assert.Len(t, issues, 2) // missing path, privileged
	for _, issue := range issues {
		assert.False(t, issue.Blocking)
	}
}

func TestCopyRunStateDoesNotShareMaps(t *testing.T) {
	state := newRunContainerState("alpine")
	state.text["name"] = "web"
	state.lists["ports"] = []string{"8080:80"}

	copyState := copyRunState(state)
	copyState.text["name"] = "other"
	copyState.lists["ports"][0] = "9090:90"

	assert.Equal(t, "web", state.text["name"])
	assert.Equal(t, "8080:80", state.lists["ports"][0])
}

func TestLastRunStateRoundTrip(t *testing.T) {
	gui := &Gui{}
	state := newRunContainerState("alpine")
	state.text["name"] = "web"

	gui.rememberRunState(state)
	restored := gui.lastRunFor("alpine")

	assert.NotNil(t, restored)
	assert.Equal(t, "web", restored.text["name"])
	assert.Nil(t, gui.lastRunFor("missing"))
}
