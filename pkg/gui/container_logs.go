package gui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/fatih/color"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/jesseduffield/gocui"
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

	// Buffer the stream and replace the view content in one go. Writing lines
	// straight into the view made every reload paint from the top and then jump
	// to the bottom, which looked like the log scrolling down by itself.
	logs := newLogAccumulator(maxLogLines)
	streamDone := make(chan error, 1)

	go func() {
		streamDone <- gui.writeContainerLogs(container, ctx, logs)
	}()

	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

	rendered := -1
	waitForRestart := false

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-streamDone:
			gui.flushLogs(logs, &rendered)

			if err != nil {
				gui.Log.Error(err)
				return
			}

			// The container exited: wait until it is back, then let the task
			// restart so the log stream is re-read.
			waitForRestart = true
		case <-ticker.C:
			gui.flushLogs(logs, &rendered)

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

// maxLogLines bounds the in-memory log buffer. Without a cap a long-running
// follow stream would grow until the app is closed.
const maxLogLines = 5000

// logAccumulator implements io.Writer and keeps the last maxLogLines complete
// lines plus any partial line. version changes whenever the content changes, so
// the renderer can skip identical frames.
type logAccumulator struct {
	mutex   sync.Mutex
	lines   []string
	partial string
	version int
	max     int
}

func newLogAccumulator(max int) *logAccumulator {
	return &logAccumulator{max: max}
}

func (a *logAccumulator) Write(p []byte) (int, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	parts := strings.Split(a.partial+string(p), "\n")
	a.partial = parts[len(parts)-1]
	a.lines = append(a.lines, parts[:len(parts)-1]...)

	if a.max > 0 && len(a.lines) > a.max {
		a.lines = append([]string(nil), a.lines[len(a.lines)-a.max:]...)
	}

	a.version++

	return len(p), nil
}

// snapshot returns the text to render and its version.
func (a *logAccumulator) snapshot() (string, int) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	lines := a.lines
	if a.partial != "" {
		lines = append(append([]string(nil), lines...), a.partial)
	}

	return strings.Join(lines, "\n"), a.version
}

func (gui *Gui) flushLogs(logs *logAccumulator, rendered *int) {
	content, version := logs.snapshot()
	if version == *rendered {
		return
	}

	*rendered = version

	// Autoscroll on the main view pins the origin to the last line when the
	// frame is drawn, so the whole buffer appears at once at the bottom.
	gui.g.Update(func(*gocui.Gui) error {
		if gui.Views.Main == nil {
			return nil
		}

		return gui.setViewContent(gui.Views.Main, content)
	})
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
