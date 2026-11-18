package gui

import (
	"fmt"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/jesseduffield/gocui"
)

// toggleMarkedContainer marks or unmarks one container. Marks are keyed by ID so
// they survive the list being rebuilt by a refresh.
func (gui *Gui) toggleMarkedContainer(id string) bool {
	gui.markedContainersLock.Lock()
	defer gui.markedContainersLock.Unlock()

	if gui.markedContainerIDs == nil {
		gui.markedContainerIDs = map[string]bool{}
	}

	if gui.markedContainerIDs[id] {
		delete(gui.markedContainerIDs, id)

		return false
	}

	gui.markedContainerIDs[id] = true

	return true
}

func (gui *Gui) markedContainerCount() int {
	gui.markedContainersLock.Lock()
	defer gui.markedContainersLock.Unlock()

	return len(gui.markedContainerIDs)
}

func (gui *Gui) isContainerMarked(id string) bool {
	gui.markedContainersLock.Lock()
	defer gui.markedContainersLock.Unlock()

	return gui.markedContainerIDs[id]
}

func (gui *Gui) clearMarkedContainers() {
	gui.markedContainersLock.Lock()
	defer gui.markedContainersLock.Unlock()

	gui.markedContainerIDs = nil
}

// markedContainers returns the marked items that still exist, preserving the list
// order. Pure so the selection logic is testable without a panel.
func markedContainers(items []*commands.Container, marked map[string]bool) []*commands.Container {
	selected := make([]*commands.Container, 0, len(marked))

	for _, container := range items {
		if marked[container.ID] {
			selected = append(selected, container)
		}
	}

	return selected
}

// containersToActOn is the multi-select rule used by the lifecycle keys: when
// anything is marked, act on every marked container; otherwise on the current one.
func (gui *Gui) containersToActOn() []*commands.Container {
	items := gui.Panels.Containers.List.GetAllItems()

	gui.markedContainersLock.Lock()
	marked := make(map[string]bool, len(gui.markedContainerIDs))
	for id := range gui.markedContainerIDs {
		marked[id] = true
	}
	gui.markedContainersLock.Unlock()

	if len(marked) == 0 {
		selected, err := gui.Panels.Containers.GetSelectedItem()
		if err != nil {
			return nil
		}

		return []*commands.Container{selected}
	}

	return markedContainers(items, marked)
}

func (gui *Gui) handleContainersToggleMark(g *gocui.Gui, v *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	marked := gui.toggleMarkedContainer(ctr.ID)

	if err := gui.Panels.Containers.RerenderList(); err != nil {
		gui.Log.Error(err)
	}

	if marked {
		gui.flashStatus(fmt.Sprintf(gui.Tr.MarkedCount, gui.markedContainerCount()))
	} else if count := gui.markedContainerCount(); count > 0 {
		gui.flashStatus(fmt.Sprintf(gui.Tr.MarkedCount, count))
	} else {
		gui.flashStatus(gui.Tr.MarksCleared)
	}

	return nil
}

// containerMarkPrefix is the list marker for a marked container, empty otherwise.
func (gui *Gui) containerMarkPrefix(id string) string {
	if gui.isContainerMarked(id) {
		return "● "
	}

	return ""
}
