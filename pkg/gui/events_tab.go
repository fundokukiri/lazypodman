package gui

import (
	"context"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
)

// eventsHeader is the first line of the Events tab; it names the exact command so
// the output is never mistaken for something else.
const eventsHeader = "# podman events"

// eventsUnavailable is what the tab shows when the CLI cannot be used. It avoids
// spawning a process that would only fail.
func (gui *Gui) eventsUnavailable() string {
	caps := gui.PodmanCommand.Capabilities()
	if caps.CLI.State == commands.CapabilityUnavailable {
		return gui.Tr.CliOff + " " + caps.CLI.Reason
	}

	return gui.Tr.CliOff + " podman CLI not found"
}

// renderContainerEvents streams `podman events` verbatim into the main panel.
// The stream is bound to the task context, so switching tab or panel kills the
// child instead of leaving an orphan process. The trim writer bounds a tab that
// can stay open for hours instead of keeping every event in memory.
func (gui *Gui) renderContainerEvents(_ *commands.Container) tasks.TaskFunc {
	return func(ctx context.Context) {
		if gui.PodmanCommand == nil || gui.PodmanCommand.Pods() == nil {
			gui.RenderStringMain(gui.eventsUnavailable())

			return
		}

		caps := gui.PodmanCommand.Capabilities()
		if caps.CLI.State == commands.CapabilityUnavailable {
			gui.RenderStringMain(gui.eventsUnavailable())

			return
		}

		gui.clearMainView()

		mainView := gui.Views.Main
		mainView.Autoscroll = true
		writer := newLogTrimWriter(mainView, func() bool { return true }, func() bool { return ctx.Err() != nil })
		if gui.Config != nil {
			writer.maxBytes = gui.Config.UserConfig.Logs.MaxBytes
			writer.maxLineBytes = gui.Config.UserConfig.Logs.MaxLineLength
		}
		defer writer.Close()

		_, _ = writer.Write([]byte(eventsHeader + "\n\n"))

		err := gui.PodmanCommand.Pods().StreamPodman(ctx, []string{"events"}, func(line string) {
			_, _ = writer.Write([]byte(line + "\n"))
		})
		if err != nil && ctx.Err() == nil {
			_, _ = writer.Write([]byte("\n" + err.Error() + "\n"))
		}
	}
}
