package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/jesseduffield/gocui"
)

// handleContainerQuadletExport writes a Quadlet `.container` unit for the selected
// container. Like the kube export it previews the file and only writes after a
// confirmation naming the full path.
func (gui *Gui) handleContainerQuadletExport(_ *gocui.Gui, _ *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	if !ctr.DetailsLoaded() {
		if err := gui.PodmanCommand.RefreshContainerDetails([]*commands.Container{ctr}); err != nil {
			return gui.createErrorPanel(err.Error())
		}
	}

	unit := commands.BuildQuadletContainer(ctr)
	if strings.TrimSpace(unit) == "" {
		return gui.createErrorPanel(gui.Tr.QuadletNeedsDetails)
	}

	path := gui.quadletFilePath(ctr.Name)
	_, statErr := os.Stat(path)

	gui.reRenderStringMain(unit)
	gui.promptQuadletSave(unit, path, statErr == nil)

	return nil
}

func (gui *Gui) promptQuadletSave(unit, path string, exists bool) {
	prompt := gui.quadletSavePrompt(path, exists)

	gui.g.Update(func(g *gocui.Gui) error {
		return gui.createConfirmationPanel(gui.Tr.QuadletTitle, prompt, func(g *gocui.Gui, v *gocui.View) error {
			if err := commands.WriteManifestFile(path, []byte(unit), true); err != nil {
				return gui.createErrorPanel(err.Error())
			}

			gui.flashStatus(fmt.Sprintf(gui.Tr.QuadletSavedStatus, path))

			return nil
		}, func(g *gocui.Gui, v *gocui.View) error {
			return gui.reRenderMain()
		})
	})
}

// quadletSavePrompt always contains the full path and the plaintext warning.
func (gui *Gui) quadletSavePrompt(path string, exists bool) string {
	format := gui.Tr.KubeSavePrompt
	if exists {
		format = gui.Tr.KubeOverwritePrompt
	}

	return fmt.Sprintf(format, path) + "\n\n" + gui.Tr.QuadletPlaintextWarning
}

// quadletDir resolves gui.quadletDir, defaulting to <user config dir>/containers/systemd.
func (gui *Gui) quadletDir() string {
	dir := gui.Config.UserConfig.Gui.QuadletDir
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, ".config")
		}

		return filepath.Join(base, "containers", "systemd")
	}

	if dir == "~" || strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(dir, "~"), "/"))
		}
	}

	return dir
}

func (gui *Gui) quadletFilePath(containerName string) string {
	return filepath.Join(gui.quadletDir(), commands.SafeKubeFilename(containerName)+".container")
}
