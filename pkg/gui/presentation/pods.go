package presentation

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/utils"
)

// GetPodDisplayStrings returns the cells of a pods panel row: status, name,
// container count and the ports the pod's infra container publishes.
func GetPodDisplayStrings(pod *commands.Pod) []string {
	return []string{
		podDisplayStatus(pod),
		pod.Name,
		fmt.Sprintf("%d", pod.NumberOfContainers),
		utils.ColoredString(pod.Ports, color.FgYellow),
	}
}

func podDisplayStatus(pod *commands.Pod) string {
	switch strings.ToLower(pod.Status) {
	case "running":
		return utils.ColoredString("R", color.FgGreen)
	case "exited", "stopped":
		return utils.ColoredString("X", color.FgRed)
	case "paused":
		return utils.ColoredString("P", color.FgYellow)
	case "created":
		return utils.ColoredString("C", color.FgCyan)
	case "degraded":
		return utils.ColoredString("D", color.FgRed)
	default:
		// An unknown status is not worth an error: show it as unknown and let the
		// user open the pod.
		return utils.ColoredString("?", color.FgYellow)
	}
}

var podLogNameColors = []color.Attribute{
	color.FgBlue,
	color.FgMagenta,
	color.FgCyan,
	color.FgYellow,
	color.FgGreen,
}

// ColoredPodLogLine renders one merged log line. `podman logs --names` prefixes
// every line with its container name; colouring the name (deterministically, by
// hashing it) keeps the streams apart without reordering them.
func ColoredPodLogLine(name, message string) string {
	if name == "" {
		return message
	}

	return utils.ColoredString(name, PodContainerNameColor(name)) + " " + message
}

// RenderPodStats renders the pod stats tab. The numbers are a sum over the pod's
// containers, which is what the socket backend can report; the tab title says so.
func RenderPodStats(pod *commands.Pod, row commands.PodStatsRow) string {
	memPercent := 0
	if row.MemLimit > 0 {
		memPercent = int(row.MemUsage * 100 / row.MemLimit)
	}

	rows := [][]string{
		{utils.ColoredString("Pod", color.FgGreen), pod.Name},
		{utils.ColoredString("Status", color.FgGreen), pod.Status},
		{utils.ColoredString("Containers", color.FgGreen), fmt.Sprintf("%d", pod.NumberOfContainers)},
		{utils.ColoredString("CPU", color.FgGreen), fmt.Sprintf("%.2f%%", row.CPU)},
		{utils.ColoredString("Mem usage", color.FgGreen), utils.FormatDecimalBytes(int(row.MemUsage))},
		{utils.ColoredString("Mem limit", color.FgGreen), utils.FormatDecimalBytes(int(row.MemLimit))},
		{utils.ColoredString("Mem %", color.FgGreen), fmt.Sprintf("%d%%", memPercent)},
	}

	rendered, err := utils.RenderTable(rows)
	if err != nil {
		return fmt.Sprintf("CPU: %.2f%%\nMem usage: %s", row.CPU, utils.FormatDecimalBytes(int(row.MemUsage)))
	}

	return rendered
}

// PodContainerNameColor returns a stable colour for a container name, shared by
// the pod containers tab and its log lines so a name is recognisable at a glance.
func PodContainerNameColor(name string) color.Attribute {
	hash := 0
	for _, r := range name {
		hash += int(r)
	}

	return podLogNameColors[hash%len(podLogNameColors)]
}
