package gui

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func diagnosedContainer(state *container.State, restartCount int) *commands.Container {
	ctr := &commands.Container{Name: "api", ID: "abc"}
	ctr.Details.ContainerJSONBase = &container.ContainerJSONBase{
		Path:         "sh",
		Args:         []string{"-c", "exit 1"},
		State:        state,
		RestartCount: restartCount,
		HostConfig: &container.HostConfig{
			RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyAlways},
			Resources:     container.Resources{Memory: 512 * 1024 * 1024},
		},
	}
	ctr.Details.Config = &container.Config{Image: "docker.io/library/alpine"}

	return ctr
}

func TestDiagnosisExplainsOOMKill(t *testing.T) {
	ctr := diagnosedContainer(&container.State{
		Status:     "exited",
		ExitCode:   137,
		OOMKilled:  true,
		StartedAt:  "2026-01-01T10:00:00Z",
		FinishedAt: "2026-01-01T10:00:10Z",
	}, 2)
	ctr.Details.State.Status = "exited"

	got := utils.Decolorise(containerDiagnosis(translations(), ctr))

	assert.Contains(t, got, "Likely cause")
	assert.Contains(t, got, translations().DiagnoseOOMCause)
	assert.Contains(t, got, "Restart count: 2")
	assert.Contains(t, got, "Memory limit: 536.87MB")
	assert.Contains(t, got, "Duration: 10s")
}

func TestDiagnosisExplainsMissingCommand(t *testing.T) {
	ctr := diagnosedContainer(&container.State{Status: "exited", ExitCode: 127}, 0)

	got := utils.Decolorise(containerDiagnosis(translations(), ctr))

	assert.Contains(t, got, "Exit code: 127")
	assert.Contains(t, got, translations().DiagnoseExit127Cause)
}

func TestDiagnosisExplainsRunningContainer(t *testing.T) {
	ctr := diagnosedContainer(&container.State{Status: "running", Running: true}, 0)

	got := utils.Decolorise(containerDiagnosis(translations(), ctr))

	assert.Contains(t, got, translations().DiagnoseRunningCause)
	assert.NotContains(t, got, "Exit code")
}

func TestDiagnosisWaitsForDetails(t *testing.T) {
	ctr := &commands.Container{Name: "api"}

	assert.Equal(t, translations().DiagnoseNoDetails, containerDiagnosis(translations(), ctr))
}

func TestDiagnosisFormatsExitCodeAndHealth(t *testing.T) {
	ctr := diagnosedContainer(&container.State{
		Status:   "exited",
		ExitCode: 1,
		Health: &container.Health{
			Status: "unhealthy",
			Log: []*container.HealthcheckResult{
				{ExitCode: 1, Output: "connection refused\n"},
			},
		},
	}, 0)

	got := utils.Decolorise(containerDiagnosis(translations(), ctr))

	assert.Contains(t, got, fmt.Sprintf("%s: unhealthy", translations().HealthTitle))
	assert.Contains(t, got, "connection refused")
	assert.Contains(t, got, translations().DiagnoseExitOneCause)
}
