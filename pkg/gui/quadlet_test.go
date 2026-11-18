package gui

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setConfigHome overrides the config directory on every OS: Windows ignores
// XDG_CONFIG_HOME and reads %AppData%.
func setConfigHome(t *testing.T, dir string) {
	t.Helper()

	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("AppData", dir)
}

// setUserHome overrides the home directory on every OS: Windows ignores HOME
// and reads %USERPROFILE%.
func setUserHome(t *testing.T, dir string) {
	t.Helper()

	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

func TestQuadletDir(t *testing.T) {
	base := t.TempDir()
	setConfigHome(t, base)

	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	gui.Config.UserConfig.Gui.QuadletDir = ""

	assert.Equal(t, filepath.Join(base, "containers", "systemd"), gui.quadletDir())
}

func TestQuadletDirExpandsHomeAndKeepsAbsolute(t *testing.T) {
	home := t.TempDir()
	setUserHome(t, home)

	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	gui.Config.UserConfig.Gui.QuadletDir = "~/units"
	assert.Equal(t, filepath.Join(home, "units"), gui.quadletDir())

	gui.Config.UserConfig.Gui.QuadletDir = "/srv/units"
	assert.Equal(t, "/srv/units", gui.quadletDir())
}

func TestQuadletFilePathIsSafe(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	gui.Config.UserConfig.Gui.QuadletDir = "/srv/units"

	assert.Equal(t, filepath.Join("/srv/units", "web.container"), gui.quadletFilePath("web"))
	path := gui.quadletFilePath("../../etc/passwd")
	assert.Equal(t, filepath.Dir(gui.quadletFilePath("web")), filepath.Dir(path), "the name must stay a single segment")
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
