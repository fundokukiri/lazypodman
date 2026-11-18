package gui

import (
	"bytes"
	"strings"
	"sync"
	"time"
)

const (
	// logFirstPaintWindow is how long the first burst of logs is collected before
	// it is written to the view in one go. Keeping it short preserves the old
	// streaming speed while avoiding a frame that paints from the top.
	logFirstPaintWindow = 150 * time.Millisecond
	// logInitialBufferLimit flushes early if the container dumps a lot of lines at
	// once, so a noisy container cannot allocate an unbounded buffer.
	logInitialBufferLimit = 2 << 20 // 2 MiB
	// logKeepLines is how many lines the view keeps. Follow mode would otherwise
	// grow the view (and RAM) until the app is closed.
	logKeepLines = 5000
	// logTrimAt triggers the trim above the keep limit, so trims are amortised.
	logTrimAt = 8000
)

// logTarget is the subset of *gocui.View the log writer needs.
type logTarget interface {
	Write(p []byte) (int, error)
	SetContent(string)
	BufferLines() []string
}

// logStreamWriter streams log lines into the view like the original code, but
// buffers the first burst for a short window so the tab opens at the bottom, and
// trims old lines so follow mode does not grow memory without bound.
type logStreamWriter struct {
	mutex       sync.Mutex
	view        logTarget
	buffer      bytes.Buffer
	buffering   bool
	lines       int
	trimPending bool
}

func newLogStreamWriter(view logTarget) *logStreamWriter {
	return &logStreamWriter{view: view, buffering: true}
}

func (w *logStreamWriter) Write(p []byte) (int, error) {
	w.mutex.Lock()

	if w.buffering {
		w.buffer.Write(p)
		w.lines += bytes.Count(p, []byte("\n"))

		// Flush early if the first burst is large: keeping it buffered would
		// double memory exactly when the container is noisy.
		if w.buffer.Len() >= logInitialBufferLimit {
			w.flushInitialLocked()
		}

		w.mutex.Unlock()

		return len(p), nil
	}

	w.mutex.Unlock()

	n, err := w.view.Write(p)

	w.mutex.Lock()
	w.lines += bytes.Count(p[:n], []byte("\n"))
	if w.lines > logTrimAt {
		w.trimPending = true
	}
	w.mutex.Unlock()

	return n, err
}

// FlushInitial writes the buffered first burst as one frame at the bottom.
func (w *logStreamWriter) FlushInitial() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.flushInitialLocked()
}

func (w *logStreamWriter) flushInitialLocked() {
	if !w.buffering {
		return
	}

	w.buffering = false

	raw := w.buffer.String()
	endedWithNewline := strings.HasSuffix(raw, "\n")
	content := strings.TrimSuffix(raw, "\n")
	w.buffer.Reset()

	lines := strings.Split(content, "\n")
	if content == "" {
		lines = nil
	}

	if len(lines) > logKeepLines {
		lines = lines[len(lines)-logKeepLines:]
	}

	w.lines = len(lines)

	rendered := strings.Join(lines, "\n")
	if endedWithNewline && rendered != "" {
		rendered += "\n"
	}

	w.view.SetContent(rendered)
}

// TrimIfNeeded drops the oldest lines once the view grows past the trim mark.
func (w *logStreamWriter) TrimIfNeeded() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.trimPending {
		return
	}

	w.trimPending = false

	lines := w.view.BufferLines()
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) <= logKeepLines {
		w.lines = len(lines)
		return
	}

	lines = lines[len(lines)-logKeepLines:]
	w.lines = len(lines)

	w.view.SetContent(strings.Join(lines, "\n"))
}

// Buffering reports whether the initial burst is still being collected.
func (w *logStreamWriter) Buffering() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	return w.buffering
}
