package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateContainerCPUPercentage(t *testing.T) {
	container := &ContainerStats{}
	container.CPUStats.CPUUsage.TotalUsage = 10
	container.CPUStats.SystemCPUUsage = 10
	container.PrecpuStats.CPUUsage.TotalUsage = 5
	container.PrecpuStats.SystemCPUUsage = 2

	assert.EqualValues(t, 62.5, container.CalculateContainerCPUPercentage())
}

func TestCalculateContainerCPUPercentageWithZeroSystemDelta(t *testing.T) {
	// Podman may not report system_cpu_usage in its first stats frame.
	container := &ContainerStats{}
	container.CPUStats.CPUUsage.TotalUsage = 10
	container.CPUStats.SystemCPUUsage = 10
	container.PrecpuStats.SystemCPUUsage = 10

	assert.EqualValues(t, 0, container.CalculateContainerCPUPercentage())
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
