package gui

import (
	"fmt"
	"strings"
	"testing"
	"time"

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

func TestLogTrimWriterWrapsHorizontalLines(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)

	long := strings.Repeat("x", 200)
	_, err := writer.Write([]byte(long + "\nnext\n"))
	assert.NoError(t, err)

	lines := target.BufferLines()
	assert.Equal(t, "next", lines[len(lines)-2])
	assert.Equal(t, long, strings.Join(lines[:len(lines)-2], ""), "wrapping keeps every byte")
	for _, line := range lines[:len(lines)-2] {
		assert.LessOrEqual(t, len(line), 80)
	}
}

func TestLogTrimWriterTurnsCarriageReturnsIntoSeparateLines(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)

	_, err := writer.Write([]byte("progress 1\rprogress 2\r\nfinal\n"))
	assert.NoError(t, err)
	assert.Equal(t, "progress 1\nprogress 2\nfinal\n", target.content)
}

func TestLogTrimWriterReturnsTheOriginalLength(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)

	long := strings.Repeat("x", logWriteBatch*2)
	n, err := writer.Write([]byte(long))
	assert.NoError(t, err)
	assert.Equal(t, len(long), n)
}

func TestLogTrimWriterCapsConfiguredLineLength(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)
	writer.maxLineBytes = 10

	_, err := writer.Write([]byte("abcdefghijklmnop\nnext\n"))
	assert.NoError(t, err)
	assert.Equal(t, "abcdefghij ...\nnext\n", target.content)
}

func TestLogTrimWriterKeepsFullLinesWhenCapIsZero(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)
	writer.maxLineBytes = 0

	long := strings.Repeat("x", 5000)
	_, err := writer.Write([]byte(long + "\n"))
	assert.NoError(t, err)
	assert.Equal(t, long, strings.Join(target.BufferLines()[:len(target.BufferLines())-1], ""))
}

func TestLogTrimWriterDropsTerminalControlSequences(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)

	_, err := writer.Write([]byte("\x1b[2K\x1b[1Gprogress 1\r\x1b[2K\x1b[1Gprogress 2\n"))
	assert.NoError(t, err)
	assert.Equal(t, "progress 1\nprogress 2\n", target.content)
}

func TestLogTrimWriterKeepsColorSequences(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)

	_, err := writer.Write([]byte("\x1b[31mred\x1b[0m\n"))
	assert.NoError(t, err)
	assert.Equal(t, "\x1b[31mred\x1b[0m\n", target.content)
}

func TestLogTrimWriterDoesNotSpinOnIncompleteUtf8(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)

	_, err := writer.Write([]byte{0xc3})
	assert.NoError(t, err)
	_, err = writer.Write([]byte{0xa9, '\n'})
	assert.NoError(t, err)
	assert.Equal(t, "é\n", target.content)
}

func TestLogTrimWriterDoesNotSpinOnIncompleteAnsi(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)

	_, err := writer.Write([]byte("\x1b[31"))
	assert.NoError(t, err)
	_, err = writer.Write([]byte("mred\n"))
	assert.NoError(t, err)
	assert.Equal(t, "\x1b[31mred\n", target.content)
}

func TestLogTrimWriterDoesNotSpinOnLongMalformedEscape(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)

	malformed := append([]byte("\x1b["), strings.Repeat("1", logWriteBatch*2)...)
	done := make(chan struct{})
	go func() {
		_, _ = writer.Write(malformed)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Write did not return for a malformed escape sequence")
	}
}
