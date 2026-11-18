package gui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogAccumulatorJoinsWritesAndPartialLines(t *testing.T) {
	logs := newLogAccumulator(10)

	_, err := logs.Write([]byte("one\ntw"))
	assert.NoError(t, err)
	_, err = logs.Write([]byte("o\nthree\n"))
	assert.NoError(t, err)

	content, _ := logs.snapshot()
	assert.Equal(t, "one\ntwo\nthree", content)
}

func TestLogAccumulatorKeepsLastLinesOnly(t *testing.T) {
	logs := newLogAccumulator(3)

	_, err := logs.Write([]byte(strings.Repeat("line\n", 10)))
	assert.NoError(t, err)

	content, _ := logs.snapshot()
	assert.Equal(t, "line\nline\nline", content)
}

func TestLogAccumulatorVersionChangesOnlyWithContent(t *testing.T) {
	logs := newLogAccumulator(10)

	_, _ = logs.Write([]byte("hello\n"))
	_, first := logs.snapshot()

	content, second := logs.snapshot()
	assert.Equal(t, first, second, "reading twice must not change the version")
	assert.Equal(t, "hello", content)

	_, _ = logs.Write([]byte("world\n"))
	_, third := logs.snapshot()
	assert.NotEqual(t, first, third)
}
