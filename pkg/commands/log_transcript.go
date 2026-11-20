package commands

import (
	"sync"

	"github.com/hinshun/vt10x"
)

// LogTranscript replays a container log stream through a terminal emulator so
// the Logs tab can show the final screen of a TUI instead of raw control
// sequences. It is not a PTY: writes only feed the screen, and Snapshot returns
// the visible rows with their ANSI colours.
type LogTranscript struct {
	mu     sync.Mutex
	screen vt10x.Terminal
	cols   int
	rows   int
}

// NewLogTranscript creates a screen with the given visible size.
func NewLogTranscript(cols, rows int) *LogTranscript {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	return &LogTranscript{
		screen: vt10x.New(vt10x.WithSize(cols, rows)),
		cols:   cols,
		rows:   rows,
	}
}

// Write feeds raw log bytes into the terminal parser.
func (t *LogTranscript) Write(p []byte) (int, error) {
	if t == nil {
		return len(p), nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	_, _ = t.screen.Write(p)

	return len(p), nil
}

// Resize changes the emulated screen size.
func (t *LogTranscript) Resize(cols, rows int) {
	if t == nil || cols < 1 || rows < 1 {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.screen.Resize(cols, rows)
	t.cols = cols
	t.rows = rows
}

// Snapshot returns every visible row, including trailing blanks, with SGR
// colours preserved. The caller decides how much of the tail to display.
func (t *LogTranscript) Snapshot() []string {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	rows := make([]string, t.rows)
	for y := 0; y < t.rows; y++ {
		rows[y] = terminalRow(t.screen, y, t.cols)
	}

	return rows
}
