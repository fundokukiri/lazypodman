package gui

import (
	"testing"

	"github.com/jesseduffield/gocui"
	"github.com/stretchr/testify/assert"

	"github.com/fundokukiri/lazypodman/pkg/commands"
)

func TestHealthcheckKeyIsBoundExactlyOnce(t *testing.T) {
	gui := newHealthGui(&fakeHealthPodClient{})
	gui.Views.Pods = &gocui.View{}
	gui.PodmanCommand.SetCapabilities(commands.Capabilities{
		CLI:  commands.Capability{State: commands.CapabilityAvailable},
		Pods: commands.Capability{State: commands.CapabilityAvailable},
	})

	matches := []*Binding{}
	for _, binding := range gui.staticKeybindings() {
		if binding.Key == 't' {
			matches = append(matches, binding)
		}
	}

	assert.Len(t, matches, 1, "'t' must be registered exactly once")
	assert.Equal(t, "containers", matches[0].ViewName)
}
