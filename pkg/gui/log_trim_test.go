package gui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeLogTarget struct {
	content string
	originY int
	height  int
}

func (f *fakeLogTarget) Write(p []byte) (int, error) {
	f.content += string(p)

	return len(p), nil
}

func (f *fakeLogTarget) SetContent(s string) {
	f.content = s
}

func (f *fakeLogTarget) Size() (int, int) {
	return 80, f.height
}

func (f *fakeLogTarget) SetOriginY(y int) error {
	f.originY = y

	return nil
}

func (f *fakeLogTarget) ViewLinesHeight() int {
	return strings.Count(f.content, "\n")
}

func (f *fakeLogTarget) BufferLines() []string {
	if f.content == "" {
		return nil
	}

	return strings.Split(f.content, "\n")
}

func TestLogTrimWriterStreamsAndTrims(t *testing.T) {
	target := &fakeLogTarget{}
	writer := newLogTrimWriter(target, nil, nil)

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

func TestLogTrimWriterPinsTheNewestLines(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)

	_, err := writer.Write([]byte("one\ntwo\nthree\n"))
	assert.NoError(t, err)
	assert.Equal(t, 0, target.originY, "short logs start at the top")

	var builder strings.Builder
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&builder, "line %d\n", i)
	}

	_, err = writer.Write([]byte(builder.String()))
	assert.NoError(t, err)
	assert.Equal(t, 53-10, target.originY, "origin follows the newest line")
}

func TestLogTrimWriterDropsWritesOnceStale(t *testing.T) {
	target := &fakeLogTarget{}
	stale := false
	writer := newLogTrimWriter(target, nil, func() bool { return stale })

	_, err := writer.Write([]byte("before\n"))
	assert.NoError(t, err)
	assert.Equal(t, "before\n", target.content)

	stale = true

	_, err = writer.Write([]byte("after\n"))
	assert.NoError(t, err)
	assert.Equal(t, "before\n", target.content, "a cancelled task must not touch the view")
}

func TestLogTrimWriterDoesNotTrimSmallLogs(t *testing.T) {
	target := &fakeLogTarget{}
	writer := newLogTrimWriter(target, nil, nil)

	_, err := writer.Write([]byte(strings.Repeat("line\n", 100)))
	assert.NoError(t, err)
	assert.Equal(t, 100, len(target.BufferLines())-1)
}

func TestLogTrimWriterWrapsLongLinesWithoutLosingContent(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)

	long := strings.Repeat("x", logViewFallbackWidth*3)
	_, err := writer.Write([]byte(long + "\nshort\n"))
	assert.NoError(t, err)

	lines := target.BufferLines()
	assert.Equal(t, "short", lines[len(lines)-2])
	assert.Equal(t, long, strings.Join(lines[:len(lines)-2], ""), "wrapping keeps every byte")
	for _, line := range lines[:len(lines)-2] {
		assert.LessOrEqual(t, len(line), logViewFallbackWidth)
	}
}

func TestLogTrimWriterWrapsAcrossWrites(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)

	long := strings.Repeat("a", logViewFallbackWidth) + strings.Repeat("b", logViewFallbackWidth)
	_, err := writer.Write([]byte(long[:logViewFallbackWidth+7]))
	assert.NoError(t, err)
	_, err = writer.Write([]byte(long[logViewFallbackWidth+7:] + "\nnext\n"))
	assert.NoError(t, err)

	lines := target.BufferLines()
	assert.Equal(t, long, strings.Join(lines[:len(lines)-2], ""))
	assert.Equal(t, "next", lines[len(lines)-2])
}

func TestLogTrimWriterKeepsUtf8AcrossWrites(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)

	text := strings.Repeat("é", logViewFallbackWidth) + "ok\n"
	split := len(text) - len("ok\n") - 1
	_, err := writer.Write([]byte(text[:split]))
	assert.NoError(t, err)
	_, err = writer.Write([]byte(text[split:]))
	assert.NoError(t, err)

	assert.Contains(t, target.content, "ok")
	assert.Equal(t, strings.Count(text, "é"), strings.Count(target.content, "é"))
}
