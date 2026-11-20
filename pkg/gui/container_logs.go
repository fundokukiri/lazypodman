package gui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
		return gui.fullLogTail()
	}

	return gui.Config.UserConfig.Logs.Tail
}

// fullLogTail caps what ctrl+l asks for. Streaming an unbounded history takes
// minutes on a busy container and scrolls the old lines out of the view before
// they can be read, so the full-history view has a line budget too
// (logs.fullTail, 50000 by default).
func (gui *Gui) fullLogTail() string {
	if lines := gui.Config.UserConfig.Logs.FullTail; lines > 0 {
		return strconv.Itoa(lines)
	}

	return ""
}

func (gui *Gui) transcriptLogsFor(container *commands.Container) bool {
	gui.transcriptLogsMutex.Lock()
	defer gui.transcriptLogsMutex.Unlock()

	return gui.transcriptLogs[container.ID]
}

func (gui *Gui) transcriptAutoDisabledFor(container *commands.Container) bool {
	gui.transcriptLogsMutex.Lock()
	defer gui.transcriptLogsMutex.Unlock()

	return gui.transcriptAutoDisabled[container.ID]
}

func (gui *Gui) toggleTranscriptLogs(container *commands.Container) {
	gui.transcriptLogsMutex.Lock()
	if gui.transcriptLogs == nil {
		gui.transcriptLogs = map[string]bool{}
	}
	if gui.transcriptAutoDisabled == nil {
		gui.transcriptAutoDisabled = map[string]bool{}
	}
	gui.transcriptLogs[container.ID] = !gui.transcriptLogs[container.ID]
	gui.transcriptAutoDisabled[container.ID] = !gui.transcriptLogs[container.ID]
	gui.transcriptLogsMutex.Unlock()

	gui.invalidateLogViewCache()

	if gui.State.Panels != nil && gui.State.Panels.Main != nil {
		gui.State.Panels.Main.ObjectKey = ""
	}
}

// enableTranscriptLogsAutomatically switches a container to transcript mode
// after its log stream showed full-screen terminal output. It respects an
// explicit switch back to plain logs.
func (gui *Gui) enableTranscriptLogsAutomatically(container *commands.Container) {
	gui.transcriptLogsMutex.Lock()
	if gui.transcriptLogs[container.ID] || gui.transcriptAutoDisabled[container.ID] {
		gui.transcriptLogsMutex.Unlock()
		return
	}
	if gui.transcriptLogs == nil {
		gui.transcriptLogs = map[string]bool{}
	}
	gui.transcriptLogs[container.ID] = true
	gui.transcriptLogsMutex.Unlock()

	gui.invalidateLogViewCache()
	// The context key is ID + state + tab, so switching mode does not change it.
	// Force the Logs tab to queue a fresh task or HandleSelect will think the
	// existing (plain) content is still current.
	if gui.State.Panels != nil && gui.State.Panels.Main != nil {
		gui.State.Panels.Main.ObjectKey = ""
	}

	gui.g.Update(func(*gocui.Gui) error {
		if !gui.logsTabActive() || gui.Panels.Containers == nil {
			return nil
		}
		selected, err := gui.Panels.Containers.GetSelectedItem()
		if err != nil || selected.ID != container.ID {
			return nil
		}

		gui.flashStatus(gui.Tr.TranscriptDetected)
		return gui.Panels.Containers.HandleSelect()
	})
}

// transcriptLogTail bounds how much history the terminal replay reads. It is
// deliberately independent of logs.tail and of the ctrl+l full-history toggle:
// the emulator only shows the final screen, so replaying more than a few hundred
// lines costs time and cannot change what is displayed. logs.transcriptTail
// overrides it; 0 or less keeps the default instead of replaying everything,
// which is what used to make ctrl+t look like it never finished.
func (gui *Gui) transcriptLogTail() string {
	replay := gui.Config.UserConfig.Logs.TranscriptTail
	if replay <= 0 {
		replay = defaultTranscriptTail
	}

	return strconv.Itoa(replay)
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

	gui.invalidateLogViewCache()

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
func (gui *Gui) transcriptToggleDescription() string {
	if gui.Panels.Containers != nil {
		if container, err := gui.Panels.Containers.GetSelectedItem(); err == nil && gui.transcriptLogsFor(container) {
			return gui.Tr.PlainLogs
		}
	}

	return gui.Tr.TranscriptLogs
}

func (gui *Gui) logsToggleDescription() string {
	if gui.Panels.Containers != nil {
		if container, err := gui.Panels.Containers.GetSelectedItem(); err == nil && gui.logsFullFor(container) {
			return gui.Tr.RecentLogs
		}
	}

	return gui.Tr.AllLogs
}

// handleLogsJumpToLatest re-enables follow mode and scrolls to the newest line.
func (gui *Gui) handleToggleTranscriptLogs(g *gocui.Gui, v *gocui.View) error {
	if !gui.logsTabActive() {
		return nil
	}

	container, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	gui.toggleTranscriptLogs(container)

	if gui.transcriptLogsFor(container) {
		gui.flashStatus(gui.Tr.TranscriptLogs)
	} else {
		gui.flashStatus(gui.Tr.PlainLogs)
	}

	return gui.Panels.Containers.HandleSelect()
}

func (gui *Gui) handleLogsJumpToLatest(g *gocui.Gui, v *gocui.View) error {
	if !gui.logsTabActive() {
		return nil
	}

	mainView := gui.Views.Main
	mainView.Autoscroll = true

	_, height := mainView.Size()
	origin := mainView.ViewLinesHeight() - height
	if origin < 0 {
		origin = 0
	}

	if err := mainView.SetOriginY(origin); err != nil {
		return err
	}

	gui.g.Update(func(*gocui.Gui) error { return nil })

	return nil
}

// logsOptionsMap is the bottom-line help for the Logs tab.
func (gui *Gui) logsOptionsMap() map[string]string {
	options := map[string]string{
		"PgUp/PgDn": gui.Tr.Scroll,
		"G":         gui.Tr.JumpToLatest,
		"ctrl+l":    gui.Tr.AllLogs,
		"ctrl+t":    gui.Tr.TranscriptLogs,
	}

	options["ctrl+l"] = gui.logsToggleDescription()
	options["ctrl+t"] = gui.transcriptToggleDescription()

	return options
}

// configureLogWriter applies the log budgets to a writer. Full history keeps
// more of itself: ctrl+l is an explicit request to read back through the log,
// and holding only the newest megabyte of a long history makes that useless.
func (gui *Gui) configureLogWriter(container *commands.Container, writer *logTrimWriter) {
	writer.maxLineBytes = gui.Config.UserConfig.Logs.MaxLineLength
	writer.maxBytes = gui.Config.UserConfig.Logs.MaxBytes

	if gui.logsFullFor(container) {
		writer.keepLines *= logViewFullHistoryScale
		writer.trimAtLines *= logViewFullHistoryScale
		writer.maxBytes *= logViewFullHistoryScale
	}
}

func (gui *Gui) renderContainerLogsToMain(container *commands.Container) tasks.TaskFunc {
	return gui.NewTickerTask(TickerTaskOpts{
		Func: func(ctx context.Context, notifyStopped chan struct{}) {
			gui.renderContainerLogsToMainAux(container, ctx, notifyStopped)
		},
		Duration:   time.Millisecond * 200,
		Wrap:       gui.Config.UserConfig.Gui.WrapMainPanel,
		Autoscroll: true,
	})
}

func (gui *Gui) renderContainerLogsToMainAux(container *commands.Container, ctx context.Context, notifyStopped chan struct{}) {
	if gui.transcriptLogsFor(container) {
		gui.renderContainerTranscriptToMain(container, ctx, notifyStopped)
		return
	}

	mainView := gui.Views.Main
	restored, resumeSince := gui.restoreLogView(container, mainView)
	if !restored {
		gui.clearMainView()
	}
	cacheGeneration := gui.logCacheGeneration()
	defer func() {
		if ctx.Err() != nil {
			gui.cacheLogView(container, mainView, cacheGeneration)
		}
		notifyStopped <- struct{}{}
	}()

	// Keep the old streaming behaviour (lines appear immediately) but bound the
	// view's memory: follow mode would otherwise keep every line until exit.
	logWriter := newLogTrimWriter(mainView, func() bool { return mainView.Autoscroll }, func() bool { return ctx.Err() != nil })
	gui.configureLogWriter(container, logWriter)
	if restored {
		logWriter.adoptCurrentContent()
	} else {
		logWriter.holdInitialFor(logInitialHold)
	}
	defer logWriter.Close()

	tail := gui.containerLogTail(container)
	since := ""
	if restored {
		// The cached view already contains everything up to savedAt. Fetch only
		// what arrived after it, so returning to the tab neither replays the tail
		// nor loses logs emitted while another tab was open.
		tail = ""
		since = resumeSince.Format(time.RFC3339Nano)
	}
	logOutput := io.Writer(logWriter)
	if !gui.transcriptAutoDisabledFor(container) {
		logOutput = io.MultiWriter(logWriter, newTranscriptDetector(func() {
			gui.enableTranscriptLogsAutomatically(container)
		}))
	}
	writeErr := gui.writeContainerLogs(container, ctx, logOutput, tail, since)
	logWriter.flushInitial()
	if writeErr != nil {
		gui.Log.Error(writeErr)
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

// renderContainerTranscriptToMain replays the log stream through a terminal
// emulator and draws the resulting screen. This makes output from full-screen
// programs readable instead of showing their raw cursor/erase sequences.
//
// shortcut: unlike plain logs this cannot resume from the view cache, because the
// terminal state lives in the stopped task. Switching tabs re-reads the tail.
// Accepted: transcript mode is the rare case, and replaying is what makes the
// screen correct.
func (gui *Gui) renderContainerTranscriptToMain(container *commands.Container, ctx context.Context, notifyStopped chan struct{}) {
	defer func() { notifyStopped <- struct{}{} }()

	mainView := gui.Views.Main
	gui.clearMainView()

	cols, rows := mainView.Size()
	if cols < 1 || rows < 1 {
		return
	}

	transcript := commands.NewLogTranscript(cols, rows)
	done := make(chan error, 1)
	go func() {
		done <- gui.writeContainerLogs(container, ctx, transcript, gui.transcriptLogTail(), "")
	}()

	// Redrawing the whole view every tick allocates a copy of the screen even
	// when the container printed nothing, so only render real changes.
	rendered := ""
	render := func() {
		content := strings.Join(transcript.Snapshot(), "\n")
		if content == rendered {
			return
		}

		rendered = content
		gui.RenderStringMain(content)
	}

	ticker := time.NewTicker(time.Millisecond * 150)
	defer ticker.Stop()

	render()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-done:
			render()
			if err != nil {
				gui.Log.Error(err)
			}
			gui.waitForContainerRunning(ctx, container)
			return
		case <-ticker.C:
			render()
		}
	}
}

func (gui *Gui) waitForContainerRunning(ctx context.Context, container *commands.Container) {
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := container.Inspect()
			if err != nil {
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	defer signal.Stop(stop)

	go func() {
		select {
		case <-stop:
			cancel()
		case <-ctx.Done():
		}
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

	if err := gui.writeContainerLogs(container, ctx, os.Stdout, gui.containerLogTail(container), ""); err != nil {
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

func (gui *Gui) containerLogSince(ctr *commands.Container) string {
	if gui.logsFullFor(ctr) {
		return ""
	}

	// A stopped container usually has no output in the last 60 minutes. The
	// configured tail still bounds the request, so ignoring the age filter here
	// shows its last lines instead of an empty Logs tab.
	if ctr.Container.State == "exited" || ctr.Container.State == "stopped" || ctr.Container.State == "dead" {
		return ""
	}
	if ctr.DetailsLoaded() && ctr.DetailsSnapshot().State != nil && !ctr.DetailsSnapshot().State.Running {
		return ""
	}

	return gui.Config.UserConfig.Logs.Since
}

func (gui *Gui) writeContainerLogs(ctr *commands.Container, ctx context.Context, writer io.Writer, tail string, since string) error {
	if since == "" {
		since = gui.containerLogSince(ctr)
	}
	readCloser, err := gui.PodmanCommand.Client.ContainerLogs(ctx, ctr.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: gui.Config.UserConfig.Logs.Timestamps,
		Since:      since,
		Tail:       tail,
		Follow:     true,
	})
	if err != nil {
		gui.Log.Error(err)
		return err
	}
	defer readCloser.Close()

	// Closing the read side is what actually stops the follow stream; cancelling
	// the request context alone can leave StdCopy blocked and writing into the
	// view after the user has switched tabs.
	streamClosed := make(chan struct{})
	defer close(streamClosed)

	go func() {
		select {
		case <-ctx.Done():
			_ = readCloser.Close()
		case <-streamClosed:
		}
	}()

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

	details := ctr.DetailsSnapshot()
	if details.Config != nil && details.Config.Tty {
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
