package gui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/fatih/color"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/jesseduffield/gocui"
)

// containerLogTail returns "" (all history) when the user pressed L for this
// container, otherwise the configured tail (200 by default).
func (gui *Gui) containerLogTail(container *commands.Container) string {
	gui.fullLogsMutex.Lock()
	defer gui.fullLogsMutex.Unlock()

	if gui.fullLogs[container.ID] {
		return ""
	}

	return gui.Config.UserConfig.Logs.Tail
}

func (gui *Gui) logsFullFor(container *commands.Container) bool {
	gui.fullLogsMutex.Lock()
	defer gui.fullLogsMutex.Unlock()

	return gui.fullLogs[container.ID]
}

// toggleFullLogs switches one container between the default tail and all logs
// and restarts the log task so the new range is fetched.
func (gui *Gui) toggleFullLogs(container *commands.Container) {
	gui.fullLogsMutex.Lock()
	if gui.fullLogs == nil {
		gui.fullLogs = map[string]bool{}
	}
	gui.fullLogs[container.ID] = !gui.fullLogs[container.ID]
	gui.fullLogsMutex.Unlock()

	// Force the log task to restart with the new tail.
	if gui.State.Panels != nil && gui.State.Panels.Main != nil {
		gui.State.Panels.Main.ObjectKey = ""
	}
}

// logsTabActive is true when the Containers panel shows its Logs tab, which is
// the only place the full-history toggle applies.
func (gui *Gui) logsTabActive() bool {
	if gui.Panels.Containers == nil || gui.Panels.Containers.ContextState == nil {
		return false
	}

	if gui.currentSideViewName() != "containers" {
		return false
	}

	return gui.Panels.Containers.ContextState.GetCurrentMainTab().Key == "logs"
}

func (gui *Gui) handleToggleFullLogs(g *gocui.Gui, v *gocui.View) error {
	if !gui.logsTabActive() {
		return nil
	}

	container, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	gui.toggleFullLogs(container)

	if gui.logsFullFor(container) {
		gui.flashStatus(gui.Tr.AllLogs)
	} else {
		gui.flashStatus(gui.Tr.RecentLogs)
	}

	// Reload just this tab. The forced key makes HandleSelect queue a fresh log
	// task, and that task waits for the old stream to stop before it starts.
	return gui.Panels.Containers.HandleSelect()
}

// logsToggleDescription is "load all logs", or "load recent logs" once the full
// history is already loaded.
func (gui *Gui) logsToggleDescription() string {
	if gui.Panels.Containers != nil {
		if container, err := gui.Panels.Containers.GetSelectedItem(); err == nil && gui.logsFullFor(container) {
			return gui.Tr.RecentLogs
		}
	}

	return gui.Tr.AllLogs
}

// logsOptionsMap is the bottom-line help for the Logs tab.
func (gui *Gui) logsOptionsMap() map[string]string {
	options := map[string]string{
		"PgUp/PgDn": gui.Tr.Scroll,
		"ctrl+l":    gui.Tr.AllLogs,
	}

	options["ctrl+l"] = gui.logsToggleDescription()

	return options
}

func (gui *Gui) renderContainerLogsToMain(container *commands.Container) tasks.TaskFunc {
	return gui.NewTickerTask(TickerTaskOpts{
		Func: func(ctx context.Context, notifyStopped chan struct{}) {
			gui.renderContainerLogsToMainAux(container, ctx, notifyStopped)
		},
		Duration: time.Millisecond * 200,
		// TODO: see why this isn't working (when switching from Top tab to Logs tab in the services panel, the tops tab's content isn't removed)
		Before:     func(ctx context.Context) { gui.clearMainView() },
		Wrap:       gui.Config.UserConfig.Gui.WrapMainPanel,
		Autoscroll: true,
	})
}

func (gui *Gui) renderContainerLogsToMainAux(container *commands.Container, ctx context.Context, notifyStopped chan struct{}) {
	gui.clearMainView()
	defer func() {
		notifyStopped <- struct{}{}
	}()

	mainView := gui.Views.Main

	// Keep the old streaming behaviour (lines appear immediately) but bound the
	// view's memory: follow mode would otherwise keep every line until exit.
	if err := gui.writeContainerLogs(container, ctx, newLogTrimWriter(mainView)); err != nil {
		gui.Log.Error(err)
	}

	// if we are here because the task has been stopped, we should return
	// if we are here then the container must have exited, meaning we should wait until it's back again before
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := container.Inspect()
			if err != nil {
				// if we get an error, then the container has probably been removed so we'll get out of here
				gui.Log.Error(err)
				return
			}
			if result.State.Running {
				return
			}
		}
	}
}

func (gui *Gui) renderLogsToStdout(container *commands.Container) {
	stop := make(chan os.Signal, 1)
	defer signal.Stop(stop)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		signal.Notify(stop, os.Interrupt)
		<-stop
		cancel()
	}()

	if err := gui.g.Suspend(); err != nil {
		gui.Log.Error(err)
		return
	}

	defer func() {
		if err := gui.g.Resume(); err != nil {
			gui.Log.Error(err)
		}
	}()

	if err := gui.writeContainerLogs(container, ctx, os.Stdout); err != nil {
		gui.Log.Error(err)
		return
	}

	gui.promptToReturn()
}

func (gui *Gui) promptToReturn() {
	if !gui.Config.UserConfig.Gui.ReturnImmediately {
		fmt.Fprintf(os.Stdout, "\n\n%s", utils.ColoredString(gui.Tr.PressEnterToReturn, color.FgGreen))

		// wait for enter press
		if _, err := fmt.Scanln(); err != nil {
			gui.Log.Error(err)
		}
	}
}

func (gui *Gui) writeContainerLogs(ctr *commands.Container, ctx context.Context, writer io.Writer) error {
	readCloser, err := gui.PodmanCommand.Client.ContainerLogs(ctx, ctr.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: gui.Config.UserConfig.Logs.Timestamps,
		Since:      gui.Config.UserConfig.Logs.Since,
		Tail:       gui.containerLogTail(ctr),
		Follow:     true,
	})
	if err != nil {
		gui.Log.Error(err)
		return err
	}
	defer readCloser.Close()

	if !ctr.DetailsLoaded() {
		// loop until the details load or context is cancelled, using timer
		ticker := time.NewTicker(time.Millisecond * 100)
		defer ticker.Stop()
	outer:
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				if ctr.DetailsLoaded() {
					break outer
				}
			}
		}
	}

	if ctr.Details.Config.Tty {
		_, err = io.Copy(writer, readCloser)
		if err != nil {
			return err
		}
	} else {
		_, err = stdcopy.StdCopy(writer, writer, readCloser)
		if err != nil {
			return err
		}
	}

	return nil
}
