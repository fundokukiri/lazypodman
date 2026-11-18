package gui

import (
	"testing"

	"github.com/docker/docker/api/types/image"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/stretchr/testify/assert"
)

func TestRunImageRefPrefersRepoTag(t *testing.T) {
	img := &commands.Image{Name: "nginx", Tag: "alpine", ID: "sha256:abc"}
	img.Image.RepoTags = []string{"docker.io/library/nginx:alpine"}

	assert.Equal(t, "docker.io/library/nginx:alpine", runImageRef(img))
}

func TestRunImageRefIgnoresNoneTag(t *testing.T) {
	img := &commands.Image{Name: "nginx", Tag: "alpine", ID: "sha256:abc"}
	img.Image.RepoTags = []string{"<none>:<none>"}

	assert.Equal(t, "nginx:alpine", runImageRef(img))
}

func TestRunImageRefFallsBackToID(t *testing.T) {
	img := &commands.Image{Name: "none", Tag: "", ID: "sha256:abc"}
	img.Image.RepoTags = []string{"<none>:<none>"}

	assert.Equal(t, "sha256:abc", runImageRef(img))
}

func TestRunContainerPreviewQuotesArguments(t *testing.T) {
	command := formatRunCommand("podman", []string{"run", "-d", "--name", "my web", "nginx:alpine", "sh", "-c", "echo 'hi'"})

	assert.Equal(t, `podman run -d --name 'my web' nginx:alpine sh -c 'echo '\''hi'\'''`, command)
}

func TestRunContainerPreviewKeepsPlainArgumentsPlain(t *testing.T) {
	command := formatRunCommand("podman", []string{"run", "-p", "8080:80", "nginx:alpine"})

	assert.Equal(t, "podman run -p 8080:80 nginx:alpine", command)
}

func TestRunContainerLabelsCoverEveryField(t *testing.T) {
	gui := &Gui{Tr: translations()}

	assert.NotEmpty(t, gui.runLabel("name"))
	assert.Equal(t, "Name", gui.runLabel("name"))
	assert.Equal(t, "Network & DNS", gui.runLabel("network_group"))
	assert.Equal(t, "Health command", gui.runLabel("health_cmd"))
}

func TestImagesRunBindingComesBeforeTheOtherImageActions(t *testing.T) {
	gui := &Gui{Tr: translations()}

	var keys []interface{}
	for _, binding := range gui.staticKeybindings() {
		if binding.ViewName == "images" {
			keys = append(keys, binding.Key)
		}
	}

	assert.GreaterOrEqual(t, len(keys), 1)
	assert.Equal(t, rune('r'), keys[0])
}

func TestRunValueLabelShowsDefaultsAndLists(t *testing.T) {
	gui := &Gui{Tr: translations()}
	state := newRunContainerState("alpine")

	assert.Equal(t, "default", gui.runValueLabel(state, runField{Key: "name", Kind: runTextField}))
	assert.Equal(t, "on", gui.runValueLabel(state, runField{Key: "detach", Kind: runToggleField}))
	assert.Equal(t, "none", gui.runValueLabel(state, runField{Key: "ports", Kind: runListField}))

	state.addListValue("ports", "8080:80")
	assert.Equal(t, "1 items", gui.runValueLabel(state, runField{Key: "ports", Kind: runListField}))
}

func TestDockerImageSummaryRepoTagsTypeIsUsed(t *testing.T) {
	// Keeps the image fixture close to the real type so runImageRef cannot drift.
	img := &commands.Image{Image: image.Summary{RepoTags: []string{"alpine:latest"}}}

	assert.Equal(t, "alpine:latest", runImageRef(img))
}
