package gui

import (
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"

	"github.com/jesseduffield/gocui"
	"github.com/stretchr/testify/assert"
)

func TestTrimTerminalRows(t *testing.T) {
	rows := trimTerminalRows([]string{"hello   ", "", "world\x00\x00", "", ""})

	assert.Equal(t, []string{"hello", "", "world"}, rows)
}

func TestTerminalKeysAreForwardedAndCtrlXCloses(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	var (
		closeCount int
		printables int
		ctrlX      bool
	)

	for _, binding := range gui.terminalKeyBindings() {
		assert.Equal(t, "terminal", binding.ViewName)

		if _, isRune := binding.Key.(rune); isRune {
			printables++
		}

		if binding.Key == gocui.KeyCtrlX {
			ctrlX = true
			closeCount++
		}

		if binding.Key == gocui.KeySpace || binding.Key == gocui.KeyEnter || binding.Key == gocui.KeyEsc {
			assert.False(t, binding.Modifier == gocui.ModAlt)
		}
	}

	assert.Equal(t, 94, printables, "all printable ASCII keys are forwarded")
	assert.True(t, ctrlX)
	assert.Equal(t, 1, closeCount, "ctrl+x closes the shell exactly once")
}

func TestTerminalTabOnlyForItsContainer(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	container := &commands.Container{ID: "abc", Name: "web"}

	assert.False(t, gui.terminalTabFor(container))
	assert.False(t, gui.terminalTabFor(nil))

	gui.setTerminal(&terminalState{containerID: "abc"})
	assert.True(t, gui.terminalTabFor(container))
	assert.False(t, gui.terminalTabFor(&commands.Container{ID: "other"}))

	gui.setTerminal(nil)
	assert.False(t, gui.terminalTabFor(container))
}
