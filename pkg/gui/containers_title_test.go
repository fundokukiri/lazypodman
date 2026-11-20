package gui

import (
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/stretchr/testify/assert"

	"github.com/docker/docker/api/types/container"
)

func TestContainerTerminalTitle(t *testing.T) {
	ctr := &commands.Container{Name: "web"}
	ctr.ID = "abcdef0123456789"
	ctr.Details = container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{Path: "/usr/sbin/nginx"},
	}

	assert.Equal(t, "Container: web - abcdef012345", containerTerminalTitle(ctr, false))
	assert.Equal(t, "Container: web - abcdef012345 - nginx", containerTerminalTitle(ctr, true))
}

func TestContainerTerminalTitleFallsBackToTheCommand(t *testing.T) {
	ctr := &commands.Container{Name: "web"}
	ctr.ID = "abcdef0123456789"
	ctr.Details = container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{},
		Config:            &container.Config{Cmd: []string{"/bin/bash", "-c", "sleep 1"}},
	}

	assert.Equal(t, "Container: web - abcdef012345 - bash", containerTerminalTitle(ctr, true))
}

func TestContainerTerminalTitleWithUnknownProcess(t *testing.T) {
	ctr := &commands.Container{Name: "web"}
	ctr.ID = "abcdef0123456789"

	assert.Equal(t, "Container: web - abcdef012345 - running process", containerTerminalTitle(ctr, true))
}
