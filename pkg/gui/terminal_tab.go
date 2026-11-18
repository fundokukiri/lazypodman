package gui

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
	"github.com/jesseduffield/gocui"
)

const (
	terminalTabKey = "shell"
	terminalTick   = 60 * time.Millisecond
)

// terminalState is the single interactive shell currently attached to a tab.
type terminalState struct {
	containerID string
	terminal    *commands.Terminal
	mutex       sync.Mutex
	cols, rows  int
}

func (gui *Gui) currentTerminal() *terminalState {
	gui.terminalMutex.Lock()
	defer gui.terminalMutex.Unlock()

	return gui.terminal
}

func (gui *Gui) setTerminal(state *terminalState) {
	gui.terminalMutex.Lock()
	defer gui.terminalMutex.Unlock()

	gui.terminal = state
}

// terminalTabFor reports the container whose shell tab should be visible.
func (gui *Gui) terminalTabFor(container *commands.Container) bool {
	state := gui.currentTerminal()

	return state != nil && container != nil && state.containerID == container.ID
}

// handleContainersExecShellMenu is bound to x: it offers the shell action instead
// of starting it, so the tab only appears when the user asks for it.
func (gui *Gui) handleContainersExecShellMenu(g *gocui.Gui, v *gocui.View) error {
	container, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	if gui.terminalTabFor(container) {
		return gui.startContainerTerminal(container)
	}

	return gui.Menu(CreateMenuOptions{
		Title: gui.Tr.ContainerShellTitle,
		Items: []*types.MenuItem{
			{
				LabelColumns: []string{gui.Tr.ExecShell},
				OnPress:      func() error { return gui.startContainerTerminal(container) },
			},
		},
	})
}

func (gui *Gui) startContainerTerminal(container *commands.Container) error {
	if container == nil {
		return nil
	}

	if state := gui.currentTerminal(); state != nil {
		if state.containerID == container.ID {
			return gui.focusTerminalView()
		}

		if err := gui.closeContainerTerminal(); err != nil {
			return err
		}
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

	terminal, err := client.OpenTerminal(context.Background(), container.ID, cols, rows)
	if err != nil {
		return gui.createErrorPanel(err.Error())
	}

	gui.setTerminal(&terminalState{containerID: container.ID, terminal: terminal})

	if state := gui.currentTerminal(); state != nil {
		state.cols, state.rows = cols, rows
	}

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

	state := gui.currentTerminal()
	if state == nil || state.terminal != terminal {
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

	gui.setTerminal(nil)
	_ = state.terminal.Close()

	gui.g.Update(func(*gocui.Gui) error {
		gui.Views.Terminal.Visible = false
		gui.g.Cursor = false

		return nil
	})

	if gui.Panels.Containers == nil {
		return nil
	}

	if wasFocused {
		if err := gui.returnFocus(); err != nil {
			return err
		}

		gui.Panels.Containers.ContextState.SetMainTabIndex(0)
	}

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

	return cols, rows
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
	if _, err := g.SetView("terminal", x0, y0, x1, y1, 0); err != nil && err.Error() != UNKNOWN_VIEW_ERROR_MSG {
		return err
	}

	terminalView.Frame = false
	terminalView.Visible = true

	cols, rows := terminalView.Size()
	if cols < 2 || rows < 2 {
		return nil
	}

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

func (gui *Gui) drawTerminal(container *commands.Container) {
	state := gui.currentTerminal()
	if state == nil || state.containerID != container.ID {
		return
	}

	state.mutex.Lock()
	rows, cursorX, cursorY, cursorVisible := state.terminal.Snapshot()
	state.mutex.Unlock()

	content := strings.Join(trimTerminalRows(rows), "\n")

	gui.g.Update(func(*gocui.Gui) error {
		if gui.Views.Terminal == nil || !gui.Views.Terminal.Visible {
			return nil
		}

		_ = gui.setViewContent(gui.Views.Terminal, content)
		gui.g.Cursor = cursorVisible && gui.g.CurrentView() == gui.Views.Terminal

		_, maxY := gui.Views.Terminal.Size()
		if cursorY >= 0 && cursorY < maxY {
			_ = gui.Views.Terminal.SetCursor(cursorX, cursorY)
		}

		return nil
	})
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

// terminalKeyBindings forwards keystrokes to the PTY. They live on the terminal
// view only, so normal panels keep their bindings. ctrl+x closes the shell.
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
		&Binding{ViewName: "terminal", Key: gocui.KeyPgup, Modifier: gocui.ModNone, Handler: send("\x1b[5~")},
		&Binding{ViewName: "terminal", Key: gocui.KeyPgdn, Modifier: gocui.ModNone, Handler: send("\x1b[6~")},
		&Binding{ViewName: "terminal", Key: gocui.KeyCtrlX, Modifier: gocui.ModNone, Handler: func(_ *gocui.Gui, _ *gocui.View) error { return gui.closeContainerTerminal() }, Description: ""},
	)

	for key := gocui.KeyCtrlA; key <= gocui.KeyCtrlZ; key++ {
		if key == gocui.KeyCtrlX {
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

func (gui *Gui) writeTerminal(data []byte) error {
	state := gui.currentTerminal()
	if state == nil {
		return nil
	}

	state.mutex.Lock()
	defer state.mutex.Unlock()

	return state.terminal.Write(data)
}
