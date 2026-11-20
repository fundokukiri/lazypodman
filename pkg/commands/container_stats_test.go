package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// statsWithCPU returns a sample pair where the container used cpuUsage
// nanoseconds of CPU time since the previous sample.
func statsWithCPU(cpuUsage int64) *ContainerStats {
	container := &ContainerStats{}
	container.CPUStats.CPUUsage.TotalUsage = cpuUsage

	return container
}

func TestCalculateContainerCPUPercentage(t *testing.T) {
	// One full core of CPU time in one second of wall time.
	assert.EqualValues(t, 100, statsWithCPU(int64(time.Second)).CalculateContainerCPUPercentage(time.Second))
}

func TestCalculateContainerCPUPercentageCountsCoresNotHostShare(t *testing.T) {
	// 2.5 cores busy; an idle host must not inflate this.
	assert.EqualValues(t, 250, statsWithCPU(2500*int64(time.Millisecond)).CalculateContainerCPUPercentage(time.Second))
}

func TestCalculateContainerCPUPercentageIgnoresNegativeDelta(t *testing.T) {
	// The cgroup counter dropped, so the container was restarted.
	container := statsWithCPU(int64(time.Second))
	container.PrecpuStats.CPUUsage.TotalUsage = 2 * int64(time.Second)

	assert.EqualValues(t, 0, container.CalculateContainerCPUPercentage(time.Second))
}

func TestCalculateContainerCPUPercentageIgnoresTooShortInterval(t *testing.T) {
	// A single cgroup tick over a sub-millisecond gap is noise.
	assert.EqualValues(t, 0, statsWithCPU(int64(time.Millisecond)).CalculateContainerCPUPercentage(time.Microsecond))
}

func TestCalculateContainerCPUPercentageWithoutElapsedTime(t *testing.T) {
	// The first sample of a stream has no predecessor to measure against.
	assert.EqualValues(t, 0, statsWithCPU(int64(time.Second)).CalculateContainerCPUPercentage(0))
}

func TestCalculateContainerMemoryUsage(t *testing.T) {
	container := &ContainerStats{}
	container.MemoryStats.Usage = 256
	container.MemoryStats.Limit = 1024

	assert.EqualValues(t, 25, container.CalculateContainerMemoryUsage())
}

func TestCalculateContainerMemoryUsageWithZeroLimit(t *testing.T) {
	container := &ContainerStats{}
	container.MemoryStats.Usage = 256
	container.MemoryStats.Limit = 0

	assert.EqualValues(t, 0, container.CalculateContainerMemoryUsage())
}
