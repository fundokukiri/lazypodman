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

func TestLogTrimWriterStreamsAndTrims(t *testing.T) {
	target := &fakeLogTarget{}
	writer := newLogTrimWriter(target)

	_, err := writer.Write([]byte("one\n"))
	assert.NoError(t, err)
	assert.Equal(t, "one\n", target.content, "lines are written immediately, as before")

	var builder strings.Builder
	for i := 0; i < logViewTrimAt+50; i++ {
		fmt.Fprintf(&builder, "line %d\n", i)
	}

	_, err = writer.Write([]byte(builder.String()))
	assert.NoError(t, err)

	lines := target.BufferLines()
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	assert.Equal(t, logViewKeepLines, len(lines))
	assert.Equal(t, fmt.Sprintf("line %d", logViewTrimAt+49), lines[len(lines)-1], "keeps the newest lines")
}

func TestLogTrimWriterDoesNotTrimSmallLogs(t *testing.T) {
	target := &fakeLogTarget{}
	writer := newLogTrimWriter(target)

	_, err := writer.Write([]byte(strings.Repeat("line\n", 100)))
	assert.NoError(t, err)
	assert.Equal(t, 100, len(target.BufferLines())-1)
}
