package gui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTranscriptDetectorRecognisesAFullScreenProgram(t *testing.T) {
	assert.True(t, looksLikeTranscript([]byte("\x1b[?1049hhello")))
	assert.True(t, looksLikeTranscript([]byte("\x1b[2J\x1b[H\x1b[10;20Htext")))
	assert.True(t, looksLikeTranscript([]byte("\x1b[H\x1b[1A\x1b[3B\x1b[10;20Hrepaint")))
}

func TestTranscriptDetectorIgnoresCursorHidingOnly(t *testing.T) {
	assert.False(t, looksLikeTranscript([]byte("\x1b[?25lhello")))
	assert.True(t, looksLikeTranscript([]byte("\x1b[?2004hhello")))
}

func TestTranscriptDetectorIgnoresPlainColouredLogs(t *testing.T) {
	assert.False(t, looksLikeTranscript([]byte("\x1b[31mred\x1b[0m normal log line\n")))
	assert.False(t, looksLikeTranscript([]byte("progress: 50%\rprogress: 100%\n")))
}

// A progress bar erases its line and moves up one row. That is readable as plain
// text, and treating it as a full-screen program re-reads hundreds of lines of
// logs for nothing.
func TestTranscriptDetectorIgnoresProgressBars(t *testing.T) {
	assert.False(t, looksLikeTranscript([]byte("\x1b[2K\x1b[1A\x1b[K 45%\r")))
	assert.False(t, looksLikeTranscript([]byte("\x1b[1Gdownloading\x1b[K 12 MB\r")))
}

func TestTranscriptDetectorKeepsSplitSequences(t *testing.T) {
	detected := make(chan struct{}, 1)
	detector := newTranscriptDetector(func() { detected <- struct{}{} })

	_, _ = detector.Write([]byte("\x1b[2"))
	_, _ = detector.Write([]byte("J\x1b[H\x1b[3A"))

	select {
	case <-detected:
	case <-time.After(time.Second):
		t.Fatal("split transcript sequence was not detected")
	}
}

func TestTranscriptDetectorTriggersOnlyOnce(t *testing.T) {
	detected := make(chan struct{}, 2)
	detector := newTranscriptDetector(func() { detected <- struct{}{} })

	_, _ = detector.Write([]byte("\x1b[2J\x1b[H\x1b[3A"))
	_, _ = detector.Write([]byte("\x1b[2J\x1b[H\x1b[3A"))

	select {
	case <-detected:
	case <-time.After(time.Second):
		t.Fatal("transcript sequence was not detected")
	}

	select {
	case <-detected:
		t.Fatal("detector triggered more than once")
	case <-time.After(50 * time.Millisecond):
	}
}
