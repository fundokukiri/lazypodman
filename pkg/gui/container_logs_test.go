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

	content := logs.snapshot()
	assert.Equal(t, "one\ntwo\nthree", content)
}

func TestLogAccumulatorKeepsLastLinesOnly(t *testing.T) {
	logs := newLogAccumulator(3)

	_, err := logs.Write([]byte(strings.Repeat("line\n", 10)))
	assert.NoError(t, err)

	content := logs.snapshot()
	assert.Equal(t, "line\nline\nline", content)
}

func TestLogAccumulatorTakeReturnsOnlyNewLines(t *testing.T) {
	logs := newLogAccumulator(10)

	_, _ = logs.Write([]byte("one\n"))

	initial, lines := logs.take()
	assert.True(t, initial)
	assert.Equal(t, []string{"one"}, lines)

	_, lines = logs.take()
	assert.Empty(t, lines, "nothing new since the last take")

	_, _ = logs.Write([]byte("two\nthree\n"))

	initial, lines = logs.take()
	assert.False(t, initial)
	assert.Equal(t, []string{"two", "three"}, lines)
}

func TestLogAccumulatorTakeAfterDrop(t *testing.T) {
	logs := newLogAccumulator(2)

	_, _ = logs.Write([]byte(strings.Repeat("line\n", 5)))

	initial, lines := logs.take()
	assert.True(t, initial)
	assert.Equal(t, []string{"line", "line"}, lines, "keeps only the cap")

	_, _ = logs.Write([]byte("next\n"))

	_, lines = logs.take()
	assert.Equal(t, []string{"next"}, lines)
}
