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
)

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

	// Stream straight into the view (as before) but collect the first short burst
	// so the tab opens already scrolled to the bottom, and trim old lines so a
	// long follow session cannot grow memory without bound.
	logs := newLogStreamWriter(gui.Views.Main)
	streamDone := make(chan error, 1)

	go func() {
		streamDone <- gui.writeContainerLogs(container, ctx, logs)
	}()

	startedAt := time.Now()
	waitForRestart := false

	ticker := time.NewTicker(time.Millisecond * 50)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logs.FlushInitial()
			logs.TrimIfNeeded()

			return
		case err := <-streamDone:
			logs.FlushInitial()
			logs.TrimIfNeeded()

			if err != nil {
				gui.Log.Error(err)
				return
			}

			// The container exited: wait until it is back, then let the task
			// restart so the log stream is re-read.
			waitForRestart = true
		case <-ticker.C:
			if logs.Buffering() && time.Since(startedAt) >= logFirstPaintWindow {
				logs.FlushInitial()
			}

			logs.TrimIfNeeded()

			if !waitForRestart {
				continue
			}

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
		Tail:       gui.Config.UserConfig.Logs.Tail,
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
