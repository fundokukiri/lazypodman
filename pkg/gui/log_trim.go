package gui

import (
	"bytes"
	"strings"
	"sync"
	"time"
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
	// logInitialBytes bounds the hidden first burst even when the user disables
	// the normal byte budget. A stream can otherwise fill memory before the
	// first render.
	logInitialBytes = 1024 * 1024
	// logInitialHold is short enough to feel immediate and long enough to hide
	// the initial tail arriving in several chunks.
	logInitialHold = 120 * time.Millisecond
	// logViewPausedScale multiplies both budgets while the reader has scrolled
	// up: keeping the window still beats keeping it small, but memory still has
	// a ceiling.
	logViewPausedScale = 4
	// logViewFullHistoryScale multiplies both budgets when the user asked for
	// the whole history with ctrl+l. Keeping only the newest megabyte of a
	// multi-megabyte history makes scrolling back through it pointless.
	logViewFullHistoryScale = 10
)

// logTrimTarget is the part of *gocui.View the trimmer needs.
type logTrimTarget interface {
	Write(p []byte) (int, error)
	SetContent(string)
	BufferLines() []string
	Size() (int, int)
	OriginY() int
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
	maxLineBytes int
	bytes        int
	maxBytes     int
	keepLines    int
	trimAtLines  int
	initial      []byte
	initialTimer *time.Timer
	flushed      bool
	pending      []byte
	buf          []byte
	skipLF       bool
	shouldPin    func() bool
	stale        func() bool
}

func newLogTrimWriter(view logTrimTarget, shouldPin func() bool, stale func() bool) *logTrimWriter {
	return &logTrimWriter{
		view:        view,
		shouldPin:   shouldPin,
		stale:       stale,
		flushed:     true,
		keepLines:   logViewKeepLines,
		trimAtLines: logViewTrimAt,
	}
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
		// A tab switch must not wait for a multi-batch log chunk to finish.
		if w.stale != nil && w.stale() {
			return originalLen, nil
		}

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
		if !w.flushed {
			w.appendInitialLocked(out)
		} else {
			if _, err := w.view.Write(out); err != nil {
				return 0, err
			}

			w.bytes += len(out)
			w.lines += bytes.Count(out, []byte("\n"))
			if w.needsTrimLocked() {
				w.trimLocked()
			}
			w.pinToBottomLocked()
		}

		p = p[len(batch):]
		// Only carry pending bytes into the next batch when there is more input.
		// Otherwise an incomplete UTF-8 rune or escape sequence at the end of the
		// stream would be re-queued forever.
		if len(p) > 0 && len(w.pending) > 0 {
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
	w.buf = append(w.buf, '\n')
	w.lineCols = 0
	w.lineBytes = 0
}

// breakLongLineLocked starts a new display line once a single logical line has
// reached the configured cap. The text is not dropped; this keeps a container
// that writes one endless line readable and keeps new output arriving.
func (w *logTrimWriter) breakLongLineLocked() {
	w.buf = append(w.buf, '\n')
	w.lineCols = 0
	w.lineBytes = 0
}

func (w *logTrimWriter) wrapLocked(data []byte, maxX int) []byte {
	w.buf = w.buf[:0]

	for len(data) > 0 {
		if data[0] == 0x1b {
			size := ansiSequenceLen(data)
			if size == 0 {
				// Escape sequences are short. Do not let a malformed one (or a
				// binary log that happens to contain ESC) sit in pending forever.
				if len(data) > 64 {
					data = data[1:]
					continue
				}
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
				w.breakLongLineLocked()
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
				w.breakLongLineLocked()
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

func (w *logTrimWriter) needsTrimLocked() bool {
	trimAt := w.trimAtLines
	maxBytes := w.maxBytes

	// While the reader is scrolled up, the newest lines are not the ones on
	// screen: trimming would slide the text out from under them. Give both
	// budgets room to spare instead of freezing or growing without bound.
	if w.shouldPin != nil && !w.shouldPin() {
		trimAt *= logViewPausedScale
		maxBytes *= logViewPausedScale
	}

	if w.lines > trimAt {
		return true
	}

	return w.maxBytes > 0 && w.bytes > maxBytes+maxBytes/2
}

func (w *logTrimWriter) trimLocked() {
	originY := w.view.OriginY()
	lines := w.view.BufferLines()

	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	start := len(lines)
	total := 0
	for i := len(lines) - 1; i >= 0; i-- {
		cost := len(lines[i]) + 1
		if i != len(lines)-1 {
			if len(lines)-i > w.keepLines {
				break
			}
			if w.maxBytes > 0 && total+cost > w.maxBytes {
				break
			}
		}
		total += cost
		start = i
	}

	if start == 0 {
		w.lines = len(lines)
		w.bytes = total
		return
	}

	lines = lines[start:]
	content := strings.Join(lines, "\n")
	w.view.SetContent(content)
	w.lines = len(lines)
	w.bytes = len(content)

	// SetContent rewinds the origin. A reader who scrolled up is looking at a
	// fixed line, so keep that line under their cursor as the top falls off.
	if w.shouldPin != nil && !w.shouldPin() {
		if origin := originY - start; origin > 0 {
			_ = w.view.SetOriginY(origin)
		}
	}
}

// holdInitialFor keeps the first burst out of the view for delay. Podman can
// deliver the initial tail in many chunks; writing each one makes the view
// visibly scroll from the top. The timer writes the bounded burst once, at the
// bottom.
func (w *logTrimWriter) holdInitialFor(delay time.Duration) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.initialTimer != nil {
		return
	}

	w.flushed = false
	w.initialTimer = time.AfterFunc(delay, w.flushInitial)
}

func (w *logTrimWriter) appendInitialLocked(out []byte) {
	w.initial = append(w.initial, out...)

	budget := w.maxBytes
	if budget <= 0 {
		budget = logInitialBytes
	}
	limit := budget + budget/2
	if len(w.initial) <= limit {
		return
	}

	// Drop complete old lines so the hidden buffer does not itself grow without
	// bound. Wrapped output always contains newlines unless the stream ended in
	// the middle of a line.
	excess := len(w.initial) - budget
	start := 0
	if index := bytes.IndexByte(w.initial[excess:], '\n'); index >= 0 {
		start = excess + index + 1
	}
	w.initial = append([]byte(nil), w.initial[start:]...)
}

func (w *logTrimWriter) flushInitial() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.flushInitialLocked()
}

func (w *logTrimWriter) flushInitialLocked() {
	if w.flushed {
		return
	}
	w.flushed = true

	if w.initialTimer != nil {
		w.initialTimer.Stop()
		w.initialTimer = nil
	}

	if w.stale != nil && w.stale() {
		w.initial = nil
		return
	}

	if len(w.initial) == 0 {
		return
	}

	if _, err := w.view.Write(w.initial); err != nil {
		w.initial = nil
		return
	}

	w.bytes += len(w.initial)
	w.lines += bytes.Count(w.initial, []byte("\n"))
	w.initial = nil

	if w.needsTrimLocked() {
		w.trimLocked()
	}
	w.pinToBottomLocked()
}

// Close flushes the hidden initial burst when the log stream ends within the
// debounce window.
func (w *logTrimWriter) Close() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.initialTimer != nil {
		w.initialTimer.Stop()
		w.initialTimer = nil
	}
	w.flushInitialLocked()
}
