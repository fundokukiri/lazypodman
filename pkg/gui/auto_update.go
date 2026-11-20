package gui

import (
	"context"

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
			gui.clearMainView()

			mainView := gui.Views.Main
			mainView.Autoscroll = true

			writer := newLogTrimWriter(mainView, func() bool { return true }, func() bool { return ctx.Err() != nil })
			if gui.Config != nil {
				writer.maxBytes = gui.Config.UserConfig.Logs.MaxBytes
				writer.maxLineBytes = gui.Config.UserConfig.Logs.MaxLineLength
			}
			defer writer.Close()

			_, _ = writer.Write([]byte(gui.Tr.AutoUpdateHeader + "\n\n"))

			err := gui.PodmanCommand.Pods().StreamPodman(ctx, []string{"auto-update"}, func(line string) {
				_, _ = writer.Write([]byte(line + "\n"))
			})
			if err != nil && ctx.Err() == nil {
				_, _ = writer.Write([]byte("\n" + err.Error() + "\n"))
			}

			if refreshErr := gui.refreshContainersAndServices(); refreshErr != nil {
				gui.Log.Warn(refreshErr)
			}
		})
	}, nil)
}
