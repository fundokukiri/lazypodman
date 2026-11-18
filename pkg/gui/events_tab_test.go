package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fundokukiri/lazypodman/pkg/commands"
)

func TestEventsUnavailableExplainsWhy(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	gui.PodmanCommand.SetCapabilities(commands.Capabilities{
		CLI: commands.Capability{State: commands.CapabilityUnavailable, Reason: "podman CLI not found"},
	})

	assert.Equal(t, "cli off: podman CLI not found", gui.eventsUnavailable())
}

func TestJoinLines(t *testing.T) {
	assert.Equal(t, "", joinLines(nil))
	assert.Equal(t, "a\nb", joinLines([]string{"a", "b"}))
}
