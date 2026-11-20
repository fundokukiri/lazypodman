package gui

import (
	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	"github.com/jesseduffield/gocui"
)

// handleCommandPalette lists the actions that are easy to forget or buried behind
// another key. It reuses the exact handlers, so the palette cannot drift from the
// keybindings: it only makes them discoverable.
func (gui *Gui) handleCommandPalette(g *gocui.Gui, v *gocui.View) error {
	return gui.Menu(CreateMenuOptions{Title: gui.Tr.PaletteTitle, Items: gui.paletteItems()})
}

func (gui *Gui) paletteItems() []*types.MenuItem {
	items := make([]*types.MenuItem, 0, len(gui.sidePanelKeys())+6)

	for _, panel := range gui.sidePanelKeys() {
		panel := panel
		items = append(items, &types.MenuItem{
			LabelColumns: []string{panel.description},
			OnPress:      func() error { return gui.handleGoTo(panel.view)(nil, nil) },
		})
	}

	items = append(items,
		&types.MenuItem{
			LabelColumns: []string{gui.Tr.ContainerShellTitle},
			OnPress: func() error {
				container, err := gui.Panels.Containers.GetSelectedItem()
				if err != nil {
					return nil
				}

				return gui.startContainerTerminal(container)
			},
		},
		&types.MenuItem{
			LabelColumns: []string{gui.Tr.CloseShell},
			OnPress:      func() error { return gui.closeContainerTerminal() },
		},
		&types.MenuItem{
			LabelColumns: []string{gui.Tr.ConnectionsTitle},
			OnPress:      func() error { return gui.handleConnectionsMenu(nil, nil) },
		},
		&types.MenuItem{
			LabelColumns: []string{gui.Tr.LogSearchTitle},
			OnPress:      func() error { return gui.handleLogSearch(nil, nil) },
		},
		&types.MenuItem{
			LabelColumns: []string{gui.Tr.KubeMenuTitle},
			OnPress:      func() error { return gui.handlePodsKubeMenu(nil, nil) },
		},
		&types.MenuItem{
			LabelColumns: []string{gui.Tr.RunHealthcheck},
			OnPress:      func() error { return gui.handleContainerHealthcheckRun(nil, nil) },
		},
		&types.MenuItem{
			LabelColumns: []string{gui.Tr.AutoUpdateContainers},
			OnPress:      func() error { return gui.handleAutoUpdate() },
		},
		&types.MenuItem{
			LabelColumns: []string{gui.Tr.QuadletMenuTitle},
			OnPress:      func() error { return gui.handleQuadletMenu(nil, nil) },
		},
		&types.MenuItem{
			LabelColumns: []string{gui.Tr.Quit},
			OnPress:      func() error { return gui.quit(nil, nil) },
		},
	)

	return items
}
