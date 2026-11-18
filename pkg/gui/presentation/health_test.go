package presentation

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"

	"github.com/docker/docker/api/types/container"
)

// The data below is SYNTHETIC: no fixture here was captured from a real Podman.
func syntheticProbe(exitCode int, output string) *container.HealthcheckResult {
	start := time.Date(2026, 11, 18, 10, 30, 0, 0, time.Local)

	return &container.HealthcheckResult{
		Start:    start,
		End:      start.Add(2 * time.Second),
		ExitCode: exitCode,
		Output:   output,
	}
}

func TestHealthRowsNewestFirstAndCapped(t *testing.T) {
	log := make([]*container.HealthcheckResult, 0, 25)
	for i := range 25 {
		log = append(log, syntheticProbe(i%3, fmt.Sprintf("probe %d", i)))
	}

	rows := HealthRows(log)

	assert.Len(t, rows, MaxHealthRows)
	assert.Equal(t, "probe 24", rows[0].Output, "newest entry comes first")
	assert.Equal(t, "probe 5", rows[len(rows)-1].Output)
}

func TestHealthRowsSkipsNilEntries(t *testing.T) {
	rows := HealthRows([]*container.HealthcheckResult{nil, syntheticProbe(0, "ok"), nil})

	assert.Len(t, rows, 1)
	assert.Equal(t, "ok", rows[0].Output)
	assert.Empty(t, HealthRows(nil))
}

func TestHealthRowsTimestamps(t *testing.T) {
	start := time.Date(2026, 11, 18, 10, 30, 0, 0, time.Local)
	end := time.Date(2026, 11, 18, 10, 30, 3, 0, time.Local)

	rows := HealthRows([]*container.HealthcheckResult{
		{Start: start, End: end},
		{Start: start},
		{},
	})

	assert.Equal(t, "10:30:03", rows[2].Time, "end time wins")
	assert.Equal(t, "10:30:00", rows[1].Time, "start time is the fallback")
	assert.Equal(t, "--:--:--", rows[0].Time, "zero times stay obvious")
}

func TestTruncateHealthOutput(t *testing.T) {
	t.Run("multiline is flattened", func(t *testing.T) {
		assert.Equal(t, "a⏎b→c", TruncateHealthOutput("a\nb\tc\n"))
	})

	t.Run("empty stays empty", func(t *testing.T) {
		assert.Equal(t, "", TruncateHealthOutput(""))
		assert.Equal(t, "", TruncateHealthOutput("\n\n"))
	})

	t.Run("ascii is capped", func(t *testing.T) {
		out := TruncateHealthOutput(strings.Repeat("a", 250))

		assert.Equal(t, MaxHealthOutputRunes+1, utf8.RuneCountInString(out))
		assert.True(t, strings.HasSuffix(out, "…"))
	})

	t.Run("multibyte is not split", func(t *testing.T) {
		out := TruncateHealthOutput(strings.Repeat("ệ", 250))

		assert.True(t, utf8.ValidString(out))
		assert.Equal(t, MaxHealthOutputRunes+1, utf8.RuneCountInString(out))
	})

	t.Run("exactly at the cap has no ellipsis", func(t *testing.T) {
		out := TruncateHealthOutput(strings.Repeat("a", MaxHealthOutputRunes))

		assert.Equal(t, MaxHealthOutputRunes, utf8.RuneCountInString(out))
		assert.False(t, strings.HasSuffix(out, "…"))
	})
}

func TestHealthExitColor(t *testing.T) {
	assert.Equal(t, color.FgGreen, HealthExitColor(0))
	assert.Equal(t, color.FgRed, HealthExitColor(1))
	assert.Equal(t, color.FgRed, HealthExitColor(2))
	assert.Equal(t, color.FgYellow, HealthExitColor(125))
	assert.Equal(t, color.FgYellow, HealthExitColor(-1))
}

func TestHealthRowCells(t *testing.T) {
	row := HealthRow{Time: "10:30:03", ExitCode: 1, Output: ""}

	assert.Equal(t, []string{"10:30:03", "1", "(no output)"}, HealthRowCells(row, "(no output)"))

	row.Output = "boom"
	assert.Equal(t, []string{"10:30:03", "1", "boom"}, HealthRowCells(row, "(no output)"))
}
