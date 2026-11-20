package gui

import (
	"testing"

	"github.com/docker/docker/api/types/container"
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
	assert.Equal(t, gui.Tr.TranscriptLogs, options["ctrl+t"])
	assert.Equal(t, gui.Tr.JumpToLatest, options["G"])
	assert.Contains(t, options, "PgUp/PgDn")
}

func TestTranscriptLogsToggleIsPerContainer(t *testing.T) {
	gui := logsGuiWithTail("200")
	first := &commands.Container{ID: "first"}
	second := &commands.Container{ID: "second"}

	gui.toggleTranscriptLogs(first)

	assert.True(t, gui.transcriptLogsFor(first))
	assert.False(t, gui.transcriptLogsFor(second))

	gui.toggleTranscriptLogs(first)

	assert.False(t, gui.transcriptLogsFor(first))
}

func TestTranscriptLogTailKeepsEnoughHistoryToRebuildAScreen(t *testing.T) {
	gui := logsGuiWithTail("20")
	container := &commands.Container{ID: "abc"}

	assert.Equal(t, "5000", gui.transcriptLogTail(container))

	gui.Config.UserConfig.Logs.Tail = "9000"
	assert.Equal(t, "9000", gui.transcriptLogTail(container))

	gui.Config.UserConfig.Logs.Tail = ""
	assert.Equal(t, "", gui.transcriptLogTail(container), "an empty tail still means everything")
}

func TestDefaultLogsTailIsBounded(t *testing.T) {
	assert.Equal(t, "20", config.GetDefaultConfig().Logs.Tail)
}

func TestDefaultLogsLineCapIsBounded(t *testing.T) {
	assert.Equal(t, 4096, config.GetDefaultConfig().Logs.MaxLineLength)
}

func TestContainerLogSinceIgnoresAgeForStoppedContainers(t *testing.T) {
	gui := logsGuiWithTail("20")
	gui.Config.UserConfig.Logs.Since = "60m"
	ctr := &commands.Container{ID: "abc"}
	ctr.Details.ContainerJSONBase = &container.ContainerJSONBase{
		State: &container.State{Running: false},
	}

	assert.Equal(t, "", gui.containerLogSince(ctr))

	ctr.Details.State.Running = true
	assert.Equal(t, "60m", gui.containerLogSince(ctr))
}

func TestContainerLogSinceIsEmptyForFullLogs(t *testing.T) {
	gui := logsGuiWithTail("")
	gui.Config.UserConfig.Logs.Since = "60m"
	container := &commands.Container{ID: "abc"}
	gui.fullLogs = map[string]bool{container.ID: true}

	assert.Equal(t, "", gui.containerLogSince(container))
}

func TestContainerLogSinceIgnoresAgeForExitedSummary(t *testing.T) {
	gui := logsGuiWithTail("20")
	gui.Config.UserConfig.Logs.Since = "60m"
	ctr := &commands.Container{ID: "abc"}
	ctr.Container.State = "exited"

	assert.Equal(t, "", gui.containerLogSince(ctr))
}
