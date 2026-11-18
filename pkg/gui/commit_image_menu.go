package gui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	"github.com/jesseduffield/gocui"
)

func (gui *Gui) handleContainersCommitImage(g *gocui.Gui, v *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	return gui.showCommitImageMenu(newCommitImageState(ctr), ctr, 0)
}

func commitContainerRef(ctr *commands.Container) string {
	if ctr.Name != "" {
		return ctr.Name
	}
	return ctr.ID
}

func (gui *Gui) showCommitImageMenu(state *runContainerState, ctr *commands.Container, selected int) error {
	items := []*types.MenuItem{}

	for _, field := range commitImageFields {
		field := field
		row := len(items)
		items = append(items, &types.MenuItem{
			LabelColumns: gui.runColumns(gui.runLabel(field.Key), gui.runValueLabel(state, field), gui.runFieldHint(field)),
			OnPress: func() error {
				return gui.activateRunContainerField(state, field, func() error {
					return gui.showCommitImageMenu(state, ctr, row)
				})
			},
		})
	}

	advancedRow := len(items)
	items = append(items,
		&types.MenuItem{
			LabelColumns: gui.runColumns(gui.trString(gui.Tr.RunContainerAdvanced, "Advanced"), "", "enter"),
			OnPress: func() error {
				return gui.showCommitImageAdvanced(state, ctr, func() error {
					return gui.showCommitImageMenu(state, ctr, advancedRow)
				}, 0)
			},
		},
		&types.MenuItem{
			LabelColumns: gui.runColumns(gui.trString(gui.Tr.CommitImage, "commit"), "", "enter"),
			OnPress:      func() error { return gui.previewCommitImage(state, ctr) },
		},
	)

	return gui.Menu(CreateMenuOptions{
		Title:        gui.trString(gui.Tr.CommitImageTitle, "Commit container"),
		Items:        items,
		SelectedLine: selected,
	})
}

func (gui *Gui) showCommitImageAdvanced(state *runContainerState, ctr *commands.Container, back func() error, selected int) error {
	items := []*types.MenuItem{
		{
			LabelColumns: gui.runColumns(gui.trString(gui.Tr.RunContainerBack, "back"), "", ""),
			OnPress:      back,
		},
	}

	for _, group := range commitImageGroups {
		group := group
		row := len(items)
		items = append(items, &types.MenuItem{
			LabelColumns: gui.runColumns(gui.runLabel(group.Key), "", "enter"),
			OnPress: func() error {
				return gui.showCommitImageGroup(state, ctr, group, func() error {
					return gui.showCommitImageAdvanced(state, ctr, back, row)
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
			}, func() error { return gui.showCommitImageAdvanced(state, ctr, back, extraRow) })
		},
	})

	return gui.Menu(CreateMenuOptions{
		Title:        gui.trString(gui.Tr.CommitImageTitle, "Commit container"),
		Items:        items,
		OnCancel:     back,
		SelectedLine: selected,
	})
}

func (gui *Gui) showCommitImageGroup(state *runContainerState, ctr *commands.Container, group runFieldGroup, back func() error, selected int) error {
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
					return gui.showCommitImageGroup(state, ctr, group, back, row)
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

func (gui *Gui) previewCommitImage(state *runContainerState, ctr *commands.Container) error {
	argv := state.commitArgv(commitContainerRef(ctr), state.extraArgs)
	command := formatRunCommand(gui.Config.PodmanBinary(), argv)

	return gui.createConfirmationPanel(
		gui.trString(gui.Tr.CommitImageConfirmTitle, "Confirm commit"),
		fmt.Sprintf("%s\n\n%s", command, gui.trString(gui.Tr.CommitImageConfirm, "Commit this container to an image?")),
		func(g *gocui.Gui, v *gocui.View) error {
			return gui.executeCommitImage(state, ctr)
		},
		nil,
	)
}

func (gui *Gui) executeCommitImage(state *runContainerState, ctr *commands.Container) error {
	argv := state.commitArgv(commitContainerRef(ctr), state.extraArgs)
	command := exec.Command(gui.Config.PodmanBinary(), argv...)

	return gui.WithWaitingStatus(gui.trString(gui.Tr.CommitImageRunningStatus, "Committing container"), func() error {
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
