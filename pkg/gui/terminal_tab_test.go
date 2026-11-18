package gui

import (
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"

	dockertypes "github.com/docker/docker/api/types/container"
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
		ctrlO      bool
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

		if binding.Key == gocui.KeyCtrlO {
			ctrlO = true
		}

		if binding.Key == gocui.KeySpace || binding.Key == gocui.KeyEnter || binding.Key == gocui.KeyEsc {
			assert.False(t, binding.Modifier == gocui.ModAlt)
		}
	}

	assert.Equal(t, 94, printables, "all printable ASCII keys are forwarded")
	assert.True(t, ctrlX)
	assert.True(t, ctrlO, "ctrl+o leaves the shell focused")
	assert.Equal(t, 1, closeCount, "ctrl+x closes the shell exactly once")
}

func containerSummary(image, status string) dockertypes.Summary {
	return dockertypes.Summary{Image: image, Status: status}
}

func TestTerminalHeaderShowsContainerInfo(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	container := &commands.Container{
		Name:      "web",
		ID:        "0123456789abcdef0123",
		PodName:   "app-pod",
		Container: containerSummary("nginx:alpine", "Up 5 minutes"),
	}

	header := terminalHeader(container, gui.Tr)

	assert.Contains(t, header, "container: web")
	assert.Contains(t, header, "image: nginx:alpine")
	assert.Contains(t, header, "status: Up 5 minutes")
	assert.Contains(t, header, "id: 0123456789ab")
	assert.Contains(t, header, "pod: app-pod")
}

func TestTerminalHeaderOmitsEmptyPod(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	container := &commands.Container{
		Name:      "web",
		ID:        "abc",
		Container: containerSummary("nginx", "running"),
	}

	assert.NotContains(t, terminalHeader(container, gui.Tr), "pod:")
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

func TestTerminalOptionsShowHowToClose(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	options := gui.terminalOptionsMap()

	assert.Equal(t, map[string]string{
		"ctrl+o": gui.Tr.LeaveShell,
		"ctrl+x": gui.Tr.CloseShell,
	}, options)
	assert.Equal(t, "ctrl+o: leave shell, keep it running, ctrl+x: close shell", gui.optionsMapToString(options))
}
