package commands

import (
	"strings"
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestLogTranscriptReplaysTheFinalScreen(t *testing.T) {
	transcript := NewLogTranscript(20, 4)
	_, _ = transcript.Write([]byte("old screen"))
	_, _ = transcript.Write([]byte("\x1b[2J\x1b[H\x1b[32mgreen\x1b[0m\r\nsecond"))

	rows := transcript.Snapshot()
	assert.Equal(t, "green", utils.Decolorise(strings.TrimRight(rows[0], " ")))
	assert.Equal(t, "second", utils.Decolorise(strings.TrimRight(rows[1], " ")))
	assert.Contains(t, rows[0], "\x1b[0;32m")
}

func TestLogTranscriptKeepsCursorMovement(t *testing.T) {
	transcript := NewLogTranscript(20, 3)
	_, _ = transcript.Write([]byte("loading..."))

	rows := transcript.Snapshot()
	assert.Equal(t, "loading...", strings.TrimRight(rows[0], " "))
}

func TestLogTranscriptResize(t *testing.T) {
	transcript := NewLogTranscript(5, 2)
	transcript.Resize(10, 3)
	_, _ = transcript.Write([]byte("abcdefghij"))

	rows := transcript.Snapshot()
	assert.Len(t, rows, 3)
	assert.Equal(t, "abcdefghij", strings.TrimRight(rows[0], " "))
}
