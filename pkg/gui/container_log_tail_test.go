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

	logs := config.GetDefaultConfig().Logs
	logs.Tail = tail

	return &Gui{
		Config: &config.AppConfig{UserConfig: &config.UserConfig{Logs: logs}},
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
	assert.Equal(t, "10000", gui.containerLogTail(container), "full history is capped, not unbounded")

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

	assert.Equal(t, "500", gui.transcriptLogTail(), "the default replay budget, not the 20-line tail")

	gui.Config.UserConfig.Logs.Tail = "9000"
	assert.Equal(t, "500", gui.transcriptLogTail(), "the tail does not inflate the replay")

	gui.Config.UserConfig.Logs.Tail = ""
	assert.Equal(t, "500", gui.transcriptLogTail(), "an unbounded tail does not replay everything")
}

func TestTranscriptLogTailIgnoresFullHistory(t *testing.T) {
	gui := logsGuiWithTail("20")
	gui.fullLogs = map[string]bool{"abc": true}

	assert.Equal(t, "500", gui.transcriptLogTail(), "ctrl+l must not make the emulator replay the whole log")
}

func TestTranscriptLogTailFollowsTheReplayBudget(t *testing.T) {
	gui := logsGuiWithTail("20")
	gui.Config.UserConfig.Logs.TranscriptTail = 0

	assert.Equal(t, "500", gui.transcriptLogTail(), "0 keeps the default instead of replaying everything")

	gui.Config.UserConfig.Logs.TranscriptTail = 1200
	assert.Equal(t, "1200", gui.transcriptLogTail())
}

func TestFullHistoryTailIsBounded(t *testing.T) {
	gui := logsGuiWithTail("20")
	container := &commands.Container{ID: "abc"}
	gui.fullLogs = map[string]bool{container.ID: true}

	assert.Equal(t, "10000", gui.containerLogTail(container))

	gui.Config.UserConfig.Logs.FullTail = 0
	assert.Equal(t, "", gui.containerLogTail(container), "0 keeps the unbounded behaviour")
}

func TestDefaultFullTailIsBounded(t *testing.T) {
	assert.Equal(t, 10000, config.GetDefaultConfig().Logs.FullTail)
}

// Fetching more than the view keeps means paying for lines only to trim them
// away, which is how ctrl+l used to fill memory with rendered cells.
func TestDefaultFullTailMatchesWhatTheViewKeeps(t *testing.T) {
	assert.Equal(t, logViewKeepLines, config.GetDefaultConfig().Logs.FullTail)
}

func TestDefaultTranscriptReplayIsBounded(t *testing.T) {
	assert.Equal(t, 500, config.GetDefaultConfig().Logs.TranscriptTail)
}

func TestLogsIsTheFirstTab(t *testing.T) {
	gui := logsGuiWithTail("20")

	containers := gui.getContainersPanel()
	assert.Equal(t, "logs", containers.ContextState.GetMainTabs()[0].Key)

	services := gui.getServicesPanel()
	assert.Equal(t, "logs", services.ContextState.GetMainTabs()[0].Key)

	pods := gui.getPodsPanel()
	assert.Equal(t, "logs", pods.ContextState.GetMainTabs()[0].Key)
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

	assert.Equal(t, "", gui.containerLogSince(container), "the tail budget bounds it instead")
}

func TestContainerLogSinceIgnoresAgeForExitedSummary(t *testing.T) {
	gui := logsGuiWithTail("20")
	gui.Config.UserConfig.Logs.Since = "60m"
	ctr := &commands.Container{ID: "abc"}
	ctr.Container.State = "exited"

	assert.Equal(t, "", gui.containerLogSince(ctr))
}
