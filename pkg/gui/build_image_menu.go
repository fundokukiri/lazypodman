package gui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	"github.com/jesseduffield/gocui"
)

func (gui *Gui) handleImagesBuildImage(g *gocui.Gui, v *gocui.View) error {
	return gui.showBuildImageMenu(newBuildImageState(), 0)
}

func (gui *Gui) showBuildImageMenu(state *runContainerState, selected int) error {
	items := []*types.MenuItem{}

	for _, field := range buildImageFields {
		field := field
		row := len(items)
		items = append(items, &types.MenuItem{
			LabelColumns: gui.runColumns(gui.runLabel(field.Key), gui.runValueLabel(state, field), gui.runFieldHint(field)),
			OnPress: func() error {
				return gui.activateRunContainerField(state, field, func() error {
					return gui.showBuildImageMenu(state, row)
				})
			},
		})
	}

	advancedRow := len(items)
	items = append(items,
		&types.MenuItem{
			LabelColumns: gui.runColumns(gui.trString(gui.Tr.RunContainerAdvanced, "Advanced"), "", "enter"),
			OnPress: func() error {
				return gui.showBuildImageAdvanced(state, func() error {
					return gui.showBuildImageMenu(state, advancedRow)
				}, 0)
			},
		},
		&types.MenuItem{
			LabelColumns: gui.runColumns(gui.trString(gui.Tr.BuildImage, "Build"), "", "enter"),
			OnPress:      func() error { return gui.previewBuildImage(state) },
		},
	)

	return gui.Menu(CreateMenuOptions{
		Title:        gui.trString(gui.Tr.BuildImageTitle, "Build image"),
		Items:        items,
		SelectedLine: selected,
	})
}

func (gui *Gui) showBuildImageAdvanced(state *runContainerState, back func() error, selected int) error {
	items := []*types.MenuItem{
		{
			LabelColumns: gui.runColumns(gui.trString(gui.Tr.RunContainerBack, "back"), "", ""),
			OnPress:      back,
		},
	}

	for _, group := range buildImageGroups {
		group := group
		row := len(items)
		items = append(items, &types.MenuItem{
			LabelColumns: gui.runColumns(gui.runLabel(group.Key), "", "enter"),
			OnPress: func() error {
				return gui.showBuildImageGroup(state, group, func() error {
					return gui.showBuildImageAdvanced(state, back, row)
				}, 0)
			},
		})
	}

	extraRow := len(items)
	items = append(items, &types.MenuItem{
		LabelColumns: gui.runColumns(gui.trString(gui.Tr.RunContainerExtraArgs, "Extra args"), state.extraArgs, "enter"),
		OnPress: func() error {
			return gui.editRunContainerValue("extra_args", state.extraArgs, func(value string) {
				state.extraArgs = value
			}, func() error { return gui.showBuildImageAdvanced(state, back, extraRow) })
		},
	})

	return gui.Menu(CreateMenuOptions{
		Title:        gui.trString(gui.Tr.BuildImageTitle, "Build image"),
		Items:        items,
		OnCancel:     back,
		SelectedLine: selected,
	})
}

func (gui *Gui) showBuildImageGroup(state *runContainerState, group runFieldGroup, back func() error, selected int) error {
	items := []*types.MenuItem{
		{
			LabelColumns: gui.runColumns(gui.trString(gui.Tr.RunContainerBack, "back"), "", ""),
			OnPress:      back,
		},
	}

	for _, field := range group.Fields {
		field := field
		row := len(items)
		items = append(items, &types.MenuItem{
			LabelColumns: gui.runColumns(gui.runLabel(field.Key), gui.runValueLabel(state, field), gui.runFieldHint(field)),
			OnPress: func() error {
				return gui.activateRunContainerField(state, field, func() error {
					return gui.showBuildImageGroup(state, group, back, row)
				})
			},
		})
	}

	return gui.Menu(CreateMenuOptions{
		Title:        gui.runLabel(group.Key),
		Items:        items,
		OnCancel:     back,
		SelectedLine: selected,
	})
}

func (gui *Gui) previewBuildImage(state *runContainerState) error {
	argv := state.buildArgv(state.extraArgs)
	command := formatRunCommand(gui.Config.PodmanBinary(), argv)

	return gui.createConfirmationPanel(
		gui.trString(gui.Tr.BuildImageConfirmTitle, "Confirm build"),
		fmt.Sprintf("%s\n\n%s", command, gui.trString(gui.Tr.BuildImageConfirm, "Build this image?")),
		func(g *gocui.Gui, v *gocui.View) error {
			return gui.executeBuildImage(state)
		},
		nil,
	)
}

func (gui *Gui) executeBuildImage(state *runContainerState) error {
	argv := state.buildArgv(state.extraArgs)
	command := exec.Command(gui.Config.PodmanBinary(), argv...)

	return gui.WithWaitingStatus(gui.trString(gui.Tr.BuildImageRunningStatus, "Building image"), func() error {
		output, err := command.CombinedOutput()
		if err != nil {
			message := strings.TrimSpace(string(output))
			if message == "" {
				message = err.Error()
			}
			return fmt.Errorf("%s", message)
		}

		return gui.reloadImages()
	})
}
