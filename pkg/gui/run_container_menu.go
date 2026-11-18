package gui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
	"github.com/jesseduffield/gocui"
)

func (gui *Gui) handleImagesRunContainer(g *gocui.Gui, v *gocui.View) error {
	image, err := gui.Panels.Images.GetSelectedItem()
	if err != nil {
		return nil
	}

	state := newRunContainerState(runImageRef(image))

	return gui.showRunContainerMenu(state)
}

func runImageRef(image *commands.Image) string {
	for _, tag := range image.Image.RepoTags {
		if tag != "" && !strings.Contains(tag, "<none>") {
			return tag
		}
	}

	if image.Name != "" && image.Name != "none" && image.Tag != "" {
		return image.Name + ":" + image.Tag
	}

	return image.ID
}

func (gui *Gui) runLabel(key string) string {
	if gui.Tr != nil && gui.Tr.RunContainerLabels != nil {
		if label := gui.Tr.RunContainerLabels[key]; label != "" {
			return label
		}
	}

	if sets := i18n.GetTranslationSets(); sets != nil {
		english := sets["en"]
		if label := english.RunContainerLabels[key]; label != "" {
			return label
		}
	}

	return key
}

func (gui *Gui) trString(value, fallback string) string {
	if value != "" {
		return value
	}

	return fallback
}

func (gui *Gui) runValueLabel(state *runContainerState, field runField) string {
	switch field.Kind {
	case runListField:
		values := state.lists[field.Key]
		if len(values) == 0 {
			return gui.trString(gui.Tr.RunContainerEmpty, "none")
		}
		return fmt.Sprintf("%d %s", len(values), gui.trString(gui.Tr.RunContainerItems, "items"))
	case runToggleField:
		if state.toggles[field.Key] {
			return gui.trString(gui.Tr.RunContainerOn, "on")
		}
		return gui.trString(gui.Tr.RunContainerOff, "off")
	default:
		if value := state.text[field.Key]; value != "" {
			return value
		}
		return gui.trString(gui.Tr.RunContainerDefault, "default")
	}
}

func (gui *Gui) runFieldHint(field runField) string {
	switch field.Kind {
	case runToggleField:
		return "space"
	case runListField:
		return "enter/a"
	default:
		return "enter"
	}
}

func (gui *Gui) showRunContainerMenu(state *runContainerState) error {
	items := []*types.MenuItem{
		{
			LabelColumns: []string{
				gui.trString(func() string {
					if gui.Tr != nil {
						return gui.Tr.RunContainerImage
					}
					return ""
				}(), "Image"),
				state.image,
				"enter",
			},
			OnPress: func() error {
				return gui.editRunContainerValue("image", state.image, func(value string) {
					if value != "" {
						state.image = value
					}
				}, func() error { return gui.showRunContainerMenu(state) })
			},
		},
	}

	for _, field := range runContainerFields {
		field := field
		items = append(items, &types.MenuItem{
			LabelColumns: []string{gui.runLabel(field.Key), gui.runValueLabel(state, field), gui.runFieldHint(field)},
			OnPress: func() error {
				return gui.activateRunContainerField(state, field, func() error {
					return gui.showRunContainerMenu(state)
				})
			},
		})
	}

	items = append(items,
		&types.MenuItem{
			LabelColumns: []string{gui.trString(gui.Tr.RunContainerAdvanced, "Advanced"), "", "enter"},
			OnPress:      func() error { return gui.showRunContainerAdvanced(state) },
		},
		&types.MenuItem{
			LabelColumns: []string{gui.trString(gui.Tr.RunContainerRun, "Run"), "", "enter"},
			OnPress:      func() error { return gui.previewRunContainer(state) },
		},
	)

	return gui.Menu(CreateMenuOptions{Title: gui.trString(gui.Tr.RunContainerTitle, "Run container"), Items: items})
}

func (gui *Gui) showRunContainerAdvanced(state *runContainerState) error {
	items := []*types.MenuItem{
		{
			LabelColumns: []string{gui.trString(gui.Tr.RunContainerBack, "back")},
			OnPress:      func() error { return gui.showRunContainerMenu(state) },
		},
	}

	for _, group := range runContainerGroups {
		group := group
		items = append(items, &types.MenuItem{
			LabelColumns: []string{gui.runLabel(group.Key), "", "enter"},
			OnPress:      func() error { return gui.showRunContainerGroup(state, group) },
		})
	}

	items = append(items, &types.MenuItem{
		LabelColumns: []string{gui.trString(gui.Tr.RunContainerExtraArgs, "Extra args"), state.extraArgs, "enter"},
		OnPress: func() error {
			return gui.editRunContainerValue("extra_args", state.extraArgs, func(value string) {
				state.extraArgs = value
			}, func() error { return gui.showRunContainerAdvanced(state) })
		},
	})

	return gui.Menu(CreateMenuOptions{Title: gui.trString(gui.Tr.RunContainerAdvanced, "Advanced"), Items: items})
}

func (gui *Gui) showRunContainerGroup(state *runContainerState, group runFieldGroup) error {
	items := []*types.MenuItem{
		{
			LabelColumns: []string{gui.trString(gui.Tr.RunContainerBack, "back")},
			OnPress:      func() error { return gui.showRunContainerAdvanced(state) },
		},
	}

	for _, field := range group.Fields {
		field := field
		items = append(items, &types.MenuItem{
			LabelColumns: []string{gui.runLabel(field.Key), gui.runValueLabel(state, field), gui.runFieldHint(field)},
			OnPress: func() error {
				return gui.activateRunContainerField(state, field, func() error {
					return gui.showRunContainerGroup(state, group)
				})
			},
		})
	}

	return gui.Menu(CreateMenuOptions{Title: gui.runLabel(group.Key), Items: items})
}

func (gui *Gui) activateRunContainerField(state *runContainerState, field runField, back func() error) error {
	switch field.Kind {
	case runToggleField:
		state.toggles[field.Key] = !state.toggles[field.Key]
		return back()
	case runChoiceField:
		return gui.showRunContainerChoice(state, field, back)
	case runListField:
		return gui.showRunContainerList(state, field, back)
	default:
		return gui.editRunContainerValue(field.Key, state.text[field.Key], func(value string) {
			state.text[field.Key] = value
		}, back)
	}
}

func (gui *Gui) showRunContainerChoice(state *runContainerState, field runField, back func() error) error {
	items := []*types.MenuItem{
		{
			LabelColumns: []string{gui.trString(gui.Tr.RunContainerDefault, "default")},
			OnPress: func() error {
				delete(state.text, field.Key)
				return back()
			},
		},
	}

	for _, choice := range field.Choices {
		choice := choice
		items = append(items, &types.MenuItem{
			LabelColumns: []string{choice},
			OnPress: func() error {
				state.text[field.Key] = choice
				return back()
			},
		})
	}

	return gui.Menu(CreateMenuOptions{Title: gui.runLabel(field.Key), Items: items})
}

func (gui *Gui) showRunContainerList(state *runContainerState, field runField, back func() error) error {
	items := []*types.MenuItem{
		{
			LabelColumns: []string{gui.trString(gui.Tr.RunContainerBack, "back")},
			OnPress:      back,
		},
		{
			LabelColumns: []string{fmt.Sprintf("+ %s %s", gui.trString(gui.Tr.RunContainerAdd, "add"), gui.runLabel(field.Key))},
			OnPress: func() error {
				return gui.editRunContainerValue(field.Key, "", func(value string) {
					state.addListValue(field.Key, value)
				}, func() error { return gui.showRunContainerList(state, field, back) })
			},
		},
	}

	for index, value := range state.lists[field.Key] {
		index, value := index, value
		items = append(items, &types.MenuItem{
			LabelColumns: []string{value},
			OnPress:      func() error { return gui.showRunContainerListValue(state, field, index, value, back) },
		})
	}

	return gui.Menu(CreateMenuOptions{Title: gui.runLabel(field.Key), Items: items})
}

func (gui *Gui) showRunContainerListValue(state *runContainerState, field runField, index int, value string, back func() error) error {
	items := []*types.MenuItem{
		{
			LabelColumns: []string{gui.trString(gui.Tr.RunContainerBack, "back")},
			OnPress:      func() error { return gui.showRunContainerList(state, field, back) },
		},
		{
			LabelColumns: []string{gui.trString(gui.Tr.RunContainerEdit, "edit"), value},
			OnPress: func() error {
				return gui.editRunContainerValue(field.Key, value, func(newValue string) {
					state.setListValue(field.Key, index, newValue)
				}, func() error { return gui.showRunContainerList(state, field, back) })
			},
		},
		{
			LabelColumns: []string{gui.trString(gui.Tr.RunContainerDelete, "delete"), value},
			OnPress: func() error {
				state.removeListValue(field.Key, index)
				return gui.showRunContainerList(state, field, back)
			},
		},
	}

	return gui.Menu(CreateMenuOptions{Title: gui.runLabel(field.Key), Items: items})
}

func (gui *Gui) editRunContainerValue(key, initial string, apply func(string), back func() error) error {
	return gui.createPromptPanelWithInitial(gui.runLabel(key), initial, func(g *gocui.Gui, v *gocui.View) error {
		apply(strings.TrimSpace(gui.trimmedContent(v)))
		return back()
	})
}

func formatRunCommand(binary string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, binary)
	for _, arg := range args {
		parts = append(parts, quoteRunArg(arg))
	}

	return strings.Join(parts, " ")
}

func quoteRunArg(arg string) string {
	if arg != "" && !strings.ContainsAny(arg, " \t\n'\"\\$`;&|<>()*?[]{}") {
		return arg
	}

	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

func (gui *Gui) previewRunContainer(state *runContainerState) error {
	argv := state.argv(state.extraArgs)
	command := formatRunCommand(gui.Config.PodmanBinary(), argv)

	return gui.createConfirmationPanel(
		gui.trString(gui.Tr.RunContainerConfirmTitle, "Confirm"),
		fmt.Sprintf("%s\n\n%s", command, gui.trString(gui.Tr.RunContainerConfirm, "Run this container?")),
		func(g *gocui.Gui, v *gocui.View) error {
			return gui.executeRunContainer(state)
		},
		nil,
	)
}

func (gui *Gui) executeRunContainer(state *runContainerState) error {
	argv := state.argv(state.extraArgs)
	command := exec.Command(gui.Config.PodmanBinary(), argv...)

	if !state.toggles["detach"] {
		if err := gui.runSubprocess(command); err != nil {
			return err
		}
		return gui.refreshContainersAndServices()
	}

	return gui.WithWaitingStatus(gui.trString(gui.Tr.RunContainerRunningStatus, "Starting container"), func() error {
		output, err := command.CombinedOutput()
		if err != nil {
			message := strings.TrimSpace(string(output))
			if message == "" {
				message = err.Error()
			}
			return fmt.Errorf("%s", message)
		}

		return gui.refreshContainersAndServices()
	})
}
