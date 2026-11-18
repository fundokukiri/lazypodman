package gui

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
	"github.com/jesseduffield/gocui"
)

const (
	shellTabKey  = "shell"
	shellTicker  = 200 * time.Millisecond
	maxShellRows = 1000
)

// shellState is the per-container shell buffer. Output arrives from a goroutine
// while the render task only reads the lines, so both sides take the mutex.
type shellState struct {
	mutex   sync.Mutex
	session *commands.ShellSession
	lines   []string
	done    bool
}

func (st *shellState) append(lines ...string) {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	st.lines = append(st.lines, lines...)
	if len(st.lines) > maxShellRows {
		st.lines = append([]string(nil), st.lines[len(st.lines)-maxShellRows:]...)
	}
}

func (st *shellState) snapshot() []string {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	return append([]string(nil), st.lines...)
}

func (st *shellState) markDone() {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	st.done = true
}

func (gui *Gui) shellStateFor(containerID string) *shellState {
	gui.shellSessionsMutex.Lock()
	defer gui.shellSessionsMutex.Unlock()

	if gui.shellSessions == nil {
		gui.shellSessions = map[string]*shellState{}
	}

	if state, ok := gui.shellSessions[containerID]; ok {
		return state
	}

	state := &shellState{}
	gui.shellSessions[containerID] = state

	return state
}

// ensureShell opens the shell the first time it is needed and keeps it for later
// renders. A dead shell is replaced, so the user can reopen it after a restart.
func (gui *Gui) ensureShell(container *commands.Container) (*shellState, error) {
	state := gui.shellStateFor(container.ID)

	state.mutex.Lock()
	defer state.mutex.Unlock()

	if state.session != nil && !state.done {
		return state, nil
	}

	if capability := gui.PodmanCommand.Capabilities().CLI; capability.State == commands.CapabilityUnavailable {
		return nil, fmt.Errorf("cli off: %s", capability.Reason)
	}

	client := gui.PodmanCommand.Pods()
	if client == nil {
		return nil, fmt.Errorf("cli off: podman CLI not available")
	}

	if !containerIsRunning(container) {
		return nil, fmt.Errorf("container is not running")
	}

	session, err := client.OpenShell(context.Background(), container.ID)
	if err != nil {
		return nil, err
	}

	state.session = session
	state.done = false
	state.lines = append(state.lines, gui.Tr.ShellStarted)

	go gui.readShellOutput(state, session)

	return state, nil
}

func (gui *Gui) readShellOutput(state *shellState, session *commands.ShellSession) {
	scanner := bufio.NewScanner(session.Output())
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		state.append(scanner.Text())
	}

	state.markDone()
	state.append(gui.Tr.ShellExited)
}

func (gui *Gui) renderContainerShell(container *commands.Container) tasks.TaskFunc {
	return gui.NewTickerTask(TickerTaskOpts{
		Duration: shellTicker,
		Func: func(_ context.Context, notifyStopped chan struct{}) {
			gui.reRenderStringMain(gui.shellContents(container))
			notifyStopped <- struct{}{}
		},
		Autoscroll: true,
	})
}

func (gui *Gui) shellContents(container *commands.Container) string {
	state := gui.shellStateFor(container.ID)
	lines := state.snapshot()

	if len(lines) == 0 {
		return strings.Join([]string{
			fmt.Sprintf("# shell %s", container.Name),
			"",
			gui.Tr.ShellNoSession,
		}, "\n")
	}

	return strings.Join(lines, "\n")
}

// handleShellPrompt is bound to Enter in the main view. It only acts on the Shell
// tab, so the other tabs keep their existing Enter behaviour (none).
func (gui *Gui) handleShellPrompt(g *gocui.Gui, v *gocui.View) error {
	if gui.Panels.Containers == nil || !gui.shellTabActive() {
		return nil
	}

	container, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	if _, err := gui.ensureShell(container); err != nil {
		return gui.createErrorPanel(err.Error())
	}

	state := gui.shellStateFor(container.ID)

	return gui.createPromptPanel(gui.Tr.ShellCommandTitle, func(_ *gocui.Gui, promptView *gocui.View) error {
		command := gui.trimmedContent(promptView)
		if command == "" {
			return nil
		}

		state.mutex.Lock()
		session := state.session
		state.mutex.Unlock()

		if session == nil {
			return nil
		}

		if err := session.Send(command); err != nil {
			return gui.createErrorPanel(err.Error())
		}

		state.append("$ " + command)

		gui.reRenderStringMain(gui.shellContents(container))

		return nil
	})
}

func (gui *Gui) shellTabActive() bool {
	if gui.Panels.Containers == nil {
		return false
	}

	state := gui.Panels.Containers.ContextState
	if state == nil {
		return false
	}

	return state.GetCurrentMainTab().Key == shellTabKey
}

func (gui *Gui) closeShellSessions() {
	gui.shellSessionsMutex.Lock()
	defer gui.shellSessionsMutex.Unlock()

	for _, state := range gui.shellSessions {
		state.mutex.Lock()
		if state.session != nil {
			_ = state.session.Close()
		}
		state.mutex.Unlock()
	}

	gui.shellSessions = nil
}
