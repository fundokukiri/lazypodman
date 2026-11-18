package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fundokukiri/lazypodman/pkg/commands"
)

func TestCapabilityOffMessages(t *testing.T) {
	t.Run("no podman command at all", func(t *testing.T) {
		gui := &Gui{Tr: translations()}

		assert.Empty(t, gui.capabilityOffMessages())
	})

	t.Run("cli down shows one line only", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})
		gui.PodmanCommand.SetCapabilities(commands.Capabilities{
			CLI:       commands.Capability{State: commands.CapabilityUnavailable, Reason: "podman CLI not found"},
			Pods:      commands.Capability{State: commands.CapabilityUnavailable, Reason: "podman CLI not found"},
			HealthRun: commands.Capability{State: commands.CapabilityUnavailable, Reason: "podman CLI not found"},
		})

		assert.Equal(t, []string{"cli off: podman CLI not found"}, gui.capabilityOffMessages())
	})

	t.Run("pods disabled in config", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})
		gui.PodmanCommand.SetCapabilities(commands.Capabilities{
			CLI: commands.Capability{State: commands.CapabilityUnknown},
			Pods: commands.Capability{
				State:  commands.CapabilityUnavailable,
				Reason: "disabled in config",
			},
		})

		assert.Equal(t, []string{"pods off: disabled in config"}, gui.capabilityOffMessages())
	})

	t.Run("only a probed and failed feature claims a line", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})
		gui.PodmanCommand.SetCapabilities(commands.Capabilities{
			CLI:  commands.Capability{State: commands.CapabilityAvailable},
			Pods: commands.Capability{State: commands.CapabilityAvailable},
			HealthRun: commands.Capability{
				State:  commands.CapabilityUnavailable,
				Reason: "no healthcheck subcommand",
			},
		})

		assert.Equal(t, []string{"health run off: no healthcheck subcommand"}, gui.capabilityOffMessages())
	})

	t.Run("unknown adds nothing", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})

		assert.Empty(t, gui.capabilityOffMessages())
	})
}
