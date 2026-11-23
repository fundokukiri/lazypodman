package gui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/fatih/color"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/config"
	"github.com/fundokukiri/lazypodman/pkg/gui/panels"
	"github.com/fundokukiri/lazypodman/pkg/gui/presentation"
	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/jesseduffield/gocui"
	"github.com/samber/lo"
)

func (gui *Gui) getImagesPanel() *panels.SideListPanel[*commands.Image] {
	noneLabel := "<none>"

	return &panels.SideListPanel[*commands.Image]{
		ContextState: &panels.ContextState[*commands.Image]{
			GetMainTabs: func() []panels.MainTab[*commands.Image] {
				return []panels.MainTab[*commands.Image]{
					{
						Key:    "config",
						Title:  gui.Tr.ConfigTitle,
						Render: gui.renderImageConfigTask,
					},
				}
			},
			GetItemContextCacheKey: func(image *commands.Image) string {
				return "images-" + image.ID
			},
		},
		ListPanel: panels.ListPanel[*commands.Image]{
			List: panels.NewFilteredList[*commands.Image](),
			View: gui.Views.Images,
		},
		NoItemsMessage: gui.Tr.NoImages,
		Gui:            gui.intoInterface(),
		Sort: func(a *commands.Image, b *commands.Image) bool {
			if a.Name == noneLabel && b.Name != noneLabel {
				return false
			}

			if a.Name != noneLabel && b.Name == noneLabel {
				return true
			}

			if a.Name != b.Name {
				return a.Name < b.Name
			}

			if a.Tag != b.Tag {
				return a.Tag < b.Tag
			}

			return a.ID < b.ID
		},
		GetTableCells: func(image *commands.Image) []string {
			cells := presentation.GetImageDisplayStrings(image)
			// The lock is part of the name cell so the table keeps its width.
			cells[0] = gui.imageLockPrefix(image.ID) + cells[0]

			return cells
		},
	}
}

func (gui *Gui) renderImageConfigTask(image *commands.Image) tasks.TaskFunc {
	return gui.NewRenderStringTask(RenderStringTaskOpts{
		GetStrContent: func() string { return gui.imageConfigStr(image) },
		Autoscroll:    false,
		Wrap:          false, // don't care what your config is this page is ugly without wrapping
	})
}

func (gui *Gui) imageConfigStr(image *commands.Image) string {
	padding := 10
	output := ""
	output += utils.WithPadding("Name: ", padding) + image.Name + "\n"
	output += utils.WithPadding("ID: ", padding) + image.Image.ID + "\n"
	output += utils.WithPadding("Tags: ", padding) + utils.ColoredString(strings.Join(image.Image.RepoTags, ", "), color.FgGreen) + "\n"
	output += utils.WithPadding("Size: ", padding) + utils.FormatDecimalBytes(image.Image.Size) + "\n"
	output += utils.WithPadding("Created: ", padding) + fmt.Sprintf("%v", time.Unix(image.Image.Created, 0).Format(time.RFC1123)) + "\n"

	history, err := image.RenderHistory()
	if err != nil {
		gui.Log.Error(err)
	}

	output += "\n\n" + history

	return output
}

func (gui *Gui) reloadImages() error {
	if err := gui.refreshStateImages(); err != nil {
		return err
	}

	return gui.Panels.Images.RerenderList()
}

func (gui *Gui) refreshStateImages() error {
	images, err := gui.PodmanCommand.RefreshImages()
	if err != nil {
		return err
	}

	gui.Panels.Images.SetItems(images)

	return nil
}

func (gui *Gui) FilterString(view *gocui.View) string {
	if gui.State.Filter.panel != nil && gui.State.Filter.panel.GetView() != view {
		return ""
	}

	return gui.State.Filter.needle
}

func (gui *Gui) handleImagesRemoveMenu(g *gocui.Gui, v *gocui.View) error {
	type removeImageOption struct {
		description   string
		command       string
		configOptions image.RemoveOptions
	}

	img, err := gui.Panels.Images.GetSelectedItem()
	if err != nil {
		return nil
	}

	shortSha := img.ID[7:17]

	// TODO: have a way of toggling in a menu instead of showing each permutation as a separate menu item
	options := []*removeImageOption{
		{
			description:   gui.Tr.Remove,
			command:       gui.Config.PodmanBinary() + " image rm " + shortSha,
			configOptions: image.RemoveOptions{PruneChildren: true, Force: false},
		},
		{
			description:   gui.Tr.RemoveWithoutPrune,
			command:       gui.Config.PodmanBinary() + " image rm --no-prune " + shortSha,
			configOptions: image.RemoveOptions{PruneChildren: false, Force: false},
		},
		{
			description:   gui.Tr.RemoveWithForce,
			command:       gui.Config.PodmanBinary() + " image rm --force " + shortSha,
			configOptions: image.RemoveOptions{PruneChildren: true, Force: true},
		},
		{
			description:   gui.Tr.RemoveWithoutPruneWithForce,
			command:       gui.Config.PodmanBinary() + " image rm --no-prune --force " + shortSha,
			configOptions: image.RemoveOptions{PruneChildren: false, Force: true},
		},
	}

	menuItems := lo.Map(options, func(option *removeImageOption, _ int) *types.MenuItem {
		return &types.MenuItem{
			LabelColumns: []string{
				option.description,
				color.New(color.FgRed).Sprint(option.command),
			},
			OnPress: func() error {
				return gui.confirmLockedRemovals(gui.imageRemovalLocks(img), func() error {
					if err := img.Remove(option.configOptions); err != nil {
						return gui.createErrorPanel(err.Error())
					}

					return nil
				})
			},
		}
	})

	return gui.Menu(CreateMenuOptions{
		Title: "",
		Items: menuItems,
	})
}

// imagePullRef returns the repository reference to pull, or "" when the image
// has no tag Podman could update.
func imagePullRef(img *commands.Image) string {
	for _, tag := range img.Image.RepoTags {
		if tag != "" && !strings.Contains(tag, "<none>") {
			return tag
		}
	}

	if img.Name != "" && img.Name != "none" {
		if img.Tag != "" {
			return img.Name + ":" + img.Tag
		}
		return img.Name
	}

	return ""
}

// findImageByRef finds the image that now owns ref after a pull.
func findImageByRef(images []*commands.Image, ref string) *commands.Image {
	for _, img := range images {
		for _, tag := range img.Image.RepoTags {
			if tag == ref {
				return img
			}
		}
		if img.Name == ref || (img.Tag != "" && img.Name+":"+img.Tag == ref) {
			return img
		}
	}

	return nil
}

func (gui *Gui) handleImagesUpdate(g *gocui.Gui, v *gocui.View) error {
	img, err := gui.Panels.Images.GetSelectedItem()
	if err != nil {
		return nil
	}

	ref := imagePullRef(img)
	if ref == "" {
		return gui.createErrorPanel(gui.trString(gui.Tr.UpdateImageNoTag, "This image has no repository tag to pull"))
	}
	oldID := img.ID

	command := exec.Command(gui.Config.PodmanBinary(), "pull", ref)
	return gui.WithWaitingStatus(gui.trString(gui.Tr.UpdatingImageStatus, "updating image"), func() error {
		output, err := command.CombinedOutput()
		if err != nil {
			message := strings.TrimSpace(string(output))
			if message == "" {
				message = err.Error()
			}
			return fmt.Errorf("%s", message)
		}

		if err := gui.reloadImages(); err != nil {
			return err
		}

		updated := findImageByRef(gui.Panels.Images.List.GetAllItems(), ref)
		if updated != nil && updated.ID == oldID {
			gui.flashStatus(fmt.Sprintf(gui.trString(gui.Tr.ImageAlreadyUpToDate, "image already up to date: %s"), ref))
		} else {
			gui.flashStatus(fmt.Sprintf(gui.trString(gui.Tr.ImageUpdated, "image updated: %s"), ref))
		}

		return nil
	})
}

func (gui *Gui) handlePruneImages() error {
	return gui.createConfirmationPanel(gui.Tr.Confirm, gui.Tr.ConfirmPruneImages, func(g *gocui.Gui, v *gocui.View) error {
		return gui.WithWaitingStatus(gui.Tr.PruningStatus, func() error {
			err := gui.PodmanCommand.PruneImages()
			if err != nil {
				return gui.createErrorPanel(err.Error())
			}
			return gui.reloadImages()
		})
	}, nil)
}

func (gui *Gui) handleImagesCustomCommand(g *gocui.Gui, v *gocui.View) error {
	img, err := gui.Panels.Images.GetSelectedItem()
	if err != nil {
		return nil
	}

	commandObject := gui.PodmanCommand.NewCommandObject(commands.CommandObject{
		Image: img,
	})

	customCommands := gui.Config.UserConfig.CustomCommands.Images

	return gui.createCustomCommandMenu(customCommands, commandObject)
}

func (gui *Gui) handleImagesBulkCommand(g *gocui.Gui, v *gocui.View) error {
	baseBulkCommands := []config.CustomCommand{
		{
			Name:             gui.Tr.PruneImages,
			InternalFunction: gui.handlePruneImages,
		},
	}

	bulkCommands := append(baseBulkCommands, gui.Config.UserConfig.BulkCommands.Images...)
	commandObject := gui.PodmanCommand.NewCommandObject(commands.CommandObject{})

	return gui.createBulkCommandMenu(bulkCommands, commandObject)
}
