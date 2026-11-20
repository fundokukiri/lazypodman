package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
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
	// historyScreen swallows the same output on a very tall screen so scrollback
	// survives bursts (e.g. cat of a long file) that a per-frame snapshot would
	// miss. Full-screen apps are detected through ModeAltScreen and ignored.
	historyScreen vt10x.Terminal
	pty           *os.File

	cancel    context.CancelFunc
	closeOnce sync.Once
	closeErr  error
	done      chan struct{}
}

// TerminalBudget bounds how long Close waits for the child to exit.
var TerminalBudget = 2 * time.Second

// HistoryRows is how many rows the scrollback screen keeps.
const HistoryRows = 2000

// interactiveShellCommand starts the best available shell. It deliberately
// prefers zsh, then bash, and only falls back to sh because a container's
// passwd entry can still point at sh after chsh failed or after the image was
// built without the shell.
const interactiveShellCommand = `if command -v zsh >/dev/null 2>&1; then exec zsh; elif [ -x /bin/zsh ]; then exec /bin/zsh; elif [ -x /usr/bin/zsh ]; then exec /usr/bin/zsh; elif [ -x /usr/local/bin/zsh ]; then exec /usr/local/bin/zsh; elif command -v bash >/dev/null 2>&1; then exec bash; elif [ -x /bin/bash ]; then exec /bin/bash; elif [ -x /usr/bin/bash ]; then exec /usr/bin/bash; elif [ -x /usr/local/bin/bash ]; then exec /usr/local/bin/bash; else exec sh; fi`

// ShellTerminalArgv builds a `podman exec` command that starts the preferred
// shell in a terminal window. a/A use it instead of `podman attach`, so they
// open zsh (or bash/sh) rather than whatever shell the container was started
// with.
func ShellTerminalArgv(podmanBinary, ref string) []string {
	return []string{podmanBinary, "exec", "-it", "--", ref, "/bin/sh", "-c", interactiveShellCommand}
}

// OpenTerminal starts an interactive shell in the container and sizes the PTY to
// cols x rows. The caller owns the returned Terminal and must Close it.
func (c *cliPodmanClient) OpenTerminal(ctx context.Context, ref string, cols, rows int) (*Terminal, error) {
	return c.openTerminal(ctx, ref, cols, rows, []string{
		"exec", "-it", "--", ref, "/bin/sh", "-c", interactiveShellCommand,
	})
}

func (c *cliPodmanClient) openTerminal(ctx context.Context, ref string, cols, rows int, args []string) (*Terminal, error) {
	if err := validatePodRef(ref); err != nil {
		return nil, err
	}

	if cols < 1 || rows < 1 {
		return nil, fmt.Errorf("invalid terminal size %dx%d", cols, rows)
	}

	execCtx, cancel := context.WithCancel(ctx)

	command := exec.CommandContext(execCtx, c.binary, args...)
	command.WaitDelay = TerminalBudget

	file, err := pty.StartWithSize(command, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		cancel()

		return nil, err
	}

	terminal := &Terminal{
		screen:        vt10x.New(vt10x.WithSize(cols, rows)),
		historyScreen: vt10x.New(vt10x.WithSize(cols, HistoryRows)),
		pty:           file,
		cancel:        cancel,
		done:          make(chan struct{}),
	}

	go func() {
		defer close(terminal.done)

		_, _ = io.Copy(io.MultiWriter(terminal.screen, terminal.historyScreen), file)
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
	t.historyScreen.Resize(cols, HistoryRows)

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
		rows[y] = terminalRow(t.screen, y, cols)
	}

	return rows, cursor.X, cursor.Y, t.screen.CursorVisible()
}

// Scrollback returns the full history of a plain (non-alt-screen) session as
// plain text rows, with leading and trailing blank rows trimmed. The cursor is
// returned relative to the trimmed rows. ok is false for full-screen programs.
func (t *Terminal) Scrollback() (rows []string, cursorX, cursorY int, ok bool) {
	if t == nil || t.historyScreen == nil {
		return nil, 0, 0, false
	}

	t.historyScreen.Lock()
	defer t.historyScreen.Unlock()

	if t.historyScreen.Mode()&vt10x.ModeAltScreen != 0 {
		return nil, 0, 0, false
	}

	cols, screenRows := t.historyScreen.Size()
	cursor := t.historyScreen.Cursor()

	isBlank := func(y int) bool {
		for x := 0; x < cols; x++ {
			if ch := t.historyScreen.Cell(x, y).Char; ch != 0 && ch != ' ' {
				return false
			}
		}

		return true
	}

	start := 0
	for start < screenRows && isBlank(start) {
		start++
	}

	if start == screenRows {
		return nil, cursor.X, 0, true
	}

	end := screenRows
	for end > start && isBlank(end-1) {
		end--
	}

	if cursor.Y+1 > end {
		end = cursor.Y + 1
	}

	rows = make([]string, 0, end-start)
	for y := start; y < end; y++ {
		rows = append(rows, strings.TrimRight(terminalRow(t.historyScreen, y, cols), " "))
	}

	cursorY = cursor.Y - start
	if cursorY < 0 {
		cursorY = 0
	}

	return rows, cursor.X, cursorY, true
}

// terminalRow renders one screen row with ANSI SGR colour sequences. Colours are
// emitted only when they change, so an ordinary row stays small and a coloured
// prompt/command keeps the colours the program intended.
func terminalRow(view vt10x.View, y int, cols int) string {
	var row strings.Builder
	row.Grow(cols)

	lastFG, lastBG := vt10x.Color(0), vt10x.Color(0)
	styled := false

	for x := 0; x < cols; x++ {
		cell := view.Cell(x, y)

		if !styled {
			lastFG, lastBG = cell.FG, cell.BG
			styled = true

			if sgr := sgrFor(lastFG, lastBG); sgr != "" && sgr != "\x1b[0m" {
				row.WriteString(sgr)
			}
		} else if cell.FG != lastFG || cell.BG != lastBG {
			row.WriteString(sgrFor(cell.FG, cell.BG))
			lastFG, lastBG = cell.FG, cell.BG
		}

		row.WriteRune(cell.Char)
	}

	if !isDefaultColor(lastFG) || !isDefaultColor(lastBG) || isReverseDefault(lastFG, lastBG) {
		row.WriteString("\x1b[0m")
	}

	return row.String()
}

func sgrFor(fg, bg vt10x.Color) string {
	// vt10x applies reverse video by swapping the cell colours. For the default
	// foreground/background pair both halves are then "default", so emitting no
	// colours would erase the reverse block. That block is how full-screen TUIs
	// draw their input caret when they hide the hardware cursor.
	if isReverseDefault(fg, bg) {
		return "\x1b[7m"
	}

	params := []string{"0"}

	if code := sgrColor(fg, true); code != "" {
		params = append(params, code)
	}

	if code := sgrColor(bg, false); code != "" {
		params = append(params, code)
	}

	return "\x1b[" + strings.Join(params, ";") + "m"
}

func sgrColor(color vt10x.Color, foreground bool) string {
	if isDefaultColor(color) {
		return ""
	}

	base := 30
	if !foreground {
		base = 40
	}

	if color.ANSI() {
		value := int(color)
		if value >= 8 {
			base += 60
			value -= 8
		}

		return fmt.Sprintf("%d", base+value)
	}

	if color < 256 {
		return fmt.Sprintf("%d;5;%d", base+8, color)
	}

	r := (color >> 16) & 0xff
	g := (color >> 8) & 0xff
	b := color & 0xff

	return fmt.Sprintf("%d;2;%d;%d;%d", base+8, r, g, b)
}

func isDefaultColor(color vt10x.Color) bool {
	return color == vt10x.DefaultFG || color == vt10x.DefaultBG || color == vt10x.DefaultCursor
}

func isReverseDefault(fg, bg vt10x.Color) bool {
	return fg == vt10x.DefaultBG && bg == vt10x.DefaultFG
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
