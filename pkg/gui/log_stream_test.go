package gui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeLogTarget struct {
	content string
}

func (f *fakeLogTarget) Write(p []byte) (int, error) {
	f.content += string(p)

	return len(p), nil
}

func (f *fakeLogTarget) SetContent(s string) {
	f.content = s
}

func (f *fakeLogTarget) BufferLines() []string {
	if f.content == "" {
		return nil
	}

	return strings.Split(f.content, "\n")
}

func TestLogStreamWriterBuffersFirstBurstThenStreams(t *testing.T) {
	target := &fakeLogTarget{}
	writer := newLogStreamWriter(target)

	_, err := writer.Write([]byte("one\ntwo\n"))
	assert.NoError(t, err)
	assert.Empty(t, target.content, "the first burst stays buffered")
	assert.True(t, writer.Buffering())

	writer.FlushInitial()
	assert.Equal(t, "one\ntwo\n", target.content)
	assert.False(t, writer.Buffering())

	_, err = writer.Write([]byte("three\n"))
	assert.NoError(t, err)
	assert.Equal(t, "one\ntwo\nthree\n", target.content, "later lines stream straight through")
}

func TestLogStreamWriterInitialBurstIsCapped(t *testing.T) {
	target := &fakeLogTarget{}
	writer := newLogStreamWriter(target)

	var builder strings.Builder
	for i := 0; i < logKeepLines+500; i++ {
		fmt.Fprintf(&builder, "line %d\n", i)
	}

	_, err := writer.Write([]byte(builder.String()))
	assert.NoError(t, err)
	writer.FlushInitial()

	lines := target.BufferLines()
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	assert.Equal(t, logKeepLines, len(lines))
	assert.Equal(t, fmt.Sprintf("line %d", logKeepLines+499), lines[logKeepLines-1])
}

func TestLogStreamWriterFlushesEarlyOnHugeBurst(t *testing.T) {
	target := &fakeLogTarget{}
	writer := newLogStreamWriter(target)

	_, err := writer.Write([]byte(strings.Repeat("x\n", logInitialBufferLimit/2+10)))
	assert.NoError(t, err)
	assert.False(t, writer.Buffering(), "a burst over the buffer limit flushes immediately")
	assert.NotEmpty(t, target.content)
}

func TestLogStreamWriterTrimsOldLines(t *testing.T) {
	target := &fakeLogTarget{}
	writer := newLogStreamWriter(target)

	writer.FlushInitial()

	var builder strings.Builder
	for i := 0; i < logTrimAt+100; i++ {
		fmt.Fprintf(&builder, "line %d\n", i)
	}

	_, err := writer.Write([]byte(builder.String()))
	assert.NoError(t, err)

	writer.TrimIfNeeded()

	lines := target.BufferLines()
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	assert.Equal(t, logKeepLines, len(lines))
	assert.Equal(t, fmt.Sprintf("line %d", logTrimAt+99), lines[len(lines)-1])

	writer.TrimIfNeeded()
	assert.Equal(t, logKeepLines, len(target.BufferLines()), "trim is a no-op until new lines arrive")
}
