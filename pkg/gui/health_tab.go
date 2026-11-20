package gui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/gui/presentation"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/jesseduffield/gocui"
)

// healthcheckOutcome is the fate of one manual probe run: either a result line
// for the status bar, or an error for the error panel. An unhealthy result is a
// result, never an error.
type healthcheckOutcome struct {
	message string
	err     error
}

// containerHealth returns the container's health struct, if the inspect data has
// been loaded and declares one.
func containerHealth(ctr *commands.Container) *container.Health {
	details := ctr.DetailsSnapshot()
	if details.ContainerJSONBase == nil || details.State == nil {
		return nil
	}

	return details.State.Health
}

// hasHealthcheck tells whether a healthcheck is configured. "none" is what the
// API reports for containers started without --health-cmd.
func hasHealthcheck(health *container.Health) bool {
	return health != nil && health.Status != "" && health.Status != container.NoHealthcheck
}

// containerIsRunning trusts inspect details when they are loaded: they carry the
// health data and are what the probe acts on. Before details load, the list state
// is the only answer.
func containerIsRunning(ctr *commands.Container) bool {
	if ctr.DetailsLoaded() && ctr.DetailsSnapshot().State != nil {
		return ctr.DetailsSnapshot().State.Running
	}

	return strings.EqualFold(ctr.Container.State, "running")
}

// classifyHealthcheck maps one run's exit code to a result.
//
// Only exit 0 (healthy) is verified. 1 and 2 come from Podman's documentation and
// are UNVERIFIED. Any other code means we cannot interpret the run, so it becomes
// an error and podman's output is shown verbatim rather than guessed at.
func classifyHealthcheck(tr *i18n.TranslationSet, output string, exitCode int, err error) healthcheckOutcome {
	if err != nil {
		return healthcheckOutcome{err: healthcheckError(output, err.Error())}
	}

	switch exitCode {
	case 0:
		return healthcheckOutcome{message: tr.HealthcheckHealthyResult}
	case 1, 2:
		return healthcheckOutcome{message: fmt.Sprintf(tr.HealthcheckUnhealthyResult, exitCode)}
	default:
		return healthcheckOutcome{err: healthcheckError(output, fmt.Sprintf("unexpected exit code %d", exitCode))}
	}
}

func healthcheckError(output, fallback string) error {
	if trimmed := strings.TrimSpace(output); trimmed != "" {
		return fmt.Errorf("%s", trimmed)
	}

	return fmt.Errorf("%s", fallback)
}

// healthcheckBlockedReason returns why a run must not start, or "" when it may.
// All of these checks happen before any process is spawned.
func (gui *Gui) healthcheckBlockedReason(ctr *commands.Container) string {
	if !containerIsRunning(ctr) {
		return gui.Tr.ContainerNotRunning
	}

	if !hasHealthcheck(containerHealth(ctr)) {
		return gui.Tr.NoHealthcheck
	}

	if gui.PodmanCommand != nil {
		if cli := gui.PodmanCommand.Capabilities().CLI; cli.State == commands.CapabilityUnavailable {
			return gui.Tr.CliOff + " " + cli.Reason
		}
	}

	return ""
}

// runHealthcheck probes the capability (once, lazily) and then runs the probe.
// It performs no UI work, so it is testable and safe off the UI goroutine.
func (gui *Gui) runHealthcheck(ctr *commands.Container) healthcheckOutcome {
	if gui.PodmanCommand == nil || gui.PodmanCommand.Pods() == nil {
		return healthcheckOutcome{message: gui.Tr.CliOff + " podman CLI not found"}
	}

	capability := gui.PodmanCommand.HealthcheckCapability(context.Background())
	if capability.State != commands.CapabilityAvailable {
		return healthcheckOutcome{message: gui.Tr.HealthRunOff + " " + capability.Reason}
	}

	output, exitCode, err := gui.PodmanCommand.Pods().HealthcheckRun(context.Background(), ctr.ID)

	return classifyHealthcheck(gui.Tr, output, exitCode, err)
}

func (gui *Gui) handleContainerHealthcheckRun(g *gocui.Gui, v *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	if reason := gui.healthcheckBlockedReason(ctr); reason != "" {
		gui.flashStatus(reason)

		return nil
	}

	return gui.WithWaitingStatus(gui.Tr.RunningHealthcheckStatus, func() error {
		outcome := gui.runHealthcheck(ctr)
		if outcome.err != nil {
			return outcome.err
		}

		gui.refreshHealthAfterRun(ctr)
		gui.flashStatus(outcome.message)

		return nil
	})
}

// refreshHealthAfterRun re-fetches the details so the Health tab and the list
// colour show the probe that was just recorded, without waiting for the ticker.
// The ticker itself also re-fetches the container it displays, because health
// changes only reach us through Podman events, and older Podman versions (and
// a dropped event stream) do not send health_status events.
func (gui *Gui) refreshHealthAfterRun(ctr *commands.Container) {
	if err := gui.PodmanCommand.RefreshContainerDetails([]*commands.Container{ctr}); err != nil {
		gui.Log.Warn(err)
	}

	if err := gui.reRenderMain(); err != nil {
		gui.Log.Warn(err)
	}
}

// renderContainerHealth is the Health tab: status and probe history from inspect
// data, re-read once per tick for the container on screen.
func (gui *Gui) renderContainerHealth(ctr *commands.Container) tasks.TaskFunc {
	return gui.NewTickerTask(TickerTaskOpts{
		Func: func(ctx context.Context, notifyStopped chan struct{}) {
			// One inspect per second for the container on screen, so the health
			// status and history cannot freeze when no event arrives.
			if err := gui.PodmanCommand.RefreshContainerDetails([]*commands.Container{ctr}); err != nil {
				gui.Log.Warn(err)
			}

			gui.reRenderStringMain(gui.healthTabContents(ctr))
		},
		Duration:   time.Second,
		Before:     func(ctx context.Context) { gui.clearMainView() },
		Wrap:       false, // fixed-width table; wrapping would break the columns
		Autoscroll: false,
	})
}

func (gui *Gui) healthTabContents(ctr *commands.Container) string {
	health := containerHealth(ctr)
	if !hasHealthcheck(health) {
		return gui.Tr.NoHealthcheck
	}

	table := [][]string{{gui.Tr.HealthTimeHeader, gui.Tr.HealthExitHeader, gui.Tr.HealthOutputHeader}}

	for _, row := range presentation.HealthRows(health.Log) {
		cells := presentation.HealthRowCells(row, gui.Tr.NoProbeOutput)
		cells[1] = utils.ColoredString(cells[1], presentation.HealthExitColor(row.ExitCode))
		table = append(table, cells)
	}

	rendered, err := utils.RenderTable(table)
	if err != nil {
		return err.Error()
	}

	header := fmt.Sprintf(
		"%s: %s\n%s: %d\n\n",
		gui.Tr.HealthStatusLabel, health.Status,
		gui.Tr.HealthFailingStreakLabel, health.FailingStreak,
	)

	return header + rendered
}
