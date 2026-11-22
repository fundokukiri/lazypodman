package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// statsWithCPU returns a sample pair where the container used cpuUsage
// nanoseconds of CPU time since the previous sample, on a 16-core host.
func statsWithCPU(cpuUsage int64) *ContainerStats {
	container := &ContainerStats{}
	container.CPUStats.CPUUsage.TotalUsage = cpuUsage
	container.CPUStats.OnlineCpus = 16

	return container
}

func TestCalculateContainerCPUPercentage(t *testing.T) {
	// One core busy for one second, on a 16-core host.
	assert.EqualValues(t, 6.25, statsWithCPU(int64(time.Second)).CalculateContainerCPUPercentage(time.Second))
}

func TestCalculateContainerCPUPercentageCountsCoresNotHostShare(t *testing.T) {
	// 2.5 cores busy out of 16, regardless of what the rest of the host does.
	assert.EqualValues(t, 15.625, statsWithCPU(2500*int64(time.Millisecond)).CalculateContainerCPUPercentage(time.Second))
}

func TestCalculateContainerCPUPercentageWithoutOnlineCpus(t *testing.T) {
	// Podman omits online_cpus in some payloads; fall back to percpu_usage.
	container := statsWithCPU(int64(time.Second))
	container.CPUStats.OnlineCpus = 0
	container.CPUStats.CPUUsage.PercpuUsage = make([]int64, 4)

	assert.EqualValues(t, 25, container.CalculateContainerCPUPercentage(time.Second))
}

func TestCalculateContainerCPUPercentageWithoutAnyCoreCount(t *testing.T) {
	// Unknown CPU count still yields a sane number instead of +/-Inf.
	container := statsWithCPU(int64(time.Second))
	container.CPUStats.OnlineCpus = 0

	assert.EqualValues(t, 100, container.CalculateContainerCPUPercentage(time.Second))
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
