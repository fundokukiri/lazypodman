package gui

import (
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/stretchr/testify/assert"
)

func TestImagePullRefPrefersRepoTag(t *testing.T) {
	img := &commands.Image{Name: "nginx", Tag: "alpine", ID: "sha256:abc"}
	img.Image.RepoTags = []string{"docker.io/library/nginx:alpine"}

	assert.Equal(t, "docker.io/library/nginx:alpine", imagePullRef(img))
}

func TestImagePullRefFallsBackToNameAndTag(t *testing.T) {
	img := &commands.Image{Name: "nginx", Tag: "latest", ID: "sha256:abc"}
	img.Image.RepoTags = []string{"<none>:<none>"}

	assert.Equal(t, "nginx:latest", imagePullRef(img))
}

func TestImagePullRefUsesNameWithoutTag(t *testing.T) {
	img := &commands.Image{Name: "nginx", ID: "sha256:abc"}

	assert.Equal(t, "nginx", imagePullRef(img))
}

func TestImagePullRefReturnsEmptyForUntaggedImage(t *testing.T) {
	img := &commands.Image{Name: "none", ID: "sha256:abc"}
	img.Image.RepoTags = []string{"<none>:<none>"}

	assert.Empty(t, imagePullRef(img))
}

func TestImagesUpdateBindingIsAvailable(t *testing.T) {
	gui := &Gui{Tr: translations()}

	found := false
	for _, binding := range gui.staticKeybindings() {
		if binding.ViewName == "images" && binding.Key == 'u' {
			found = true
		}
	}

	assert.True(t, found, "images panel should bind u to update an image")
}
