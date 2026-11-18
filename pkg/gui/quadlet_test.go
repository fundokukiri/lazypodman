package gui

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuadletDir(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)

	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	gui.Config.UserConfig.Gui.QuadletDir = ""

	assert.Equal(t, filepath.Join(base, "containers", "systemd"), gui.quadletDir())
}

func TestQuadletDirExpandsHomeAndKeepsAbsolute(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	gui.Config.UserConfig.Gui.QuadletDir = "~/units"
	assert.Equal(t, filepath.Join(home, "units"), gui.quadletDir())

	gui.Config.UserConfig.Gui.QuadletDir = "/srv/units"
	assert.Equal(t, "/srv/units", gui.quadletDir())
}

func TestQuadletFilePathIsSafe(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	gui.Config.UserConfig.Gui.QuadletDir = "/srv/units"

	assert.Equal(t, "/srv/units/web.container", gui.quadletFilePath("web"))
	path := gui.quadletFilePath("../../etc/passwd")
	assert.Equal(t, "/srv/units", filepath.Dir(path), "the name must stay a single segment")
	assert.Contains(t, filepath.Base(path), "etc-passwd")
	assert.NotContains(t, filepath.Base(path), "/")
}

func TestQuadletSavePrompt(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	prompt := gui.quadletSavePrompt("/srv/units/web.container", false)
	assert.Contains(t, prompt, "/srv/units/web.container")
	assert.Contains(t, prompt, "plaintext")

	assert.Contains(t, gui.quadletSavePrompt("/srv/units/web.container", true), "File exists")
}
