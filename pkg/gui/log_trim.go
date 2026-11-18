package gui

import (
	"bytes"
	"strings"
	"sync"
)

const (
	// logViewKeepLines is how many lines the Logs view keeps after a trim.
	logViewKeepLines = 10000
	// logViewTrimAt is when a trim happens. Trimming is a full re-render, so it
	// is done rarely (every 20000 lines) to keep the UI responsive.
	logViewTrimAt = 20000
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
// but caps the view's line count. A follow stream otherwise keeps every line in
// memory until the app is closed, which is what made RAM explode on long logs.
type logTrimWriter struct {
	mutex sync.Mutex
	view  logTrimTarget
	lines int
}

func newLogTrimWriter(view logTrimTarget) *logTrimWriter {
	return &logTrimWriter{view: view}
}

func (w *logTrimWriter) Write(p []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	n, err := w.view.Write(p)

	w.lines += bytes.Count(p[:n], []byte("\n"))
	if w.lines > logViewTrimAt {
		w.trimLocked()
	}

	w.pinToBottomLocked()

	return n, err
}

// pinToBottomLocked makes the newest line visible. gocui's Autoscroll does this
// on draw, but pinning the origin here keeps the follow view at the bottom even
// when a draw is skipped, which otherwise leaves realtime logs stuck.
func (w *logTrimWriter) pinToBottomLocked() {
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
