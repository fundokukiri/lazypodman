package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildImageDefaults(t *testing.T) {
	state := newBuildImageState()

	assert.Equal(t, []string{"build", "."}, state.buildArgv(""))
	assert.Equal(t, ".", guiValueLabel("context"))
}

func TestBuildImageBuildsTheFullCommand(t *testing.T) {
	state := newBuildImageState()
	state.text["context"] = "./app"
	state.text["containerfile"] = "Containerfile.dev"
	state.text["target"] = "runtime"
	state.text["platform"] = "linux/arm64"
	state.lists["tags"] = []string{"example/app:dev", "example/app:latest"}
	state.lists["build_arg"] = []string{"VERSION=1.2.3"}
	state.lists["build_label"] = []string{"org.example=build"}
	state.toggles["no_cache"] = true
	state.toggles["squash"] = true

	args := state.buildArgv("--pull=always")

	assert.Equal(t, []string{
		"build",
		"--file", "Containerfile.dev",
		"--tag", "example/app:dev",
		"--tag", "example/app:latest",
		"--target", "runtime",
		"--platform", "linux/arm64",
		"--no-cache",
		"--squash",
		"--build-arg", "VERSION=1.2.3",
		"--label", "org.example=build",
		"--pull=always",
		"./app",
	}, args)
}

func TestBuildImageOmitsEmptyValues(t *testing.T) {
	state := newBuildImageState()
	state.text["containerfile"] = "  "
	state.lists["tags"] = []string{"", "  "}
	state.lists["build_arg"] = []string{"  "}

	assert.Equal(t, []string{"build", "."}, state.buildArgv("  "))
}

func TestBuildImageSchemaKeysAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, field := range buildImageFields {
		assert.False(t, seen[field.Key], "duplicate key %q", field.Key)
		seen[field.Key] = true
	}
	for _, group := range buildImageGroups {
		for _, field := range group.Fields {
			assert.False(t, seen[field.Key], "duplicate key %q", field.Key)
			seen[field.Key] = true
		}
	}
}

func TestBuildImageLabelsCoverEveryField(t *testing.T) {
	gui := &Gui{Tr: translations()}

	for _, field := range buildImageFields {
		assert.NotEmpty(t, gui.runLabel(field.Key), "missing label for %q", field.Key)
	}
	for _, group := range buildImageGroups {
		assert.NotEmpty(t, gui.runLabel(group.Key), "missing label for %q", group.Key)
		for _, field := range group.Fields {
			assert.NotEmpty(t, gui.runLabel(field.Key), "missing label for %q", field.Key)
		}
	}
}

func TestBuildImageBindingIsGlobal(t *testing.T) {
	gui := &Gui{Tr: translations()}

	found := false
	for _, binding := range gui.staticKeybindings() {
		if binding.ViewName == "" && binding.Key == 'B' {
			found = true
		}
	}

	assert.True(t, found, "B should open the build menu from any panel")
}

func guiValueLabel(key string) string {
	gui := &Gui{Tr: translations()}
	return gui.runValueLabel(newBuildImageState(), buildFieldByKey(key))
}
