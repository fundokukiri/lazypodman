package presentation

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/stretchr/testify/assert"
)

func TestContainerUptimeFormatsRunningTime(t *testing.T) {
	started := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	ctr := &commands.Container{Details: container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			State: &container.State{Running: true, StartedAt: started.Format(time.RFC3339Nano)},
		},
	}}

	assert.Equal(t, "2h 15m", containerUptime(ctr, started.Add(2*time.Hour+15*time.Minute)))
	assert.Equal(t, "3d 4h 5m", containerUptime(ctr, started.Add(3*24*time.Hour+4*time.Hour+5*time.Minute)))
	assert.Equal(t, "42s", containerUptime(ctr, started.Add(42*time.Second)))
}

func TestContainerUptimeWithoutDetailsIsUnavailable(t *testing.T) {
	assert.Equal(t, "unavailable", containerUptime(&commands.Container{}, time.Now()))
}

func TestContainerUptimeForStoppedContainer(t *testing.T) {
	ctr := &commands.Container{Details: container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			State: &container.State{Running: false},
		},
	}}

	assert.Equal(t, "not running", containerUptime(ctr, time.Now()))
}
