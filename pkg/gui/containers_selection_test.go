package gui

import (
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/gui/panels"
	"github.com/stretchr/testify/assert"
)

func TestRestoreSelectedContainerFollowsTheIDAfterRestart(t *testing.T) {
	restarted := &commands.Container{ID: "restarted"}
	other := &commands.Container{ID: "other"}
	panel := &panels.SideListPanel[*commands.Container]{
		ListPanel: panels.ListPanel[*commands.Container]{
			List: panels.NewFilteredList[*commands.Container](),
		},
	}
	panel.List.SetItems([]*commands.Container{other, restarted})
	panel.SelectedIdx = 1
	gui := &Gui{Panels: Panels{Containers: panel}}

	// podman now reports the restarted container as running, so the list moves
	// it to the top.
	panel.List.SetItems([]*commands.Container{restarted, other})
	gui.restoreSelectedContainer(restarted)

	assert.Equal(t, 0, panel.SelectedIdx)
	selected, err := panel.GetSelectedItem()
	assert.NoError(t, err)
	assert.Equal(t, restarted, selected)
}

func TestRestoreSelectedContainerLeavesTheIndexWhenTheContainerIsGone(t *testing.T) {
	panel := &panels.SideListPanel[*commands.Container]{
		ListPanel: panels.ListPanel[*commands.Container]{
			List: panels.NewFilteredList[*commands.Container](),
		},
	}
	panel.List.SetItems([]*commands.Container{{ID: "one"}, {ID: "two"}})
	panel.SelectedIdx = 1
	gui := &Gui{Panels: Panels{Containers: panel}}

	gui.restoreSelectedContainer(&commands.Container{ID: "gone"})

	assert.Equal(t, 1, panel.SelectedIdx)
}
