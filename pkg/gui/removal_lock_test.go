package gui

import (
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestShortID(t *testing.T) {
	assert.Equal(t, "abcd", shortID("abcdef0123456789"))
	assert.Equal(t, "abcd", shortID("sha256:ABCDEF0123456789"))
	assert.Equal(t, "abc", shortID("abc"))
}

// The prompt prints the longer id so the 4 characters to type are in sight.
func TestDisplayID(t *testing.T) {
	assert.Equal(t, "abcdef012345", displayID("ABCDEF0123456789"))
	assert.Equal(t, "abcdef012345", displayID("sha256:abcdef0123456789"))
	assert.Equal(t, "abc", displayID("abc"))
}

func TestConfirmLockedRemovalsRunsStraightThroughWhenNothingIsLocked(t *testing.T) {
	gui := &Gui{removalLocks: &config.RemovalLock{}}
	ran := false

	if err := gui.confirmLockedRemovals(gui.containerRemovalLocks([]*commands.Container{{Name: "web"}}), func() error {
		ran = true

		return nil
	}); err != nil {
		t.Fatal(err)
	}
	assert.True(t, ran)
}

func TestContainerRemovalLocksOnlyIncludesLockedContainers(t *testing.T) {
	locked := &commands.Container{Name: "db"}
	locked.ID = "abcdef0123456789"
	other := &commands.Container{Name: "web"}
	other.ID = "123456789abcdef0"

	gui := &Gui{removalLocks: &config.RemovalLock{Containers: []string{locked.ID}}}

	assert.Equal(t, []lockedRemoval{{name: "db", display: "abcdef012345", secret: "abcd"}}, gui.containerRemovalLocks([]*commands.Container{locked, other}))
	assert.Equal(t, removalLockHint, gui.containerLockPrefix(locked.ID))
	assert.Equal(t, "", gui.containerLockPrefix(other.ID))
}

func TestImageRemovalLocksStripTheSHA256Prefix(t *testing.T) {
	img := &commands.Image{}
	img.ID = "sha256:ABCDEF0123456789"
	img.Name = "alpine"

	gui := &Gui{removalLocks: &config.RemovalLock{Images: []string{img.ID}}}

	assert.Equal(t, []lockedRemoval{{name: "alpine", display: "abcdef012345", secret: "abcd"}}, gui.imageRemovalLocks(img))
}
