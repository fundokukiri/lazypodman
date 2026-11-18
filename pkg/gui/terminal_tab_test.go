package gui

import (
	"regexp"
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
	plain := regexp.MustCompile("\x1b\\[[0-9;]*m").ReplaceAllString(header, "")

	assert.Contains(t, plain, "container: web")
	assert.Contains(t, plain, "image: nginx:alpine")
	assert.Contains(t, plain, "status: Up 5 minutes")
	assert.Contains(t, plain, "id: 0123456789ab")
	assert.Contains(t, plain, "pod: app-pod")
	assert.Contains(t, header, "\x1b[32m", "running status is green")
}

func TestTerminalHeaderOmitsEmptyPod(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	container := &commands.Container{
		Name:      "web",
		ID:        "abc",
		Container: containerSummary("nginx", "running"),
	}

	header := terminalHeader(container, gui.Tr)
	assert.NotContains(t, regexp.MustCompile("\x1b\\[[0-9;]*m").ReplaceAllString(header, ""), "pod:")
}

func TestTerminalScrollBindingsIncludeTheMouseWheel(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	keys := map[interface{}]bool{}
	for _, binding := range gui.terminalKeyBindings() {
		keys[binding.Key] = true
	}

	assert.True(t, keys[gocui.MouseWheelUp])
	assert.True(t, keys[gocui.MouseWheelDown])
	assert.True(t, keys[gocui.KeyPgup])
	assert.True(t, keys[gocui.KeyPgdn])
}

func TestScrolledOffLinesDetectsScrolling(t *testing.T) {
	oldRows := []string{"a", "b", "c", "d"}
	newRows := []string{"c", "d", "e", "f"}

	assert.Equal(t, []string{"a", "b"}, scrolledOffLines(oldRows, newRows))
	assert.Nil(t, scrolledOffLines(oldRows, []string{"x", "y", "z", "w"}))
}

func TestTerminalScrollbarTracksOffset(t *testing.T) {
	start, size, ok := terminalScrollbar(100, 20, 0)
	assert.True(t, ok)
	assert.Equal(t, 16, start, "thumb sits flush at the bottom when scrolled to the newest line")
	assert.Equal(t, 4, size)

	topStart, _, _ := terminalScrollbar(100, 20, 80)
	assert.Equal(t, 0, topStart)

	_, _, ok = terminalScrollbar(10, 20, 0)
	assert.False(t, ok, "no scrollbar when everything fits")
}

func TestTerminalWindowPagesHistory(t *testing.T) {
	history := []string{"1", "2", "3", "4"}
	screen := []string{"5", "6", "7"}

	assert.Equal(t, []string{"5", "6", "7"}, terminalWindow(history, screen, 3, 0))
	assert.Equal(t, []string{"3", "4", "5"}, terminalWindow(history, screen, 3, 2))
	assert.Equal(t, []string{"1", "2", "3"}, terminalWindow(history, screen, 3, 99), "offset is clamped")
	assert.Equal(t, []string{"1", "2", "3", "4", "5", "6", "7"}, terminalWindow(history, screen, 20, 0))
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
		"wheel/PgUp/PgDn": gui.Tr.Scroll,
		"ctrl+o":          gui.Tr.LeaveShell,
		"ctrl+x":          gui.Tr.CloseShell,
	}, options)
	assert.Equal(t, "ctrl+o: leave shell, keep it running, ctrl+x: close shell, wheel/PgUp/PgDn: scroll", gui.optionsMapToString(options))
}
