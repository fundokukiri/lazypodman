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

// lockedRemoval is one item whose deletion needs confirming first: the name we
// show and the characters the user has to type.
type lockedRemoval struct {
	name   string
	secret string
}

// shortID is the part of an ID the user types to confirm a deletion.
func shortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 6 {
		id = id[:6]
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
			locked = append(locked, lockedRemoval{name: ctr.Name, secret: shortID(ctr.ID)})
		}
	}

	return locked
}

func (gui *Gui) imageRemovalLocks(images ...*commands.Image) []lockedRemoval {
	locked := make([]lockedRemoval, 0, len(images))

	for _, img := range images {
		if gui.removalLocks.ImageLocked(img.ID) {
			locked = append(locked, lockedRemoval{name: img.Name, secret: shortID(img.ID)})
		}
	}

	return locked
}

// confirmLockedRemovals asks for the ID of every locked item in turn, then runs
// after. With nothing locked it runs after straight away, so an unlocked item is
// removed exactly as before.
func (gui *Gui) confirmLockedRemovals(locks []lockedRemoval, after func() error) error {
	if len(locks) == 0 {
		return after()
	}

	lock := locks[0]

	return gui.createPromptPanel(fmt.Sprintf(gui.Tr.RemovalLockPrompt, lock.name), func(g *gocui.Gui, v *gocui.View) error {
		if strings.ToLower(strings.TrimSpace(gui.trimmedContent(v))) != lock.secret {
			return gui.createErrorPanel(gui.Tr.RemovalLockWrongID)
		}

		return gui.confirmLockedRemovals(locks[1:], after)
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

	if gui.removalLocks.ToggleImage(img.ID) {
		gui.flashStatus(fmt.Sprintf(gui.Tr.RemovalLockOn, img.Name))
	} else {
		gui.flashStatus(fmt.Sprintf(gui.Tr.RemovalLockOff, img.Name))
	}

	gui.saveRemovalLock()

	return gui.Panels.Images.RerenderList()
}
