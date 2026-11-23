package gui

import (
	"fmt"
	"strings"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/jesseduffield/gocui"
)

// removalLockHint is the prefix that marks a locked row in the containers and
// images panels. It is text rather than an icon so the table keeps its width.
const removalLockHint = "[locked] "

// lockedRemoval is one item the user marks as locked: the name we show, the ID
// printed in the prompt so the answer is in sight, and the characters typed.
type lockedRemoval struct {
	name    string
	display string
	secret  string
}

// removalLockIDChars is how many leading characters of the ID the user types to
// confirm any lock change, on or off.
const removalLockIDChars = 4

// removalLockDisplayChars is how much of the ID the prompt prints. It is the
// short id every podman listing shows, so the characters to type are on screen.
const removalLockDisplayChars = 12

// shortID is the part of an ID the user types to confirm a lock change.
func shortID(id string) string {
	return trimID(id, removalLockIDChars)
}

// displayID is the part of an ID the prompt shows, longer than the secret so
// reading it is easy without printing the whole 64 characters in a title.
func displayID(id string) string {
	return trimID(id, removalLockDisplayChars)
}

func trimID(id string, chars int) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > chars {
		id = id[:chars]
	}

	return strings.ToLower(id)
}

func (gui *Gui) containerLockPrefix(id string) string {
	if gui.removalLocks.ContainerLocked(id) {
		return removalLockHint
	}

	return ""
}

func (gui *Gui) imageLockPrefix(id string) string {
	if gui.removalLocks.ImageLocked(id) {
		return removalLockHint
	}

	return ""
}

func (gui *Gui) containerRemovalLocks(containers []*commands.Container) []lockedRemoval {
	locked := make([]lockedRemoval, 0, len(containers))

	for _, ctr := range containers {
		if gui.removalLocks.ContainerLocked(ctr.ID) {
			locked = append(locked, newLockedRemoval(ctr.Name, ctr.ID))
		}
	}

	return locked
}

func (gui *Gui) imageRemovalLocks(images ...*commands.Image) []lockedRemoval {
	locked := make([]lockedRemoval, 0, len(images))

	for _, img := range images {
		if gui.removalLocks.ImageLocked(img.ID) {
			locked = append(locked, newLockedRemoval(img.Name, img.ID))
		}
	}

	return locked
}

// newLockedRemoval builds the prompt data for one item.
func newLockedRemoval(name, id string) lockedRemoval {
	return lockedRemoval{name: name, display: displayID(id), secret: shortID(id)}
}

// confirmLockedRemovals asks for the ID of every locked item in turn, then runs
// after. With nothing locked it runs after straight away, so an unlocked item is
// removed exactly as before.
func (gui *Gui) confirmLockedRemovals(locks []lockedRemoval, after func() error) error {
	if len(locks) == 0 {
		return after()
	}

	lock := locks[0]

	return gui.askLockID(gui.Tr.RemovalLockPrompt, lock, func() error {
		return gui.confirmLockedRemovals(locks[1:], after)
	})
}

// askLockID asks for the first characters of an item's ID before running after,
// so neither locking nor unlocking happens on a mis-pressed key. The prompt
// prints the ID (display), the name and the number of characters to type.
func (gui *Gui) askLockID(prompt string, lock lockedRemoval, after func() error) error {
	title := fmt.Sprintf(prompt, lock.display, lock.name, removalLockIDChars)

	return gui.createPromptPanel(title, func(g *gocui.Gui, v *gocui.View) error {
		if strings.ToLower(strings.TrimSpace(gui.trimmedContent(v))) != lock.secret {
			return gui.createErrorPanel(gui.Tr.RemovalLockWrongID)
		}

		return after()
	})
}

func (gui *Gui) saveRemovalLock() {
	if err := gui.Config.SaveRemovalLock(gui.removalLocks); err != nil {
		gui.Log.Error(err)
	}
}

func (gui *Gui) handleToggleContainerLock(g *gocui.Gui, v *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	if gui.removalLocks.ContainerLocked(ctr.ID) {
		return gui.askLockID(gui.Tr.RemovalUnlockPrompt, newLockedRemoval(ctr.Name, ctr.ID), func() error {
			return gui.toggleContainerLock(ctr)
		})
	}

	return gui.toggleContainerLock(ctr)
}

// toggleContainerLock flips the lock and saves the list.
func (gui *Gui) toggleContainerLock(ctr *commands.Container) error {
	if gui.removalLocks.ToggleContainer(ctr.ID) {
		gui.flashStatus(fmt.Sprintf(gui.Tr.RemovalLockOn, ctr.Name))
	} else {
		gui.flashStatus(fmt.Sprintf(gui.Tr.RemovalLockOff, ctr.Name))
	}

	gui.saveRemovalLock()

	return gui.Panels.Containers.RerenderList()
}

func (gui *Gui) handleToggleImageLock(g *gocui.Gui, v *gocui.View) error {
	img, err := gui.Panels.Images.GetSelectedItem()
	if err != nil {
		return nil
	}

	if gui.removalLocks.ImageLocked(img.ID) {
		return gui.askLockID(gui.Tr.RemovalUnlockPrompt, newLockedRemoval(img.Name, img.ID), func() error {
			return gui.toggleImageLock(img)
		})
	}

	return gui.toggleImageLock(img)
}

// toggleImageLock flips the lock and saves the list.
func (gui *Gui) toggleImageLock(img *commands.Image) error {
	if gui.removalLocks.ToggleImage(img.ID) {
		gui.flashStatus(fmt.Sprintf(gui.Tr.RemovalLockOn, img.Name))
	} else {
		gui.flashStatus(fmt.Sprintf(gui.Tr.RemovalLockOff, img.Name))
	}

	gui.saveRemovalLock()

	return gui.Panels.Images.RerenderList()
}
