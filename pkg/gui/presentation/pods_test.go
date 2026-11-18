package presentation

import (
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/stretchr/testify/assert"
)

func TestGetPodDisplayStrings(t *testing.T) {
	pod := &commands.Pod{Name: "web", Status: "Running", NumberOfContainers: 3, Ports: "80->80/tcp"}

	cells := GetPodDisplayStrings(pod)

	assert.Len(t, cells, 4)
	assert.Contains(t, cells[1], "web")
	assert.Contains(t, cells[2], "3")
	assert.Contains(t, cells[3], "80->80/tcp")
}

func TestPodDisplayStatus(t *testing.T) {
	for _, tc := range []struct {
		status string
		icon   string
	}{
		{"Running", "R"},
		{"Exited", "X"},
		{"Stopped", "X"},
		{"Paused", "P"},
		{"Created", "C"},
		{"Degraded", "D"},
		{"Mystery", "?"},
	} {
		assert.Contains(t, podDisplayStatus(&commands.Pod{Status: tc.status}), tc.icon, tc.status)
	}
}

func TestColoredPodLogLine(t *testing.T) {
	assert.Equal(t, "plain message", ColoredPodLogLine("", "plain message"))
	assert.Contains(t, ColoredPodLogLine("web", "hello"), "hello")
}
