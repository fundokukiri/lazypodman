package gui

import (
	"context"
	"sync"

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
// child instead of leaving an orphan process.
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

		var mu sync.Mutex

		lines := []string{eventsHeader, ""}

		gui.clearMainView()
		gui.reRenderStringMain(eventsHeader)

		err := gui.PodmanCommand.Pods().StreamPodman(ctx, []string{"events"}, func(line string) {
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
	}
}

func joinLines(lines []string) string {
	out := ""

	for i, line := range lines {
		if i != 0 {
			out += "\n"
		}

		out += line
	}

	return out
}
