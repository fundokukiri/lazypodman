package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fundokukiri/lazypodman/pkg/commands"
)

func TestMarkedContainers(t *testing.T) {
	items := []*commands.Container{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	marked := map[string]bool{"c": true, "a": true, "gone": true}

	selected := markedContainers(items, marked)

	assert.Len(t, selected, 2)
	assert.Equal(t, "a", selected[0].ID)
	assert.Equal(t, "c", selected[1].ID, "list order is kept, not map order")
}

func TestToggleMarkedContainer(t *testing.T) {
	gui := &Gui{}

	assert.True(t, gui.toggleMarkedContainer("a"))
	assert.Equal(t, 1, gui.markedContainerCount())
	assert.True(t, gui.isContainerMarked("a"))

	assert.False(t, gui.toggleMarkedContainer("a"))
	assert.Equal(t, 0, gui.markedContainerCount())
	assert.False(t, gui.isContainerMarked("a"))

	gui.toggleMarkedContainer("b")
	gui.clearMarkedContainers()
	assert.Equal(t, 0, gui.markedContainerCount())
}

func TestContainerMarkPrefix(t *testing.T) {
	gui := &Gui{}
	gui.toggleMarkedContainer("abc")

	assert.Equal(t, "● ", gui.containerMarkPrefix("abc"))
	assert.Equal(t, "", gui.containerMarkPrefix("other"))
}
