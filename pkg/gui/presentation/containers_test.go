package presentation

import (
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func TestCPUColorWarnsOnOneBusyCore(t *testing.T) {
	// 6.25% of a 16-core host is exactly one core at full load.
	assert.Equal(t, color.FgYellow, cpuColor(6.25, 16))
	assert.Equal(t, color.FgYellow, cpuColor(12.5, 16))
	assert.Equal(t, color.FgWhite, cpuColor(6.24, 16))
}

func TestCPUColorGoesRedWhenMostOfTheHostIsBusy(t *testing.T) {
	assert.Equal(t, color.FgRed, cpuColor(95, 16))
}

func TestCPUColorOnSingleCoreHost(t *testing.T) {
	// A full core is the whole host, so it reads as "most of the host is busy".
	assert.Equal(t, color.FgRed, cpuColor(100, 1))
}
