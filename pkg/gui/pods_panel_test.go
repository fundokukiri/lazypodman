package gui

import (
	"sort"
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestSortPods(t *testing.T) {
	pods := []*commands.Pod{
		{Name: "b", Status: "Running"},
		{Name: "a", Status: "Exited"},
		{Name: "c", Status: "Running"},
		{Name: "d", Status: "Something new"},
		{Name: "e", Status: "Paused"},
	}

	sort.Slice(pods, func(i, j int) bool { return sortPods(pods[i], pods[j]) })

	names := make([]string, 0, len(pods))
	for _, pod := range pods {
		names = append(names, pod.Name)
	}

	assert.Equal(t, []string{"b", "c", "e", "a", "d"}, names)
}

func TestPodLogTail(t *testing.T) {
	newGui := func(tail string) *Gui {
		return &Gui{Config: &config.AppConfig{UserConfig: &config.UserConfig{Logs: config.LogsConfig{Tail: tail}}}}
	}

	assert.Equal(t, 50, newGui("50").podLogTail())
	// "all" and junk fall back to a bounded tail: a pod with busy logging must not
	// flood the process.
	assert.Equal(t, 300, newGui("all").podLogTail())
	assert.Equal(t, 300, newGui("").podLogTail())
	assert.Equal(t, 300, newGui("-5").podLogTail())
}
