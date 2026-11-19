package gui

import (
	"bytes"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

const (
	// logViewKeepLines is how many lines the Logs view keeps after a trim.
	logViewKeepLines = 10000
	// logViewTrimAt is when a trim happens. Trimming is a full re-render, so it
	// is done rarely (every 20000 lines) to keep the UI responsive.
	logViewTrimAt = 20000
	// logViewFallbackWidth is used before the view has a usable width.
	logViewFallbackWidth = 120
)

// logTrimTarget is the part of *gocui.View the trimmer needs.
type logTrimTarget interface {
	Write(p []byte) (int, error)
	SetContent(string)
	BufferLines() []string
	Size() (int, int)
	ViewLinesHeight() int
	SetOriginY(int) error
}

// logTrimWriter streams lines straight into the view, exactly like the old code,
// but caps the view's line count. A follow stream otherwise keeps every line in
// memory until the app is closed, which is what made RAM explode on long logs.
type logTrimWriter struct {
	mutex     sync.Mutex
	view      logTrimTarget
	lines     int
	lineCols  int
	pending   []byte
	buf       []byte
	shouldPin func() bool
	stale     func() bool
}

func newLogTrimWriter(view logTrimTarget, shouldPin func() bool, stale func() bool) *logTrimWriter {
	return &logTrimWriter{view: view, shouldPin: shouldPin, stale: stale}
}

func (w *logTrimWriter) Write(p []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// A cancelled log task must not write into the main view after another tab
	// has rendered its own content there.
	if w.stale != nil && w.stale() {
		return len(p), nil
	}

	data := p
	if len(w.pending) > 0 {
		data = append(append([]byte(nil), w.pending...), p...)
		w.pending = nil
	}

	out := w.wrapLinesLocked(data)
	if _, err := w.view.Write(out); err != nil {
		return 0, err
	}

	w.lines += bytes.Count(out, []byte("\n"))
	if w.lines > logViewTrimAt {
		w.trimLocked()
	}

	w.pinToBottomLocked()

	return len(p), nil
}

// wrapLinesLocked inserts newlines at the view width so a single enormous
// horizontal line does not force gocui to wrap and render megabytes over and
// over. All bytes are kept: the line is wrapped, not truncated. ANSI escapes
// are carried across writes and do not count towards the column width.
func (w *logTrimWriter) wrapLinesLocked(data []byte) []byte {
	w.buf = w.buf[:0]

	maxX, _ := w.view.Size()
	if maxX <= 0 {
		maxX = logViewFallbackWidth
	}

	for len(data) > 0 {
		if data[0] == 0x1b {
			size := ansiSequenceLen(data)
			if size == 0 {
				w.pending = append(w.pending, data...)
				break
			}
			w.buf = append(w.buf, data[:size]...)
			data = data[size:]
			continue
		}

		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 && !utf8.FullRune(data) {
			w.pending = append(w.pending, data...)
			break
		}

		switch r {
		case '\n':
			w.buf = append(w.buf, '\n')
			w.lineCols = 0
		case '\r':
			w.buf = append(w.buf, '\r')
			w.lineCols = 0
		default:
			width := runewidth.RuneWidth(r)
			if width < 1 {
				width = 1
			}
			if w.lineCols > 0 && w.lineCols+width > maxX {
				w.buf = append(w.buf, '\n')
				w.lineCols = 0
			}
			w.buf = append(w.buf, data[:size]...)
			w.lineCols += width
		}

		data = data[size:]
	}

	return w.buf
}

// ansiSequenceLen returns the length of the escape sequence at the start of p,
// or 0 when the sequence is not complete yet.
func ansiSequenceLen(p []byte) int {
	if len(p) < 2 || p[0] != 0x1b {
		return 0
	}

	switch p[1] {
	case '[':
		for i := 2; i < len(p); i++ {
			if p[i] >= 0x40 && p[i] <= 0x7e {
				return i + 1
			}
		}
		return 0
	case ']':
		for i := 2; i < len(p); i++ {
			if p[i] == 0x07 {
				return i + 1
			}
			if p[i] == 0x1b && i+1 < len(p) && p[i+1] == '\\' {
				return i + 2
			}
		}
		return 0
	default:
		return 2
	}
}

// pinToBottomLocked makes the newest line visible. gocui's Autoscroll does this
// on draw, but pinning the origin here keeps the follow view at the bottom even
// when a draw is skipped, which otherwise leaves realtime logs stuck.
func (w *logTrimWriter) pinToBottomLocked() {
	if w.shouldPin != nil && !w.shouldPin() {
		return
	}

	_, height := w.view.Size()
	if height <= 0 {
		return
	}

	origin := w.view.ViewLinesHeight() - height
	if origin < 0 {
		origin = 0
	}

	_ = w.view.SetOriginY(origin)
}

func (w *logTrimWriter) trimLocked() {
	lines := w.view.BufferLines()

	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) <= logViewKeepLines {
		w.lines = len(lines)
		return
	}

	lines = lines[len(lines)-logViewKeepLines:]
	w.lines = len(lines)
	w.view.SetContent(strings.Join(lines, "\n"))
}
