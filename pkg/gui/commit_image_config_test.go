package gui

import (
	"strings"
	"testing"
	"time"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/stretchr/testify/assert"
)

func TestCommitImageDefaultTagUsesContainerName(t *testing.T) {
	now := time.Date(2026, 11, 18, 14, 32, 5, 0, time.UTC)

	assert.Equal(t, "web_20261118_143205", defaultCommitImageName("web", now))
	assert.Equal(t, "container_20261118_143205", defaultCommitImageName("  ", now))
}

func TestCommitImageBuildsTheFullCommand(t *testing.T) {
	state := newCommitImageState(&commands.Container{Name: "web"})
	state.text["image"] = "local/web:snap"
	state.text["author"] = "Fundoku"
	state.text["message"] = "snapshot"
	state.lists["changes"] = []string{"ENV A=1", "CMD [\"/app\"]"}
	state.toggles["pause"] = false
	state.toggles["squash"] = true

	args := state.commitArgv("web", "--include-volumes")

	assert.Equal(t, []string{
		"commit",
		"--author", "Fundoku",
		"--message", "snapshot",
		"--change", "ENV A=1",
		"--change", "CMD [\"/app\"]",
		"--squash",
		"--pause=false",
		"--include-volumes",
		"--", "web", "local/web:snap",
	}, args)
}

func TestCommitImagePauseIsOnByDefault(t *testing.T) {
	state := newCommitImageState(&commands.Container{Name: "web"})
	state.text["image"] = "web:snap"

	args := state.commitArgv("web", "")

	assert.Equal(t, []string{"commit", "--", "web", "web:snap"}, args)
}

func TestCommitImageOmitsEmptyTag(t *testing.T) {
	state := newCommitImageState(&commands.Container{Name: "web"})
	state.text["image"] = "  "

	args := state.commitArgv("web", "")

	assert.Equal(t, []string{"commit", "--", "web"}, args)
}

func TestCommitImageLabelsCoverEveryField(t *testing.T) {
	gui := &Gui{Tr: translations()}

	for _, field := range commitImageFields {
		assert.NotEmpty(t, gui.runLabel(field.Key), "missing label for %q", field.Key)
	}
	for _, group := range commitImageGroups {
		assert.NotEmpty(t, gui.runLabel(group.Key), "missing label for %q", group.Key)
		for _, field := range group.Fields {
			assert.NotEmpty(t, gui.runLabel(field.Key), "missing label for %q", field.Key)
		}
	}
}

func TestCommitImageBindingIsAvailableOnContainers(t *testing.T) {
	gui := &Gui{Tr: translations()}

	found := false
	for _, binding := range gui.staticKeybindings() {
		if binding.ViewName == "containers" && binding.Key == 'C' {
			found = true
		}
	}

	assert.True(t, found, "containers panel should bind C to commit to an image")
}

func TestCommitContainerRefFallsBackToID(t *testing.T) {
	assert.Equal(t, "web", commitContainerRef(&commands.Container{Name: "web", ID: "abc"}))
	assert.Equal(t, "abc", commitContainerRef(&commands.Container{ID: "abc"}))
}

func TestDefaultCommitImageNameIsSafe(t *testing.T) {
	name := defaultCommitImageName("my /web", time.Now())

	assert.True(t, strings.HasPrefix(name, "my--web_"))
}
