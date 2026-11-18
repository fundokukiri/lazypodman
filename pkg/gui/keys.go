package gui

import (
	"fmt"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/jesseduffield/gocui"
)

// sidePanelKey binds one number to one side panel. The [N] title prefixes and the
// numeric focus keybindings both come from this list, so adding or hiding a panel
// cannot leave the title and the key disagreeing.
type sidePanelKey struct {
	name        string
	key         rune
	description string
	view        *gocui.View
}

// PodsEnabled reports whether the Podman-native panel is available. It is false
// on hosts where the podman CLI is missing, too old or talking to another store,
// and the window stays exactly as it was without pods.
func (gui *Gui) PodsEnabled() bool {
	if gui.PodmanCommand == nil || gui.Views.Pods == nil {
		return false
	}

	if pods := gui.PodmanCommand.Capabilities().Pods; pods.State != commands.CapabilityUnknown {
		return pods.State == commands.CapabilityAvailable
	}

	client := gui.PodmanCommand.Pods()

	return client != nil && client.Enabled()
}

// sidePanelKeys lists the side panels in display order. The pods panel sits
// between services and containers because a pod contains containers. When pods
// are off the list is one shorter and the numbering stays contiguous (6 keys
// instead of 7), which is why no key is hardcoded anywhere else.
func (gui *Gui) sidePanelKeys() []sidePanelKey {
	keys := []sidePanelKey{
		{name: "project", key: '1', description: gui.Tr.FocusProjects, view: gui.Views.Project},
		{name: "services", key: '2', description: gui.Tr.FocusServices, view: gui.Views.Services},
	}

	if gui.PodsEnabled() {
		keys = append(keys, sidePanelKey{name: "pods", key: '3', description: gui.Tr.FocusPods, view: gui.Views.Pods})
	}

	remaining := []sidePanelKey{
		{name: "containers", description: gui.Tr.FocusContainers, view: gui.Views.Containers},
		{name: "images", description: gui.Tr.FocusImages, view: gui.Views.Images},
		{name: "volumes", description: gui.Tr.FocusVolumes, view: gui.Views.Volumes},
		{name: "networks", description: gui.Tr.FocusNetworks, view: gui.Views.Networks},
	}

	for i := range remaining {
		remaining[i].key = rune('1' + len(keys))
		keys = append(keys, remaining[i])
	}

	return keys
}

// sidePanelTitlePrefix returns the "[N]" prefix for a panel, empty when the panel
// has no focus key (pods while disabled).
func (gui *Gui) sidePanelTitlePrefix(name string) string {
	for _, panel := range gui.sidePanelKeys() {
		if panel.name == name {
			return fmt.Sprintf("[%c]", panel.key)
		}
	}

	return ""
}
