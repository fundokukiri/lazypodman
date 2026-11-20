package gui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/fatih/color"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
	"github.com/fundokukiri/lazypodman/pkg/utils"
)

// renderContainerDiagnosis is the Diagnose tab. It reads only the inspect data
// already in memory, so opening it never starts a podman process.
func (gui *Gui) renderContainerDiagnosis(ctr *commands.Container) tasks.TaskFunc {
	return gui.NewTickerTask(TickerTaskOpts{
		Duration: time.Second,
		Before:   func(ctx context.Context) { gui.clearMainView() },
		Func: func(ctx context.Context, notifyStopped chan struct{}) {
			gui.reRenderStringMain(gui.diagnosisContents(ctr))
		},
		Wrap:       false,
		Autoscroll: false,
	})
}

func (gui *Gui) diagnosisContents(ctr *commands.Container) string {
	return containerDiagnosis(gui.Tr, ctr)
}

// containerDiagnosis explains, in plain language, why a container is in its
// current state. It is a pure function so the wording and exit-code mapping can
// be tested without a terminal.
func containerDiagnosis(tr *i18n.TranslationSet, ctr *commands.Container) string {
	if !ctr.DetailsLoaded() || ctr.DetailsSnapshot().State == nil {
		return tr.DiagnoseNoDetails
	}

	state := ctr.DetailsSnapshot().State
	policy := ctr.DetailsSnapshot().HostConfig.RestartPolicy

	var b strings.Builder

	field := func(label, value string) {
		b.WriteString(utils.ColoredString(label+": ", color.FgGreen))
		b.WriteString(value)
		b.WriteString("\n")
	}

	field(tr.DiagnoseStatus, state.Status)
	if !state.Running {
		field(tr.DiagnoseExitCode, fmt.Sprintf("%d", state.ExitCode))
	}
	field(tr.DiagnoseOOMKilled, yesNo(state.OOMKilled))
	field(tr.DiagnoseRestarts, fmt.Sprintf("%d", ctr.DetailsSnapshot().RestartCount))

	policyName := string(policy.Name)
	if policyName == "" {
		policyName = "no"
	}
	field(tr.DiagnoseRestartPolicy, policyName)

	if started := parseInspectTime(state.StartedAt); !started.IsZero() {
		field(tr.DiagnoseStarted, started.Local().Format("2006-01-02 15:04:05"))
		if finished := parseInspectTime(state.FinishedAt); !finished.IsZero() && finished.After(started) {
			field(tr.DiagnoseFinished, finished.Local().Format("2006-01-02 15:04:05"))
			field(tr.DiagnoseDuration, formatDiagnosisDuration(finished.Sub(started)))
		} else if state.Running {
			field(tr.DiagnoseDuration, formatDiagnosisDuration(time.Since(started)))
		}
	}

	if state.Error != "" {
		field(tr.ErrorTitle, state.Error)
	}
	if ctr.DetailsSnapshot().HostConfig.Memory > 0 {
		field(tr.DiagnoseMemoryLimit, utils.FormatDecimalBytes(int(ctr.DetailsSnapshot().HostConfig.Memory)))
	}
	if health := state.Health; health != nil && health.Status != "" && health.Status != container.NoHealthcheck {
		value := health.Status
		if len(health.Log) > 0 {
			last := health.Log[len(health.Log)-1]
			if output := strings.TrimSpace(last.Output); output != "" {
				value += " — " + utils.Decolorise(output)
			}
		}
		field(tr.HealthTitle, value)
	}

	command := strings.TrimSpace(strings.Join(append([]string{ctr.DetailsSnapshot().Path}, ctr.DetailsSnapshot().Args...), " "))
	if command != "" {
		field(tr.DiagnoseCommand, command)
	}
	if ctr.DetailsSnapshot().Config != nil && ctr.DetailsSnapshot().Config.Image != "" {
		field(tr.DiagnoseImage, ctr.DetailsSnapshot().Config.Image)
	}

	b.WriteString("\n")
	b.WriteString(utils.ColoredString(tr.DiagnoseLikelyCause, color.FgYellow))
	b.WriteString("\n")
	b.WriteString(diagnosisCause(tr, ctr))

	return b.String()
}

func diagnosisCause(tr *i18n.TranslationSet, ctr *commands.Container) string {
	state := ctr.DetailsSnapshot().State
	if state == nil {
		return tr.DiagnoseNoDetails
	}

	if state.Running {
		return tr.DiagnoseRunningCause
	}

	switch state.Status {
	case "created":
		return tr.DiagnoseCreatedCause
	case "paused":
		return tr.DiagnosePausedCause
	}

	if state.OOMKilled {
		return tr.DiagnoseOOMCause
	}

	switch state.ExitCode {
	case 0:
		return tr.DiagnoseExitZeroCause
	case 1:
		return tr.DiagnoseExitOneCause
	case 125:
		return tr.DiagnoseExit125Cause
	case 126:
		return tr.DiagnoseExit126Cause
	case 127:
		return tr.DiagnoseExit127Cause
	case 137:
		return tr.DiagnoseExit137Cause
	case 139:
		return tr.DiagnoseExit139Cause
	case 143:
		return tr.DiagnoseExit143Cause
	default:
		return fmt.Sprintf(tr.DiagnoseExitOtherCause, state.ExitCode)
	}
}

func parseInspectTime(raw string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}

	return parsed
}

func formatDiagnosisDuration(d time.Duration) string {
	if d < time.Second {
		return "< 1s"
	}

	d = d.Round(time.Second)
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, " ")
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}
