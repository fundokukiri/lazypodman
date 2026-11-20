package gui

import (
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestFindLogMatchesIsCaseInsensitiveAndStripsColours(t *testing.T) {
	lines := []string{
		"\x1b[31mTimeout\x1b[0m while reaching redis",
		"all good",
		"timeout again",
		"TIMEOUT third",
	}

	matches := findLogMatches(lines, "timeout", 2)

	assert.Equal(t, []string{"Timeout while reaching redis", "timeout again"}, matches)
}

func TestFindLogMatchesKeepsLongLinesReadable(t *testing.T) {
	long := "timeout " + string(make([]byte, logSearchMaxMatchLine+20))

	matches := findLogMatches([]string{long}, "timeout", 1)

	assert.Len(t, matches, 1)
	assert.LessOrEqual(t, len(matches[0]), logSearchMaxMatchLine+len("…"))
}

func TestLogSearchReportGroupsMatchesAndErrors(t *testing.T) {
	results := []logSearchResult{
		{Container: &commands.Container{Name: "api"}, Matches: []string{"timeout 1", "timeout 2"}},
		{Container: &commands.Container{Name: "worker"}, Err: assert.AnError},
		{Container: &commands.Container{Name: "db"}},
	}

	report := utils.Decolorise(logSearchReport(translations(), "timeout", results, 3))

	assert.Contains(t, report, "api (2)")
	assert.Contains(t, report, "timeout 1")
	assert.Contains(t, report, "worker: "+translations().LogSearchError)
	assert.NotContains(t, report, "db (")
	assert.Contains(t, report, translations().LogSearchHint)
}

func TestLogSearchReportSaysWhenNothingMatches(t *testing.T) {
	report := utils.Decolorise(logSearchReport(translations(), "needle", nil, 0))

	assert.Contains(t, report, translations().LogSearchNoMatches)
}
