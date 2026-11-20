package gui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/jesseduffield/gocui"
)

// handleQuadletMenu is the entry point behind Q. Exporting a container and
// managing already installed units are one menu so Q does not need two keys.
func (gui *Gui) handleQuadletMenu(g *gocui.Gui, v *gocui.View) error {
	return gui.Menu(CreateMenuOptions{
		Title: gui.Tr.QuadletMenuTitle,
		Items: []*types.MenuItem{
			{
				LabelColumns: []string{gui.Tr.QuadletExport},
				OnPress:      func() error { return gui.handleContainerQuadletExport(g, v) },
			},
			{
				LabelColumns: []string{gui.Tr.QuadletManage},
				OnPress:      gui.handleQuadletManager,
			},
		},
	})
}

func (gui *Gui) handleQuadletManager() error {
	files, err := gui.quadletFiles()
	if err != nil {
		return gui.createErrorPanel(err.Error())
	}
	if len(files) == 0 {
		gui.flashStatus(fmt.Sprintf(gui.Tr.QuadletNoUnits, gui.quadletDir()))

		return nil
	}

	items := make([]*types.MenuItem, 0, len(files))
	for _, path := range files {
		path := path
		items = append(items, &types.MenuItem{
			LabelColumns: []string{filepath.Base(path), color.New(color.FgYellow).Sprint(gui.quadletUnitName(path))},
			OnPress:      func() error { return gui.handleQuadletUnitMenu(path) },
		})
	}

	return gui.Menu(CreateMenuOptions{
		Title: fmt.Sprintf("%s — %s", gui.Tr.QuadletManage, gui.quadletDir()),
		Items: items,
	})
}

func (gui *Gui) quadletFiles() ([]string, error) {
	dir := gui.quadletDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".container") || strings.HasSuffix(name, ".pod") ||
			strings.HasSuffix(name, ".kube") || strings.HasSuffix(name, ".network") ||
			strings.HasSuffix(name, ".volume") {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)

	return files, nil
}

func (gui *Gui) quadletUnitName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + ".service"
}

func (gui *Gui) handleQuadletUnitMenu(path string) error {
	unit := gui.quadletUnitName(path)
	action := func(label string, args ...string) *types.MenuItem {
		return &types.MenuItem{
			LabelColumns: []string{label, strings.Join(args, " ")},
			OnPress: func() error {
				return gui.WithWaitingStatus(gui.Tr.QuadletSystemctlStatus, func() error {
					output, err := gui.runSystemctl(args...)
					if err != nil {
						return gui.createErrorPanel(strings.TrimSpace(output + "\n" + err.Error()))
					}
					if output != "" {
						gui.flashStatus(strings.TrimSpace(output))
					}

					return nil
				})
			},
		}
	}

	items := []*types.MenuItem{
		action(gui.Tr.Start, "start", unit),
		action(gui.Tr.Stop, "stop", unit),
		action(gui.Tr.Restart, "restart", unit),
		action(gui.Tr.QuadletEnable, "enable", unit),
		action(gui.Tr.QuadletDisable, "disable", unit),
		{
			LabelColumns: []string{gui.Tr.QuadletReload},
			OnPress: func() error {
				return gui.WithWaitingStatus(gui.Tr.QuadletSystemctlStatus, func() error {
					output, err := gui.runSystemctl("daemon-reload")
					if err != nil {
						return gui.createErrorPanel(strings.TrimSpace(output + "\n" + err.Error()))
					}
					gui.flashStatus(gui.Tr.QuadletReloaded)

					return nil
				})
			},
		},
		{
			LabelColumns: []string{gui.Tr.QuadletLogs},
			OnPress:      func() error { return gui.showQuadletLogs(unit) },
		},
		{
			LabelColumns: []string{gui.Tr.EditConfig},
			OnPress:      func() error { return gui.editFile(path) },
		},
		{
			LabelColumns: []string{gui.Tr.Remove, color.New(color.FgRed).Sprint(path)},
			OnPress: func() error {
				return gui.createConfirmationPanel(gui.Tr.QuadletDeleteUnit, fmt.Sprintf(gui.Tr.QuadletConfirmDeleteUnit, path), func(g *gocui.Gui, v *gocui.View) error {
					if err := os.Remove(path); err != nil {
						return gui.createErrorPanel(err.Error())
					}
					_, _ = gui.runSystemctl("daemon-reload")
					gui.flashStatus(fmt.Sprintf(gui.Tr.QuadletDeleted, path))

					return nil
				}, nil)
			},
		},
	}

	return gui.Menu(CreateMenuOptions{Title: unit, Items: items})
}

func (gui *Gui) runSystemctl(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", append([]string{"--user"}, args...)...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("systemctl timed out")
	}

	return string(output), err
}

func (gui *Gui) showQuadletLogs(unit string) error {
	return gui.QueueTask(gui.NewRenderStringTask(RenderStringTaskOpts{
		Wrap: true,
		GetStrContent: func() string {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, "journalctl", "--user", "-u", unit, "-n", "200", "--no-pager")
			output, err := cmd.CombinedOutput()
			if err != nil && len(output) == 0 {
				return err.Error()
			}

			return utils.ColoredString("# journalctl --user -u "+unit, color.FgYellow) + "\n\n" + string(output)
		},
	}))
}
