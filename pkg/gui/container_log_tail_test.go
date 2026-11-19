package gui

import (
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/config"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func logsGuiWithTail(tail string) *Gui {
	translations := i18n.GetTranslationSets()["en"]

	return &Gui{
		Config: &config.AppConfig{UserConfig: &config.UserConfig{Logs: config.LogsConfig{Tail: tail}}},
		Tr:     &translations,
	}
}

func TestContainerLogTailDefaultsToConfiguredTail(t *testing.T) {
	gui := logsGuiWithTail("200")
	container := &commands.Container{ID: "abc"}

	assert.Equal(t, "200", gui.containerLogTail(container))
	assert.False(t, gui.logsFullFor(container))
}

func TestContainerLogTailSwitchToAllAndBack(t *testing.T) {
	gui := logsGuiWithTail("200")
	container := &commands.Container{ID: "abc"}

	gui.toggleFullLogs(container)

	assert.True(t, gui.logsFullFor(container))
	assert.Equal(t, "", gui.containerLogTail(container), "empty tail means the whole history")

	gui.toggleFullLogs(container)

	assert.False(t, gui.logsFullFor(container))
	assert.Equal(t, "200", gui.containerLogTail(container))
}

func TestLogsOptionsOfferFullHistoryToggle(t *testing.T) {
	gui := logsGuiWithTail("200")

	options := gui.logsOptionsMap()
	assert.Equal(t, gui.Tr.AllLogs, options["ctrl+l"])
	assert.Equal(t, gui.Tr.JumpToLatest, options["G"])
	assert.Contains(t, options, "PgUp/PgDn")
}

func TestDefaultLogsTailIsBounded(t *testing.T) {
	assert.Equal(t, "20", config.GetDefaultConfig().Logs.Tail)
}

func TestDefaultLogsLineCapIsBounded(t *testing.T) {
	assert.Equal(t, 4096, config.GetDefaultConfig().Logs.MaxLineLength)
}
