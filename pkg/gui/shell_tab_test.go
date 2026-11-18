package gui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/stretchr/testify/assert"
)

func TestShellStateCapsBufferedLines(t *testing.T) {
	state := &shellState{}

	for i := 0; i < maxShellRows+25; i++ {
		state.append(fmt.Sprintf("line %d", i))
	}

	lines := state.snapshot()
	assert.Len(t, lines, maxShellRows)
	assert.Equal(t, "line 25", lines[0], "oldest lines are dropped first")
	assert.Equal(t, fmt.Sprintf("line %d", maxShellRows+24), lines[len(lines)-1])
}

func TestShellStateSnapshotIsACopy(t *testing.T) {
	state := &shellState{}
	state.append("# started")

	lines := state.snapshot()
	lines[0] = "mutated"

	assert.Equal(t, "# started", state.snapshot()[0])
}

func TestShellContentsExplainsHowToStart(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	container := &commands.Container{ID: "abc", Name: "web"}

	contents := gui.shellContents(container)
	assert.Contains(t, contents, "# shell web")
	assert.Contains(t, contents, gui.Tr.ShellNoSession)
}

func TestShellContentsRendersBufferedOutput(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	container := &commands.Container{ID: "abc", Name: "web"}

	state := gui.shellStateFor(container.ID)
	state.append("# /bin/sh started", "$ id", "uid=0(root)")
	state.markDone()
	state.append(gui.Tr.ShellExited)

	contents := gui.shellContents(container)
	assert.Equal(t, strings.Join([]string{"# /bin/sh started", "$ id", "uid=0(root)", gui.Tr.ShellExited}, "\n"), contents)
}

func TestShellTabIsNotActiveWithoutPanel(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	assert.False(t, gui.shellTabActive())
	assert.NoError(t, gui.handleShellPrompt(nil, nil))
}
