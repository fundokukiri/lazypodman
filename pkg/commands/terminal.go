package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

// Terminal is a real PTY-backed shell inside a container, used by the Shell tab.
// The command runs in a pseudo-terminal and its output is kept in a VT10x screen
// buffer, so full-screen programs (vim, htop, less) lay out correctly. Colours
// are parsed but not rendered yet: gocui draws plain text.
type Terminal struct {
	screen vt10x.Terminal
	pty    *os.File

	cancel    context.CancelFunc
	closeOnce sync.Once
	closeErr  error
	done      chan struct{}
}

// TerminalBudget bounds how long Close waits for the child to exit.
var TerminalBudget = 2 * time.Second

// OpenTerminal starts an interactive shell in the container and sizes the PTY to
// cols x rows. The caller owns the returned Terminal and must Close it.
func (c *cliPodmanClient) OpenTerminal(ctx context.Context, ref string, cols, rows int) (*Terminal, error) {
	if err := validatePodRef(ref); err != nil {
		return nil, err
	}

	if cols < 1 || rows < 1 {
		return nil, fmt.Errorf("invalid terminal size %dx%d", cols, rows)
	}

	execCtx, cancel := context.WithCancel(ctx)

	command := exec.CommandContext(execCtx, c.binary, "exec", "-it", "--", ref, "/bin/sh", "-c", "eval $(grep ^$(id -un): /etc/passwd | cut -d : -f 7-)")
	command.WaitDelay = TerminalBudget

	file, err := pty.StartWithSize(command, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		cancel()

		return nil, err
	}

	terminal := &Terminal{
		screen: vt10x.New(vt10x.WithSize(cols, rows)),
		pty:    file,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		defer close(terminal.done)

		_, _ = io.Copy(terminal.screen, file)
	}()

	return terminal, nil
}

// Write forwards raw bytes (keystrokes) to the shell.
func (t *Terminal) Write(data []byte) error {
	if t == nil {
		return fmt.Errorf("no terminal")
	}

	_, err := t.pty.Write(data)

	return err
}

// Resize changes both the PTY and the screen buffer.
func (t *Terminal) Resize(cols, rows int) error {
	if t == nil {
		return fmt.Errorf("no terminal")
	}

	if cols < 1 || rows < 1 {
		return nil
	}

	t.screen.Resize(cols, rows)

	return pty.Setsize(t.pty, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

// Snapshot returns the visible screen as plain text rows plus the cursor. The
// rows always have exactly terminal height lines; callers trim trailing blanks.
func (t *Terminal) Snapshot() (rows []string, cursorX, cursorY int, visible bool) {
	if t == nil {
		return nil, 0, 0, false
	}

	t.screen.Lock()
	defer t.screen.Unlock()

	cols, screenRows := t.screen.Size()
	cursor := t.screen.Cursor()

	rows = make([]string, screenRows)
	for y := 0; y < screenRows; y++ {
		line := make([]rune, cols)
		for x := 0; x < cols; x++ {
			line[x] = t.screen.Cell(x, y).Char
		}

		rows[y] = string(line)
	}

	return rows, cursor.X, cursor.Y, t.screen.CursorVisible()
}

// Done is closed when the shell process ends.
func (t *Terminal) Done() <-chan struct{} {
	return t.done
}

// Close kills the shell and releases the PTY. Safe to call twice.
func (t *Terminal) Close() error {
	if t == nil {
		return nil
	}

	t.closeOnce.Do(func() {
		t.cancel()
		t.closeErr = t.pty.Close()
	})

	select {
	case <-t.done:
	case <-time.After(TerminalBudget):
	}

	return t.closeErr
}
