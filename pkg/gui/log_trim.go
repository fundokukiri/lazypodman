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
	// logWriteBatch bounds how much text is handed to gocui at once. A huge
	// horizontal log is wrapped in these batches instead of holding the view
	// lock for one enormous Write.
	logWriteBatch = 16 * 1024
)

// logTrimTarget is the part of *gocui.View the trimmer needs.
type logTrimTarget interface {
	Write(p []byte) (int, error)
	SetContent(string)
	BufferLines() []string
	Size() (int, int)
	SetOriginY(int) error
}

// logTrimWriter streams lines straight into the view, exactly like the old code,
// but wraps horizontal lines and caps the view's line count. A follow stream
// otherwise keeps every line in memory until the app is closed.
type logTrimWriter struct {
	mutex        sync.Mutex
	view         logTrimTarget
	lines        int
	lineCols     int
	lineBytes    int
	truncated    bool
	maxLineBytes int
	pending      []byte
	buf          []byte
	skipLF       bool
	shouldPin    func() bool
	stale        func() bool
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

	originalLen := len(p)
	if len(w.pending) > 0 {
		p = append(append([]byte(nil), w.pending...), p...)
		w.pending = nil
	}

	maxX, _ := w.view.Size()
	if maxX <= 0 {
		maxX = logViewFallbackWidth
	}

	for len(p) > 0 {
		batch := p
		if len(batch) > logWriteBatch {
			// Prefer a newline boundary so a batch does not end in the middle of
			// an ANSI sequence or a UTF-8 rune more often than necessary.
			batch = p[:logWriteBatch]
			if index := bytes.LastIndexByte(batch, '\n'); index > 0 {
				batch = batch[:index+1]
			}
		}

		out := w.wrapLocked(batch, maxX)
		if _, err := w.view.Write(out); err != nil {
			return 0, err
		}

		w.lines += bytes.Count(out, []byte("\n"))
		if w.lines > logViewTrimAt {
			w.trimLocked()
		}
		w.pinToBottomLocked()

		p = p[len(batch):]
		if len(w.pending) > 0 {
			p = append(append([]byte(nil), w.pending...), p...)
			w.pending = nil
		}
	}

	return originalLen, nil
}

// wrapLocked inserts newlines at the view width so a single enormous horizontal
// line does not force gocui to wrap and render megabytes over and over. All
// bytes are kept: the line is wrapped, not truncated. A bare carriage return is
// treated as a newline so progress-style logs do not overwrite each other.
func (w *logTrimWriter) endLineLocked() {
	if w.truncated {
		w.buf = append(w.buf, " ..."...)
	}
	w.buf = append(w.buf, '\n')
	w.lineCols = 0
	w.lineBytes = 0
	w.truncated = false
}

func (w *logTrimWriter) wrapLocked(data []byte, maxX int) []byte {
	w.buf = w.buf[:0]

	for len(data) > 0 {
		if data[0] == 0x1b {
			size := ansiSequenceLen(data)
			if size == 0 {
				w.pending = append(w.pending, data...)
				break
			}
			// Keep colour changes; drop cursor movement, erase-line and other
			// terminal control sequences. Progress output otherwise leaves
			// stray control bytes in the view.
			if isSGR(data[:size]) {
				w.buf = append(w.buf, data[:size]...)
			}
			data = data[size:]
			continue
		}

		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 && !utf8.FullRune(data) {
			w.pending = append(w.pending, data...)
			break
		}

		if (r < 0x20 && r != '\n' && r != '\r' && r != '\t') || r == 0x7f {
			data = data[size:]
			continue
		}

		if w.skipLF && r != '\n' {
			w.skipLF = false
		}

		switch r {
		case '\n':
			if w.skipLF {
				w.skipLF = false
			} else {
				w.endLineLocked()
			}
		case '\r':
			w.endLineLocked()
			w.skipLF = true
		default:
			if w.maxLineBytes > 0 && w.lineBytes >= w.maxLineBytes {
				w.truncated = true
				data = data[size:]
				continue
			}

			width := runewidth.RuneWidth(r)
			if width < 1 {
				width = 1
			}
			if w.lineCols > 0 && w.lineCols+width > maxX {
				w.buf = append(w.buf, '\n')
				w.lineCols = 0
			}

			if w.maxLineBytes > 0 && w.lineBytes+size > w.maxLineBytes {
				w.truncated = true
				data = data[size:]
				continue
			}

			w.buf = append(w.buf, data[:size]...)
			w.lineCols += width
			w.lineBytes += size
		}

		data = data[size:]
	}

	return w.buf
}

func isSGR(sequence []byte) bool {
	return len(sequence) >= 3 && sequence[1] == '[' && sequence[len(sequence)-1] == 'm'
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
		if len(p) > logWriteBatch {
			return 2 // malformed or unfinished escape: do not hold the stream
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
		if len(p) > logWriteBatch {
			return 2
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

	origin := w.lines - height
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
