package gui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/jesseduffield/gocui"
)

const containerFilesTimeout = 30 * time.Second

// handleContainerFilesMenu exposes the file operations that otherwise require
// remembering `podman cp` and `podman diff` syntax. It is deliberately a menu:
// paths are argv arguments, never a shell string.
func (gui *Gui) handleContainerFilesMenu(g *gocui.Gui, v *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	return gui.Menu(CreateMenuOptions{
		Title: fmt.Sprintf("%s — %s", gui.Tr.ContainerFiles, ctr.Name),
		Items: []*types.MenuItem{
			{
				LabelColumns: []string{gui.Tr.FilesList},
				OnPress:      func() error { return gui.promptContainerList(ctr) },
			},
			{
				LabelColumns: []string{gui.Tr.FilesDownload},
				OnPress:      func() error { return gui.promptContainerDownload(ctr) },
			},
			{
				LabelColumns: []string{gui.Tr.FilesUpload},
				OnPress:      func() error { return gui.promptContainerUpload(ctr) },
			},
			{
				LabelColumns: []string{gui.Tr.FilesDiff},
				OnPress:      func() error { return gui.showContainerDiff(ctr) },
			},
		},
	})
}

func (gui *Gui) promptContainerList(ctr *commands.Container) error {
	return gui.createPromptPanelWithInitial(gui.Tr.FilesContainerPath, "/", func(g *gocui.Gui, v *gocui.View) error {
		path := strings.TrimSpace(gui.trimmedContent(v))
		if path == "" {
			return nil
		}

		return gui.QueueTask(gui.NewRenderStringTask(RenderStringTaskOpts{
			Wrap: false,
			GetStrContent: func() string {
				output, err := gui.runPodmanOutput("exec", ctr.ID, "ls", "-la", path)
				if err != nil {
					return strings.TrimSpace(output + "\n" + err.Error())
				}

				return utils.ColoredString("# podman exec "+ctr.Name+" ls -la "+path, color.FgYellow) + "\n\n" + output
			},
		}))
	})
}

func (gui *Gui) promptContainerDownload(ctr *commands.Container) error {
	return gui.createPromptPanelWithInitial(gui.Tr.FilesContainerPath, "/", func(g *gocui.Gui, v *gocui.View) error {
		source := strings.TrimSpace(gui.trimmedContent(v))
		if source == "" {
			return nil
		}

		return gui.createPromptPanelWithInitial(gui.Tr.FilesHostPath, filepath.Base(source), func(g *gocui.Gui, v *gocui.View) error {
			destination := strings.TrimSpace(gui.trimmedContent(v))
			if destination == "" {
				return nil
			}

			run := func() error {
				return gui.WithWaitingStatus(gui.Tr.FilesCopying, func() error {
					output, err := gui.runPodmanOutput("cp", ctr.ID+":"+source, destination)
					if err != nil {
						return gui.createErrorPanel(strings.TrimSpace(output + "\n" + err.Error()))
					}
					gui.flashStatus(fmt.Sprintf(gui.Tr.FilesCopied, source, destination))

					return nil
				})
			}

			if _, err := os.Stat(destination); err == nil {
				return gui.createConfirmationPanel(gui.Tr.FilesOverwrite, fmt.Sprintf(gui.Tr.FilesOverwritePrompt, destination), func(g *gocui.Gui, v *gocui.View) error { return run() }, nil)
			}

			return run()
		})
	})
}

func (gui *Gui) promptContainerUpload(ctr *commands.Container) error {
	return gui.createPromptPanelWithInitial(gui.Tr.FilesHostPath, "", func(g *gocui.Gui, v *gocui.View) error {
		source := strings.TrimSpace(gui.trimmedContent(v))
		if source == "" {
			return nil
		}
		if _, err := os.Stat(source); err != nil {
			return gui.createErrorPanel(err.Error())
		}

		return gui.createPromptPanelWithInitial(gui.Tr.FilesContainerPath, "/tmp/"+filepath.Base(source), func(g *gocui.Gui, v *gocui.View) error {
			destination := strings.TrimSpace(gui.trimmedContent(v))
			if destination == "" {
				return nil
			}

			return gui.createConfirmationPanel(gui.Tr.FilesUpload, fmt.Sprintf(gui.Tr.FilesUploadPrompt, source, ctr.Name+":"+destination), func(g *gocui.Gui, v *gocui.View) error {
				return gui.WithWaitingStatus(gui.Tr.FilesCopying, func() error {
					output, err := gui.runPodmanOutput("cp", source, ctr.ID+":"+destination)
					if err != nil {
						return gui.createErrorPanel(strings.TrimSpace(output + "\n" + err.Error()))
					}
					gui.flashStatus(fmt.Sprintf(gui.Tr.FilesCopied, source, destination))

					return nil
				})
			}, nil)
		})
	})
}

func (gui *Gui) showContainerDiff(ctr *commands.Container) error {
	return gui.QueueTask(gui.NewRenderStringTask(RenderStringTaskOpts{
		Wrap: false,
		GetStrContent: func() string {
			output, err := gui.runPodmanOutput("diff", ctr.ID)
			if err != nil {
				return strings.TrimSpace(output + "\n" + err.Error())
			}
			if strings.TrimSpace(output) == "" {
				return gui.Tr.FilesNoChanges
			}

			return utils.ColoredString("# podman diff "+ctr.Name, color.FgYellow) + "\n\n" + output
		},
	}))
}

func (gui *Gui) runPodmanOutput(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), containerFilesTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, gui.Config.PodmanBinary(), args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("podman timed out")
	}

	return string(output), err
}
