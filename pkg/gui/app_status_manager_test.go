package gui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStatusManagerWaitingWinsOverFlash(t *testing.T) {
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.Local)
	manager := &statusManager{now: func() time.Time { return now }}

	manager.setFlash("healthcheck: healthy", 10*time.Second)
	assert.Equal(t, "healthcheck: healthy", manager.getStatusString())

	manager.addWaitingStatus("Running healthcheck")
	assert.Contains(t, manager.getStatusString(), "Running healthcheck")

	manager.removeStatus("Running healthcheck")
	assert.Equal(t, "healthcheck: healthy", manager.getStatusString(), "the flash comes back when the wait ends")
}

func TestStatusManagerFlashExpires(t *testing.T) {
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.Local)
	manager := &statusManager{now: func() time.Time { return now }}

	manager.setFlash("healthcheck: unhealthy (exit 1)", 10*time.Second)
	assert.NotEmpty(t, manager.getStatusString())

	now = now.Add(11 * time.Second)
	assert.Empty(t, manager.getStatusString())
}

func TestStatusManagerEmptyByDefault(t *testing.T) {
	manager := &statusManager{}

	assert.Empty(t, manager.getStatusString())
}
