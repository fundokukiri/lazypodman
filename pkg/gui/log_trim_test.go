package gui

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeLogTarget struct {
	mutex   sync.Mutex
	content string
	originY int
	height  int
}

func (f *fakeLogTarget) Write(p []byte) (int, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.content += string(p)

	return len(p), nil
}

func (f *fakeLogTarget) SetContent(s string) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.content = s
}

func (f *fakeLogTarget) Content() string {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.content
}

func (f *fakeLogTarget) Buffer() string {
	return f.Content()
}

func (f *fakeLogTarget) OriginY() int {
	return f.originY
}

func (f *fakeLogTarget) Size() (int, int) {
	return 80, f.height
}

func (f *fakeLogTarget) SetOriginY(y int) error {
	f.originY = y

	return nil
}

func (f *fakeLogTarget) BufferLines() []string {
	f.mutex.Lock()
	defer f.mutex.Unlock()

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
	assert.Equal(t, "abcdefghij\nklmnop\nnext\n", target.content)
}

func TestLogTrimWriterKeepsStreamingAfterLineCap(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)
	writer.maxLineBytes = 10

	_, err := writer.Write([]byte(strings.Repeat("a", 25)))
	assert.NoError(t, err)
	_, err = writer.Write([]byte("Z\n"))
	assert.NoError(t, err)

	assert.Equal(t, 25, strings.Count(target.content, "a"))
	assert.Contains(t, target.content, "Z\n")
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

func TestLogTrimWriterTrimsByBytes(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)
	writer.maxBytes = 100

	for i := 0; i < 30; i++ {
		_, err := writer.Write([]byte(fmt.Sprintf("line-%02d-abcdefghij\n", i)))
		assert.NoError(t, err)
	}

	retained := strings.Join(target.BufferLines(), "\n")
	assert.LessOrEqual(t, len(retained), 150, "trim happens once the 1.5x threshold is crossed")
	assert.Contains(t, retained, "line-29")
	assert.NotContains(t, retained, "line-00")
}

// A reader who scrolled up keeps the lines they are on: trimming would slide the
// text out from under them, so the budgets only grow.
func TestLogTrimWriterKeepsLinesWhileTheReaderScrollsUp(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, func() bool { return false }, nil)

	var builder strings.Builder
	for i := 0; i < logViewTrimAt+500; i++ {
		fmt.Fprintf(&builder, "line %d\n", i)
	}

	_, err := writer.Write([]byte(builder.String()))
	assert.NoError(t, err)

	lines := target.BufferLines()
	assert.Equal(t, logViewTrimAt+501, len(lines), "nothing was trimmed while scrolled up")
}

func TestLogTrimWriterCapsMemoryWhileScrollStaysUp(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, func() bool { return false }, nil)

	var builder strings.Builder
	for i := 0; i < logViewTrimAt*logViewPausedScale+500; i++ {
		fmt.Fprintf(&builder, "line %d\n", i)
	}

	_, err := writer.Write([]byte(builder.String()))
	assert.NoError(t, err)

	lines := target.BufferLines()
	assert.Equal(t, logViewKeepLines, len(lines), "the paused budget still bounds memory")
}

func TestLogTrimWriterKeepsTheScrollPositionAcrossATrim(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, func() bool { return false }, nil)

	var builder strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&builder, "line %d\n", i)
	}

	_, err := writer.Write([]byte(builder.String()))
	assert.NoError(t, err)

	// The reader is looking at line 40; keeping 80 lines drops the first 20.
	assert.NoError(t, target.SetOriginY(40))

	writer.keepLines = 80
	writer.trimAtLines = 1
	writer.trimLocked()

	assert.Equal(t, 20, target.originY, "the same line stays under the cursor")
}

// Every kept line costs far more memory than its text once gocui renders it, so
// the view budget must not grow with ctrl+l. The fetch cap (logs.fullTail) is
// what makes full history cheap.
func TestFullHistoryDoesNotRaiseTheViewBudget(t *testing.T) {
	gui := logsGuiWithTail("20")
	gui.Config.UserConfig.Logs.MaxBytes = 1024
	gui.fullLogs = map[string]bool{"abc": true}

	writer := newLogTrimWriter(&fakeLogTarget{}, nil, nil)
	gui.configureLogWriter(writer)

	assert.Equal(t, 1024, writer.maxBytes)
	assert.Equal(t, logViewKeepLines, writer.keepLines)
	assert.Equal(t, logViewTrimAt, writer.trimAtLines)
}

func TestLogTrimWriterHoldsInitialOutput(t *testing.T) {
	target := &fakeLogTarget{height: 2}
	writer := newLogTrimWriter(target, nil, nil)
	writer.holdInitialFor(time.Hour)
	defer writer.Close()

	_, err := writer.Write([]byte("one\ntwo\nthree\n"))
	assert.NoError(t, err)
	assert.Empty(t, target.content, "initial output stays hidden")

	writer.flushInitial()
	assert.Equal(t, "one\ntwo\nthree\n", target.content)
	assert.Equal(t, 1, target.originY)
}

func TestLogTrimWriterCloseFlushesInitialOutput(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)
	writer.holdInitialFor(time.Hour)

	_, err := writer.Write([]byte("one\n"))
	assert.NoError(t, err)
	assert.Empty(t, target.content)

	writer.Close()
	assert.Equal(t, "one\n", target.content)
}

func TestLogTrimWriterDropsStaleInitialOutput(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	stale := false
	writer := newLogTrimWriter(target, nil, func() bool { return stale })
	writer.holdInitialFor(time.Hour)
	defer writer.Close()

	_, err := writer.Write([]byte("one\n"))
	assert.NoError(t, err)
	stale = true

	writer.flushInitial()
	assert.Empty(t, target.content)
}

func TestLogTrimWriterStreamsAfterInitialFlush(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)
	writer.holdInitialFor(time.Hour)
	defer writer.Close()

	_, err := writer.Write([]byte("one\n"))
	assert.NoError(t, err)
	writer.flushInitial()

	_, err = writer.Write([]byte("two\n"))
	assert.NoError(t, err)
	assert.Equal(t, "one\ntwo\n", target.content)
}

func TestLogTrimWriterBoundsTheHiddenInitialBurst(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)
	writer.maxBytes = 100
	writer.holdInitialFor(time.Hour)
	defer writer.Close()

	_, err := writer.Write([]byte(strings.Repeat("0123456789\n", 50)))
	assert.NoError(t, err)
	writer.flushInitial()

	retained := strings.Join(target.BufferLines(), "\n")
	assert.LessOrEqual(t, len(retained), 150)
	assert.Contains(t, retained, "0123456789")
}

func TestLogTrimWriterFlushesInitialOnTimer(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)
	writer.holdInitialFor(5 * time.Millisecond)
	defer writer.Close()

	_, err := writer.Write([]byte("one\ntwo\n"))
	assert.NoError(t, err)
	assert.Eventually(t, func() bool { return target.Content() == "one\ntwo\n" }, time.Second, 5*time.Millisecond)
}
