package gui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuadletUnitNameUsesTheGeneratorServiceName(t *testing.T) {
	gui := &Gui{}

	assert.Equal(t, "web.service", gui.quadletUnitName(filepath.Join("/units", "web.container")))
	assert.Equal(t, "stack.service", gui.quadletUnitName(filepath.Join("/units", "stack.pod")))
}

func TestQuadletFilesListsOnlyQuadletExtensions(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"web.container", "db.pod", "notes.txt"} {
		assert.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600))
	}
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	gui.Config.UserConfig.Gui.QuadletDir = dir

	files, err := gui.quadletFiles()
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, filepath.Base(files[0]), "db.pod")
	assert.Contains(t, filepath.Base(files[1]), "web.container")
}
