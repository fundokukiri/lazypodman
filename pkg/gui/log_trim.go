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
	// logViewLineMaxBytes bounds one logical log line. A minified or horizontal
	// log can otherwise put megabytes on a single line despite a small tail,
	// which makes gocui wrap and render for a very long time.
	logViewLineMaxBytes = 4096
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
	mutex         sync.Mutex
	view          logTrimTarget
	lines         int
	lineBytes     int
	lineTruncated bool
	buf           []byte
	shouldPin     func() bool
	stale         func() bool
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

	out := w.capLinesLocked(p)
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

// capLinesLocked truncates individual lines to logViewLineMaxBytes before they
// reach gocui. The stream itself is still consumed, so follow mode continues
// normally; only the enormous display line is shortened.
func (w *logTrimWriter) capLinesLocked(p []byte) []byte {
	w.buf = w.buf[:0]

	for len(p) > 0 {
		index := bytes.IndexByte(p, '\n')
		if index < 0 {
			w.appendLineSegmentLocked(&w.buf, p)
			break
		}

		w.appendLineSegmentLocked(&w.buf, p[:index])
		if w.lineTruncated {
			w.buf = append(w.buf, " ..."...)
		}
		w.buf = append(w.buf, '\n')
		w.lineBytes = 0
		w.lineTruncated = false
		p = p[index+1:]
	}

	return w.buf
}

func (w *logTrimWriter) appendLineSegmentLocked(out *[]byte, segment []byte) {
	if w.lineTruncated {
		return
	}

	remaining := logViewLineMaxBytes - w.lineBytes
	if len(segment) <= remaining {
		*out = append(*out, segment...)
		w.lineBytes += len(segment)
		return
	}

	if remaining > 0 {
		*out = append(*out, segment[:remaining]...)
		w.lineBytes += remaining
	}

	w.lineTruncated = true
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
