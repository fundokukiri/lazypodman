package presentation

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/fatih/color"
)

// MaxHealthRows caps the probe history we display. Podman keeps a bounded log;
// we show what inspect returns, with this as an upper bound, not a promise.
const MaxHealthRows = 20

// MaxHealthOutputRunes caps one probe's output so a chatty command cannot flood
// the view or the clipboard.
const MaxHealthOutputRunes = 200

const noProbeTime = "--:--:--"

// HealthRow is one probe result ready for the Health tab.
type HealthRow struct {
	Time     string
	ExitCode int
	Output   string
}

// HealthRows converts the podman health log into display rows, newest first.
// Podman stores entries oldest-first and bounded; nil entries are skipped.
func HealthRows(log []*container.HealthcheckResult) []HealthRow {
	rows := make([]HealthRow, 0, min(len(log), MaxHealthRows))

	for i := len(log) - 1; i >= 0 && len(rows) < MaxHealthRows; i-- {
		entry := log[i]
		if entry == nil {
			continue
		}

		rows = append(rows, HealthRow{
			Time:     healthTimestamp(entry.End, entry.Start),
			ExitCode: entry.ExitCode,
			Output:   TruncateHealthOutput(entry.Output),
		})
	}

	return rows
}

// healthTimestamp prefers the probe's end time and falls back to its start.
func healthTimestamp(end, start time.Time) string {
	for _, timestamp := range []time.Time{end, start} {
		if !timestamp.IsZero() {
			return timestamp.Local().Format("15:04:05")
		}
	}

	return noProbeTime
}

// TruncateHealthOutput flattens a probe's output onto one line and truncates it
// by runes (never splitting a multi-byte character). An empty result stays empty:
// the caller decides which placeholder to show.
func TruncateHealthOutput(output string) string {
	output = strings.TrimRight(output, "\r\n")

	var builder strings.Builder

	count := 0
	for _, char := range output {
		if count >= MaxHealthOutputRunes {
			builder.WriteString("…")

			break
		}

		switch char {
		case '\r':
			continue
		case '\n':
			builder.WriteString("⏎")
		case '\t':
			builder.WriteString("→")
		default:
			builder.WriteRune(char)
		}

		count++
	}

	return builder.String()
}

// HealthExitColor colours a probe's exit code. Only 0 (healthy) is verified;
// 1 and 2 come from Podman's documentation and are UNVERIFIED.
func HealthExitColor(exitCode int) color.Attribute {
	switch exitCode {
	case 0:
		return color.FgGreen
	case 1, 2:
		return color.FgRed
	default:
		return color.FgYellow
	}
}

// HealthRowCells returns the cells of one Health tab row.
func HealthRowCells(row HealthRow, noOutputPlaceholder string) []string {
	output := row.Output
	if output == "" {
		output = noOutputPlaceholder
	}

	return []string{
		row.Time,
		fmt.Sprintf("%d", row.ExitCode),
		output,
	}
}
