package gui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
	"github.com/fundokukiri/lazypodman/pkg/utils"
)

type publishedPort struct {
	HostIP        string
	HostPort      int
	ContainerPort int
	Protocol      string
	Container     string
	Network       string
	Conflict      bool
}

// renderNetworkPorts lists every published port owned by the containers, so a
// host-port collision is visible in one place instead of across container tabs.
func (gui *Gui) renderNetworkPorts(_ *commands.Network) tasks.TaskFunc {
	return gui.NewSimpleRenderStringTask(func() string {
		containers := gui.Panels.Containers.List.GetAllItems()

		return renderPublishedPorts(gui.Tr, collectPublishedPorts(containers), len(containers))
	})
}

func collectPublishedPorts(containers []*commands.Container) []publishedPort {
	ports := make([]publishedPort, 0)

	for _, ctr := range containers {
		network := primaryContainerNetwork(ctr)
		for _, binding := range ctr.Container.Ports {
			ports = append(ports, publishedPort{
				HostIP:        binding.IP,
				HostPort:      int(binding.PublicPort),
				ContainerPort: int(binding.PrivatePort),
				Protocol:      binding.Type,
				Container:     ctr.Name,
				Network:       network,
			})
		}
	}

	owners := map[string]int{}
	for _, port := range ports {
		owners[portKey(port)]++
	}
	for i := range ports {
		ports[i].Conflict = owners[portKey(ports[i])] > 1
	}

	sort.Slice(ports, func(i, j int) bool {
		if ports[i].HostPort != ports[j].HostPort {
			return ports[i].HostPort < ports[j].HostPort
		}
		if ports[i].Protocol != ports[j].Protocol {
			return ports[i].Protocol < ports[j].Protocol
		}

		return ports[i].Container < ports[j].Container
	})

	return ports
}

func portKey(port publishedPort) string {
	return fmt.Sprintf("%s|%d|%s", port.HostIP, port.HostPort, port.Protocol)
}

func primaryContainerNetwork(ctr *commands.Container) string {
	if !ctr.DetailsLoaded() || ctr.Details.NetworkSettings == nil {
		return "-"
	}

	names := make([]string, 0, len(ctr.Details.NetworkSettings.Networks))
	for name := range ctr.Details.NetworkSettings.Networks {
		names = append(names, name)
	}
	if len(names) == 0 {
		return "-"
	}
	sort.Strings(names)

	return names[0]
}

func renderPublishedPorts(tr *i18n.TranslationSet, ports []publishedPort, containerCount int) string {
	if len(ports) == 0 {
		return tr.PortNonePublished
	}

	rows := [][]string{{tr.PortHostHeader, tr.PortContainerHeader, tr.PortProtoHeader, tr.ContainersTitle, tr.NetworksTitle}}
	conflicts := 0
	for _, port := range ports {
		container := port.Container
		if port.Conflict {
			conflicts++
			container = utils.ColoredString(container+" ("+tr.PortConflict+")", color.FgRed)
		}
		rows = append(rows, []string{
			strconv.Itoa(port.HostPort),
			strconv.Itoa(port.ContainerPort),
			port.Protocol,
			container,
			port.Network,
		})
	}

	table, err := utils.RenderTable(rows)
	if err != nil {
		return err.Error()
	}

	header := fmt.Sprintf(tr.PortMapSummary, len(ports), containerCount)
	if conflicts > 0 {
		header += "\n" + fmt.Sprintf(tr.PortConflictSummary, conflicts)
	}

	return header + "\n\n" + table
}

// keep strings imported for the host-IP normalisation used by future filters.
var _ = strings.TrimSpace
