package gui

import (
	"context"
	"sync"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/jesseduffield/gocui"
)

// handleAutoUpdate runs `podman auto-update` after a confirmation and streams its
// output. It is a bulk command entry because auto-update acts on every container
// that has the auto-update label, not on the current selection.
func (gui *Gui) handleAutoUpdate() error {
	if gui.PodmanCommand == nil || gui.PodmanCommand.Pods() == nil ||
		gui.PodmanCommand.Capabilities().CLI.State == commands.CapabilityUnavailable {
		gui.flashStatus(gui.eventsUnavailable())

		return nil
	}

	return gui.createConfirmationPanel(gui.Tr.Confirm, gui.Tr.ConfirmAutoUpdate, func(g *gocui.Gui, v *gocui.View) error {
		return gui.QueueTask(func(ctx context.Context) {
			var mu sync.Mutex

			lines := []string{gui.Tr.AutoUpdateHeader, ""}

			gui.clearMainView()
			gui.reRenderStringMain(gui.Tr.AutoUpdateHeader)

			err := gui.PodmanCommand.Pods().StreamPodman(ctx, []string{"auto-update"}, func(line string) {
				mu.Lock()
				lines = append(lines, line)
				contents := joinLines(lines)
				mu.Unlock()

				gui.reRenderStringMain(contents)
			})
			if err != nil && ctx.Err() == nil {
				mu.Lock()
				lines = append(lines, "", err.Error())
				contents := joinLines(lines)
				mu.Unlock()

				gui.reRenderStringMain(contents)
			}

			if refreshErr := gui.refreshContainersAndServices(); refreshErr != nil {
				gui.Log.Warn(refreshErr)
			}
		})
	}, nil)
}
