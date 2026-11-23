package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemovalLockSaveAndLoad(t *testing.T) {
	appConfig := &AppConfig{ConfigDir: filepath.Join(t.TempDir(), "cfg")}

	lock, err := appConfig.LoadRemovalLock()
	if err != nil {
		t.Fatal(err)
	}
	assert.Empty(t, lock.Containers)
	assert.Empty(t, lock.Images)

	assert.True(t, lock.ToggleContainer("abc123"))
	assert.True(t, lock.ToggleImage("sha256:def456"))
	if err := appConfig.SaveRemovalLock(lock); err != nil {
		t.Fatal(err)
	}

	reloaded, err := appConfig.LoadRemovalLock()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, reloaded.ContainerLocked("abc123"))
	assert.True(t, reloaded.ImageLocked("sha256:def456"))

	assert.False(t, reloaded.ToggleContainer("abc123"))
	assert.False(t, reloaded.ToggleImage("sha256:def456"))
	assert.False(t, reloaded.ContainerLocked("abc123"))
	assert.False(t, reloaded.ImageLocked("sha256:def456"))
}
