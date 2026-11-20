package gui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
	"github.com/jesseduffield/gocui"
	"github.com/mattn/go-runewidth"
)

const (
	terminalTabKey       = "shell"
	terminalTick         = 60 * time.Millisecond
	terminalHistoryLimit = 5000
)

// terminalState is the single interactive shell currently attached to a tab.
type terminalState struct {
	containerID string
	terminal    *commands.Terminal
	mutex       sync.Mutex
	cols, rows  int

	// history keeps lines that scrolled off the screen so the user can page back
	// through shell output, like the Logs tab.
	history      []string
	lastRows     []string
	scrollOffset int
	scrollable   int
}

func (gui *Gui) terminalFor(containerID string) *terminalState {
	gui.terminalMutex.Lock()
	defer gui.terminalMutex.Unlock()

	return gui.terminals[containerID]
}

func (gui *Gui) currentTerminal() *terminalState {
	container := gui.terminalContainer()
	if container == nil {
		return nil
	}

	return gui.terminalFor(container.ID)
}

// terminalContainer is the container whose shell tab belongs to the current
// side-panel context. Shells for other containers stay alive in the background.
func (gui *Gui) terminalContainer() *commands.Container {
	switch gui.currentSideViewName() {
	case "services":
		if gui.Panels.Services != nil {
			if service, err := gui.Panels.Services.GetSelectedItem(); err == nil && service != nil {
				return service.Container
			}
		}
	case "containers":
		if gui.Panels.Containers != nil {
			if container, err := gui.Panels.Containers.GetSelectedItem(); err == nil {
				return container
			}
		}
	}

	return nil
}

func (gui *Gui) addTerminal(state *terminalState) {
	gui.terminalMutex.Lock()
	defer gui.terminalMutex.Unlock()

	if gui.terminals == nil {
		gui.terminals = map[string]*terminalState{}
	}
	gui.terminals[state.containerID] = state
}

func (gui *Gui) removeTerminal(state *terminalState) {
	if state == nil {
		return
	}

	gui.terminalMutex.Lock()
	defer gui.terminalMutex.Unlock()

	if gui.terminals[state.containerID] == state {
		delete(gui.terminals, state.containerID)
	}
}

func (gui *Gui) terminalByTerminal(terminal *commands.Terminal) *terminalState {
	gui.terminalMutex.Lock()
	defer gui.terminalMutex.Unlock()

	for _, state := range gui.terminals {
		if state.terminal == terminal {
			return state
		}
	}

	return nil
}

// terminalTabFor reports the container whose shell tab should be visible.
func (gui *Gui) terminalTabFor(container *commands.Container) bool {
	return container != nil && gui.terminalFor(container.ID) != nil
}

func (gui *Gui) startContainerTerminal(container *commands.Container) error {
	if container == nil {
		return nil
	}

	if gui.terminalFor(container.ID) != nil {
		return gui.focusTerminalView()
	}

	if capability := gui.PodmanCommand.Capabilities().CLI; capability.State == commands.CapabilityUnavailable {
		return gui.createErrorPanel("cli off: " + capability.Reason)
	}

	if !containerIsRunning(container) {
		return gui.createErrorPanel(gui.Tr.ContainerNotRunning)
	}

	client := gui.PodmanCommand.Pods()
	if client == nil {
		return gui.createErrorPanel("cli off: podman CLI not available")
	}

	cols, rows := gui.terminalSize()

	var (
		terminal *commands.Terminal
		err      error
	)
	terminal, err = client.OpenTerminal(context.Background(), container.ID, cols, rows)
	if err != nil {
		return gui.createErrorPanel(err.Error())
	}

	state := &terminalState{containerID: container.ID, terminal: terminal, cols: cols, rows: rows}
	gui.addTerminal(state)

	if err := gui.selectShellTab(); err != nil {
		return err
	}

	if err := gui.focusTerminalView(); err != nil {
		return err
	}

	gui.flashStatus(gui.Tr.ShellCloseHint)

	go gui.watchTerminal(terminal)

	return nil
}

// watchTerminal closes the tab when the shell exits on its own.
func (gui *Gui) watchTerminal(terminal *commands.Terminal) {
	<-terminal.Done()

	state := gui.terminalByTerminal(terminal)
	if state == nil {
		return
	}

	if gui.currentTerminal() != state {
		// The shell belonged to a container that is not selected right now.
		// Drop it quietly; the next visit simply has no shell tab.
		gui.removeTerminal(state)
		return
	}

	_ = gui.closeContainerTerminal()

	gui.flashStatus(gui.Tr.ShellExited)
}

func (gui *Gui) closeContainerTerminal() error {
	state := gui.currentTerminal()
	if state == nil {
		return nil
	}

	wasFocused := gui.g.CurrentView() == gui.Views.Terminal

	gui.removeTerminal(state)
	_ = state.terminal.Close()

	// Wipe both views before reloading: the tab that was showing before the shell
	// (stats, env, ...) and the terminal's last screen must not survive behind the
	// logs that are about to load.
	gui.clearMainView()

	gui.g.Update(func(*gocui.Gui) error {
		gui.Views.Terminal.Clear()
		gui.Views.Terminal.Visible = false
		gui.g.Cursor = false

		// Do not clear the main view here. This update is asynchronous and can
		// run after the Logs task has already restored its cached content,
		// leaving the tab blank until a new log line arrives. clearMainView
		// above already wiped it synchronously.
		return nil
	})

	if gui.Panels.Containers == nil {
		return nil
	}

	if wasFocused {
		if err := gui.returnFocus(); err != nil {
			return err
		}
	}

	// Always land back on Logs and force a re-render so the tab strip drops the
	// shell tab even when Logs was already the current context.
	gui.Panels.Containers.ContextState.SetMainTabIndex(0)
	gui.State.Panels.Main.ObjectKey = ""

	return gui.Panels.Containers.HandleSelect()
}

func (gui *Gui) focusTerminalView() error {
	gui.g.Update(func(*gocui.Gui) error {
		gui.Views.Terminal.Visible = true

		return nil
	})

	if err := gui.switchFocus(gui.Views.Terminal); err != nil {
		return err
	}

	gui.g.Update(func(*gocui.Gui) error {
		gui.g.Cursor = true

		return nil
	})

	return nil
}

// handleFocusTerminal is bound to enter on the main view: on the shell tab it
// gives the keyboard to the terminal.
func (gui *Gui) handleFocusTerminal(g *gocui.Gui, v *gocui.View) error {
	state := gui.currentTerminal()
	if state == nil || !gui.shellTabActive() {
		return nil
	}

	return gui.focusTerminalView()
}

func (gui *Gui) shellTabActive() bool {
	if gui.Panels.Containers == nil || gui.Panels.Containers.ContextState == nil {
		return false
	}

	return gui.Panels.Containers.ContextState.GetCurrentMainTab().Key == terminalTabKey
}

func (gui *Gui) selectShellTab() error {
	tabs := gui.Panels.Containers.ContextState.GetMainTabs()

	for index, tab := range tabs {
		if tab.Key == terminalTabKey {
			gui.Panels.Containers.ContextState.SetMainTabIndex(index)
			return gui.Panels.Containers.HandleSelect()
		}
	}

	return nil
}

// terminalSize is the inner size of the main view, which is where the terminal
// view is placed.
func (gui *Gui) terminalSize() (int, int) {
	if gui.Views.Main == nil {
		return 80, 24
	}

	cols, rows := gui.Views.Main.Size()
	if cols < 10 || rows < 3 {
		return 80, 24
	}

	// One row is reserved for the container info line.
	return cols, rows - 1
}

// layoutTerminalView keeps the terminal view exactly over the main panel's
// content area and sizes the PTY to match.
func (gui *Gui) layoutTerminalView(g *gocui.Gui) error {
	state := gui.currentTerminal()

	terminalView := gui.Views.Terminal
	if terminalView == nil {
		return nil
	}

	if state == nil || gui.Views.Main == nil || !gui.shellTabActive() {
		terminalView.Visible = false
		return nil
	}

	mainView := gui.Views.Main
	x0, y0, x1, y1 := mainView.Dimensions()

	// gocui always offsets a view's content by one cell, even without a frame, so
	// using the main view's own top row keeps the header flush under the tabs
	// instead of leaving a leftover main-view line above it.
	if _, err := g.SetView("terminal", x0, y0, x1, y1, 0); err != nil && err.Error() != UNKNOWN_VIEW_ERROR_MSG {
		return err
	}

	terminalView.Frame = false
	terminalView.Visible = true

	cols, rows := terminalView.Size()
	if cols < 2 || rows < 2 {
		return nil
	}

	// The header occupies the first row.
	rows--

	state.mutex.Lock()
	defer state.mutex.Unlock()

	if state.cols != cols || state.rows != rows {
		state.cols, state.rows = cols, rows
		if err := state.terminal.Resize(cols, rows); err != nil {
			return nil
		}
	}

	_, err := g.SetViewOnTop("terminal")

	return err
}

func (gui *Gui) renderContainerTerminal(container *commands.Container) tasks.TaskFunc {
	return func(ctx context.Context) {
		ticker := time.NewTicker(terminalTick)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				gui.drawTerminal(container)
			}
		}
	}
}

// terminalHeader is the info line shown under the tabs while the shell is open.
func terminalHeader(container *commands.Container, tr *i18n.TranslationSet) string {
	image := container.Container.Image
	status := container.Container.Status

	details := container.DetailsSnapshot()
	if details.ContainerJSONBase != nil && details.Config != nil && details.State != nil {
		image = details.Config.Image
		status = details.State.Status
	}

	id := container.ID
	if len(id) > 12 {
		id = id[:12]
	}

	label := func(text string) string { return "\x1b[36m" + text + "\x1b[0m" }

	statusColor := "\x1b[31m"
	if strings.HasPrefix(strings.ToLower(status), "up") || strings.HasPrefix(strings.ToLower(status), "running") {
		statusColor = "\x1b[32m"
	}

	header := fmt.Sprintf(tr.ShellHeader,
		"\x1b[33m"+container.Name+"\x1b[0m",
		"\x1b[35m"+image+"\x1b[0m",
		statusColor+status+"\x1b[0m",
		"\x1b[90m"+id+"\x1b[0m",
	)
	header = strings.ReplaceAll(header, "container:", label("container:"))
	header = strings.ReplaceAll(header, "image:", label("image:"))
	header = strings.ReplaceAll(header, "status:", label("status:"))
	header = strings.ReplaceAll(header, "id:", label("id:"))

	if container.PodName != "" {
		header += fmt.Sprintf(tr.ShellHeaderPod, label("pod:")+" \x1b[35m"+container.PodName+"\x1b[0m")
	}

	return header
}

func (gui *Gui) drawTerminal(container *commands.Container) {
	state := gui.currentTerminal()
	if state == nil || state.containerID != container.ID {
		return
	}

	maxCols, maxRows := gui.Views.Terminal.Size()

	state.mutex.Lock()
	rows, screenCursorX, screenCursorY, cursorVisible := state.terminal.Snapshot()
	trimmed := trimTerminalRows(rows)

	// The first row is the container header; the terminal gets the rest.
	height := maxRows - 1
	if height < 0 {
		height = 0
	}

	historyRows, historyCursorX, historyCursorY, scrollbackOK := state.terminal.Scrollback()

	offset := state.scrollOffset

	var visible []string
	var total int
	cursorX, cursorY := screenCursorX, screenCursorY

	if scrollbackOK && len(historyRows) > 0 {
		total = len(historyRows)
		state.scrollable = total - height
		if state.scrollable < 0 {
			state.scrollable = 0
		}

		if offset > state.scrollable {
			offset = state.scrollable
			state.scrollOffset = offset
		}

		visible = terminalWindow(nil, historyRows, height, offset)

		if offset > 0 {
			start := total - height - offset
			if start < 0 {
				start = 0
			}

			cursorX, cursorY = historyCursorX, historyCursorY-start
			if cursorY < 0 || cursorY >= len(visible) {
				cursorY = -1
			}
		} else {
			// Keep the history window for the rows (it is what the shell has been
			// showing all along), but take the caret from the live screen: TUIs
			// position the hardware cursor on the live screen, and the history copy
			// can have a different cursor row.
			cursorX, cursorY = screenCursorX, screenCursorY
		}
	} else {
		// Alt-screen programs (vim, htop, less) redraw the screen; fall back to
		// per-frame scroll detection so the live screen still works.
		if lost := scrolledOffLines(state.lastRows, trimmed); len(lost) > 0 {
			state.history = append(state.history, lost...)
			if len(state.history) > terminalHistoryLimit {
				state.history = state.history[len(state.history)-terminalHistoryLimit:]
			}
		}

		state.lastRows = trimmed

		total = len(state.history) + len(trimmed)
		state.scrollable = total - height
		if state.scrollable < 0 {
			state.scrollable = 0
		}

		if offset > 0 {
			visible = terminalWindow(state.history, trimmed, height, offset)
		} else {
			// Live view: draw the screen exactly as the PTY has it, so the cursor
			// row keeps matching and history lines never push the shell down.
			visible = trimmed
			if len(visible) > height {
				visible = visible[len(visible)-height:]
			}
		}
	}

	state.mutex.Unlock()

	// Pad to the full inner height so the terminal view paints every cell of its
	// area; otherwise rows it does not write can still show the view beneath it.
	for len(visible) < height {
		visible = append(visible, "")
	}

	// Draw a scrollbar thumb in the last column while there is history to page
	// through, like the Logs view.
	if start, size, ok := terminalScrollbar(total, height, offset); ok && maxCols > 1 {
		for i := 0; i < len(visible) && i < height; i++ {
			line := truncateANSI(visible[i], maxCols-1)
			line += strings.Repeat(" ", maxCols-1-ansiWidth(line))

			if i >= start && i < start+size {
				line += "▐"
			} else {
				line += " "
			}

			visible[i] = line
		}
	}

	content := terminalHeader(container, gui.Tr)
	if len(visible) > 0 {
		content += "\n" + strings.Join(visible, "\n")
	}

	gui.g.Update(func(*gocui.Gui) error {
		if gui.Views.Terminal == nil || !gui.Views.Terminal.Visible {
			return nil
		}

		_ = gui.setViewContent(gui.Views.Terminal, content)

		// Hide the caret while paging: it belongs to the live screen, not to the
		// history window, so leaving it visible makes it float over old output.
		showCursor := offset == 0 && cursorVisible && cursorY >= 0 &&
			gui.g.CurrentView() == gui.Views.Terminal
		gui.g.Cursor = showCursor

		if showCursor && cursorY+1 < maxRows {
			_ = gui.Views.Terminal.SetCursor(cursorX, cursorY+1)
		}

		return nil
	})
}

// terminalOptionsMap is the bottom-line help for the shell: the global hints
// (q, b, x) are forwarded to the container, so they must not be advertised here.
func (gui *Gui) terminalOptionsMap() map[string]string {
	return map[string]string{
		"wheel/PgUp/PgDn": gui.Tr.Scroll,
		"ctrl+o":          gui.Tr.LeaveShell,
		"ctrl+x":          gui.Tr.CloseShell,
		"ctrl+q":          gui.Tr.Quit,
	}
}

// ansiWidth returns the printed width of a row, ignoring SGR escape sequences.
func ansiWidth(line string) int {
	width := 0

	for i := 0; i < len(line); {
		if line[i] == 0x1b {
			if size := ansiSequenceLen([]byte(line[i:])); size > 0 {
				i += size
				continue
			}
		}

		r, size := utf8.DecodeRuneInString(line[i:])
		width += runewidth.RuneWidth(r)
		i += size
	}

	return width
}

// truncateANSI cuts a row to maxWidth printed cells without breaking an SGR
// sequence. The trailing reset keeps a cut colour from leaking into later rows.
func truncateANSI(line string, maxWidth int) string {
	var out strings.Builder

	width := 0
	truncated := false

	for i := 0; i < len(line); {
		if line[i] == 0x1b {
			if size := ansiSequenceLen([]byte(line[i:])); size > 0 {
				out.WriteString(line[i : i+size])
				i += size
				continue
			}
		}

		r, size := utf8.DecodeRuneInString(line[i:])
		runeWidth := runewidth.RuneWidth(r)
		if width+runeWidth > maxWidth {
			truncated = true
			break
		}

		out.WriteRune(r)
		width += runeWidth
		i += size
	}

	if truncated {
		out.WriteString("\x1b[0m")
	}

	return out.String()
}

// trimTerminalRows drops trailing blanks and zero bytes so the view does not
// grow with invisible cells. Blank rows inside the screen are kept.
func trimTerminalRows(rows []string) []string {
	trimmed := make([]string, len(rows))

	for i, row := range rows {
		trimmed[i] = strings.TrimRight(row, " \x00")
	}

	for len(trimmed) > 0 && trimmed[len(trimmed)-1] == "" {
		trimmed = trimmed[:len(trimmed)-1]
	}

	return trimmed
}

// leaveTerminalFocus keeps the shell running but gives the keyboard back to the
// containers panel, so [ ] and the rest of the UI work again.
func (gui *Gui) leaveTerminalFocus() error {
	if gui.currentTerminal() == nil || gui.Panels.Containers == nil {
		return nil
	}

	gui.g.Update(func(*gocui.Gui) error {
		gui.g.Cursor = false

		return nil
	})

	return gui.switchFocus(gui.Panels.Containers.GetView())
}

// terminalEditor forwards printable non-ASCII runes to the PTY. The keybindings
// below cover ASCII and control keys, but there is no practical way to bind every
// Unicode code point, and IMEs (for example Vietnamese Telex) deliver composed
// characters as runes that gocui would otherwise swallow.
func (gui *Gui) terminalEditor() gocui.Editor {
	return gocui.EditorFunc(func(_ *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) bool {
		data, ok := terminalInputBytes(key, ch, mod)
		if !ok {
			return false
		}

		_ = gui.writeTerminal(data)

		return true
	})
}

func terminalInputBytes(key gocui.Key, ch rune, mod gocui.Modifier) ([]byte, bool) {
	if ch == 0 || mod != gocui.ModNone || (key != 0 && key != gocui.KeySpace) {
		return nil, false
	}

	return []byte(string(ch)), true
}

// terminalKeyBindings forwards keystrokes to the PTY. They live on the terminal
// view only, so normal panels keep their bindings. ctrl+o leaves the shell
// focused but keeps it running; ctrl+x closes it.
func (gui *Gui) terminalKeyBindings() []*Binding {
	bindings := make([]*Binding, 0, 140)

	send := func(data string) func(*gocui.Gui, *gocui.View) error {
		return func(_ *gocui.Gui, _ *gocui.View) error {
			return gui.writeTerminal([]byte(data))
		}
	}

	for r := rune(33); r <= 126; r++ {
		bindings = append(bindings, &Binding{ViewName: "terminal", Key: r, Modifier: gocui.ModNone, Handler: send(string(r))})
	}

	bindings = append(bindings,
		&Binding{ViewName: "terminal", Key: gocui.KeySpace, Modifier: gocui.ModNone, Handler: send(" ")},
		&Binding{ViewName: "terminal", Key: gocui.KeyEnter, Modifier: gocui.ModNone, Handler: send("\r")},
		&Binding{ViewName: "terminal", Key: gocui.KeyTab, Modifier: gocui.ModNone, Handler: send("\t")},
		&Binding{ViewName: "terminal", Key: gocui.KeyBackspace, Modifier: gocui.ModNone, Handler: send("\x7f")},
		&Binding{ViewName: "terminal", Key: gocui.KeyBackspace2, Modifier: gocui.ModNone, Handler: send("\x7f")},
		&Binding{ViewName: "terminal", Key: gocui.KeyEsc, Modifier: gocui.ModNone, Handler: send("\x1b")},
		&Binding{ViewName: "terminal", Key: gocui.KeyArrowUp, Modifier: gocui.ModNone, Handler: send("\x1b[A")},
		&Binding{ViewName: "terminal", Key: gocui.KeyArrowDown, Modifier: gocui.ModNone, Handler: send("\x1b[B")},
		&Binding{ViewName: "terminal", Key: gocui.KeyArrowRight, Modifier: gocui.ModNone, Handler: send("\x1b[C")},
		&Binding{ViewName: "terminal", Key: gocui.KeyArrowLeft, Modifier: gocui.ModNone, Handler: send("\x1b[D")},
		&Binding{ViewName: "terminal", Key: gocui.KeyHome, Modifier: gocui.ModNone, Handler: send("\x1b[H")},
		&Binding{ViewName: "terminal", Key: gocui.KeyEnd, Modifier: gocui.ModNone, Handler: send("\x1b[F")},
		&Binding{ViewName: "terminal", Key: gocui.KeyDelete, Modifier: gocui.ModNone, Handler: send("\x1b[3~")},
		&Binding{ViewName: "terminal", Key: gocui.MouseWheelUp, Modifier: gocui.ModNone, Handler: func(_ *gocui.Gui, _ *gocui.View) error { return gui.scrollTerminal(3) }},
		&Binding{ViewName: "terminal", Key: gocui.MouseWheelDown, Modifier: gocui.ModNone, Handler: func(_ *gocui.Gui, _ *gocui.View) error { return gui.scrollTerminal(-3) }},
		&Binding{ViewName: "terminal", Key: gocui.KeyPgup, Modifier: gocui.ModNone, Handler: func(_ *gocui.Gui, _ *gocui.View) error { return gui.scrollTerminal(10) }, Description: gui.Tr.Scroll},
		&Binding{ViewName: "terminal", Key: gocui.KeyPgdn, Modifier: gocui.ModNone, Handler: func(_ *gocui.Gui, _ *gocui.View) error { return gui.scrollTerminal(-10) }, Description: gui.Tr.Scroll},
		&Binding{ViewName: "terminal", Key: gocui.KeyCtrlO, Modifier: gocui.ModNone, Handler: func(_ *gocui.Gui, _ *gocui.View) error { return gui.leaveTerminalFocus() }, Description: gui.Tr.LeaveShell},
		&Binding{ViewName: "terminal", Key: gocui.KeyCtrlX, Modifier: gocui.ModNone, Handler: func(_ *gocui.Gui, _ *gocui.View) error { return gui.closeContainerTerminal() }, Description: gui.Tr.CloseShell},
		&Binding{ViewName: "terminal", Key: gocui.KeyCtrlQ, Modifier: gocui.ModNone, Handler: gui.quit, Description: gui.Tr.Quit},
	)

	for key := gocui.KeyCtrlA; key <= gocui.KeyCtrlZ; key++ {
		if key == gocui.KeyCtrlX || key == gocui.KeyCtrlO || key == gocui.KeyCtrlQ {
			continue
		}

		bindings = append(bindings, &Binding{
			ViewName: "terminal",
			Key:      key,
			Modifier: gocui.ModNone,
			Handler:  send(string(rune(key))),
		})
	}

	return bindings
}

// scrollTerminal pages the shell history. Positive delta scrolls up.
func (gui *Gui) scrollTerminal(delta int) error {
	state := gui.currentTerminal()
	if state == nil {
		return nil
	}

	state.mutex.Lock()
	state.scrollOffset += delta
	if state.scrollOffset > state.scrollable {
		state.scrollOffset = state.scrollable
	}
	if state.scrollOffset < 0 {
		state.scrollOffset = 0
	}
	state.mutex.Unlock()

	gui.g.Update(func(*gocui.Gui) error { return nil })

	return nil
}

// scrolledOffLines returns the lines that left the top of the screen between two
// snapshots (used to build the scrollback history).
func scrolledOffLines(oldRows, newRows []string) []string {
	start := len(oldRows) - len(newRows)
	if start < 1 {
		start = 1
	}

	for k := start; k < len(oldRows); k++ {
		if slicesEqual(oldRows[k:], newRows[:len(oldRows)-k]) {
			return oldRows[:k]
		}
	}

	return nil
}

// terminalScrollbar returns the thumb start row and height for a history of
// total lines shown in a window of the given height, where offset counts lines
// scrolled up from the bottom.
func terminalScrollbar(total, height, offset int) (int, int, bool) {
	if height < 2 || total <= height {
		return 0, 0, false
	}

	scrollable := total - height
	if offset > scrollable {
		offset = scrollable
	}
	if offset < 0 {
		offset = 0
	}

	size := height * height / total
	if size < 1 {
		size = 1
	}
	if size > height {
		size = height
	}

	start := (scrollable - offset) * (height - 1) / scrollable
	if start+size > height {
		start = height - size
	}

	return start, size, true
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// terminalWindow returns the slice of history+screen visible for a given scroll
// offset, where offset 0 means the newest lines.
func terminalWindow(history, screen []string, height, offset int) []string {
	if height <= 0 {
		return nil
	}

	all := make([]string, 0, len(history)+len(screen))
	all = append(all, history...)
	all = append(all, screen...)

	maxOffset := len(all) - height
	if maxOffset < 0 {
		maxOffset = 0
	}

	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}

	start := len(all) - height - offset
	if start < 0 {
		start = 0
	}

	end := start + height
	if end > len(all) {
		end = len(all)
	}

	return all[start:end]
}

func (gui *Gui) writeTerminal(data []byte) error {
	state := gui.currentTerminal()
	if state == nil {
		return nil
	}

	state.mutex.Lock()
	defer state.mutex.Unlock()

	// Typing means the user wants the live prompt, so snap back to the caret even
	// when they were reading scrollback.
	state.scrollOffset = 0

	return state.terminal.Write(data)
}
