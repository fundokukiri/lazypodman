package gui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	"github.com/jesseduffield/gocui"
)

// handlePodsKubeMenu is the single entry point for kube actions: generating a
// manifest, playing one and downing one. One menu keeps the key table small and
// makes the three choices discoverable.
func (gui *Gui) handlePodsKubeMenu(g *gocui.Gui, v *gocui.View) error {
	items := []*types.MenuItem{
		{
			LabelColumns: []string{gui.Tr.KubeGenerateMenu},
			OnPress:      func() error { return gui.handleKubeGenerate() },
		},
		{
			LabelColumns: []string{gui.Tr.KubePlayMenu},
			OnPress:      func() error { return gui.kubeFileMenu(false) },
		},
		{
			LabelColumns: []string{gui.Tr.KubeDownMenu},
			OnPress:      func() error { return gui.kubeFileMenu(true) },
		},
	}

	return gui.Menu(CreateMenuOptions{Title: gui.Tr.KubeMenuTitle, Items: items})
}

// handleKubeGenerate exports the selected pod, shows the YAML as a preview and
// only writes it after an explicit confirmation.
func (gui *Gui) handleKubeGenerate() error {
	pod := gui.selectedPod()

	return gui.WithWaitingStatus(gui.Tr.KubeGeneratingStatus, func() error {
		yaml, path, exists, err := gui.generateKube(pod)
		if err != nil {
			return gui.createErrorPanel(err.Error())
		}

		gui.reRenderStringMain(yaml)
		gui.promptKubeSave(yaml, path, exists)

		return nil
	})
}

// generateKube does the process work and returns everything the confirmation needs.
// It performs no UI work, so it is testable and safe off the UI goroutine.
func (gui *Gui) generateKube(pod *commands.Pod) (yaml string, path string, exists bool, err error) {
	if pod == nil {
		return "", "", false, fmt.Errorf("%s", gui.Tr.NoPods)
	}

	capability, prefix := gui.PodmanCommand.KubeGenerateCapability(context.Background())
	if capability.State != commands.CapabilityAvailable {
		return "", "", false, fmt.Errorf("%s %s", gui.Tr.KubeGenerateOff, capability.Reason)
	}

	yaml, err = gui.PodmanCommand.Pods().KubeGenerate(context.Background(), prefix, pod.Name)
	if err != nil {
		return "", "", false, err
	}

	if strings.TrimSpace(yaml) == "" {
		return "", "", false, fmt.Errorf("podman returned no YAML")
	}

	path = gui.kubeFilePath(pod.Name)
	_, statErr := os.Stat(path)

	return yaml, path, statErr == nil, nil
}

func (gui *Gui) promptKubeSave(yaml, path string, exists bool) {
	title := gui.Tr.KubeMenuTitle
	prompt := gui.kubeSavePrompt(path, exists)

	gui.g.Update(func(g *gocui.Gui) error {
		return gui.createConfirmationPanel(title, prompt, func(g *gocui.Gui, v *gocui.View) error {
			if err := commands.WriteKubeFile(path, []byte(yaml), true); err != nil {
				return gui.createErrorPanel(err.Error())
			}

			gui.flashStatus(fmt.Sprintf(gui.Tr.KubeSavedStatus, path))

			return nil
		}, func(g *gocui.Gui, v *gocui.View) error {
			return gui.reRenderMain()
		})
	})
}

// kubeSavePrompt is the confirmation text. It always contains the full path and
// warns that env vars are stored in plaintext.
func (gui *Gui) kubeSavePrompt(path string, exists bool) string {
	format := gui.Tr.KubeSavePrompt
	if exists {
		format = gui.Tr.KubeOverwritePrompt
	}

	return fmt.Sprintf(format, path) + "\n\n" + gui.Tr.KubePlaintextWarning
}

// kubeFileMenu lists the manifests in gui.kubeDir and asks for a confirmation with
// the full path before doing anything. Play and down share this path; only the
// prompt and the argv differ.
func (gui *Gui) kubeFileMenu(down bool) error {
	return gui.WithWaitingStatus(gui.Tr.KubeProbingStatus, func() error {
		capability, prefix := gui.PodmanCommand.KubePlayCapability(context.Background())
		if capability.State != commands.CapabilityAvailable {
			gui.flashStatus(gui.Tr.KubePlayOff + " " + capability.Reason)

			return nil
		}

		files, err := commands.ScanKubeDir(gui.kubeDir())
		if err != nil {
			gui.flashStatus(err.Error())

			return nil
		}

		if len(files) == 0 {
			gui.flashStatus(fmt.Sprintf(gui.Tr.KubeNoFiles, gui.kubeDir()))

			return nil
		}

		gui.g.Update(func(g *gocui.Gui) error {
			return gui.openKubeFileMenu(files, prefix, down)
		})

		return nil
	})
}

func (gui *Gui) openKubeFileMenu(files []commands.KubeFile, prefix []string, down bool) error {
	items := make([]*types.MenuItem, 0, len(files))

	for _, file := range files {
		file := file
		items = append(items, &types.MenuItem{
			LabelColumns: []string{file.Name, file.Path},
			OnPress: func() error {
				return gui.confirmKubeRun(prefix, file.Path, down)
			},
		})
	}

	return gui.Menu(CreateMenuOptions{Title: gui.Tr.KubeMenuTitle, Items: items})
}

func (gui *Gui) confirmKubeRun(prefix []string, path string, down bool) error {
	prompt := fmt.Sprintf(gui.Tr.KubePlayConfirm, path)
	if down {
		prompt = fmt.Sprintf(gui.Tr.KubeDownConfirm, path)
	}

	return gui.createConfirmationPanel(gui.Tr.KubeMenuTitle, prompt, func(g *gocui.Gui, v *gocui.View) error {
		return gui.runKubeStreamTask(prefix, path, down)
	}, nil)
}

// runKubeStreamTask streams play/down output verbatim into the main panel. The
// task is cancellable: a newer task (or leaving the tab) cancels the context, which
// kills the podman child. No enablement timeout, no hidden flags.
func (gui *Gui) runKubeStreamTask(prefix []string, path string, down bool) error {
	args := commands.KubePlayArgs(path)
	header := fmt.Sprintf(gui.Tr.KubePlayHeader, path)

	if down {
		args = commands.KubeDownArgs(path)
		header = fmt.Sprintf(gui.Tr.KubeDownHeader, path)
	}

	return gui.QueueTask(func(ctx context.Context) {
		var mu sync.Mutex

		var builder strings.Builder

		builder.WriteString(header + "\n\n")

		gui.clearMainView()
		gui.reRenderStringMain(builder.String())

		err := gui.PodmanCommand.Pods().StreamKube(ctx, prefix, args, func(line string) {
			mu.Lock()
			builder.WriteString(line + "\n")
			contents := builder.String()
			mu.Unlock()

			gui.reRenderStringMain(contents)
		})
		if err != nil {
			mu.Lock()
			builder.WriteString("\n" + err.Error() + "\n")
			contents := builder.String()
			mu.Unlock()

			gui.reRenderStringMain(contents)
		}

		if refreshErr := gui.refreshPods(); refreshErr != nil {
			gui.Log.Warn(refreshErr)
		}
	})
}

// kubeDir resolves gui.kubeDir, defaulting to <user config dir>/lazypodman/kube and
// expanding a leading ~.
func (gui *Gui) kubeDir() string {
	dir := gui.Config.UserConfig.Gui.KubeDir
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, ".config")
		}

		return filepath.Join(base, "lazypodman", "kube")
	}

	if dir == "~" || strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(dir, "~"), "/"))
		}
	}

	return dir
}

// kubeFilePath is the save target for a pod. The filename is sanitised so a pod
// name can never point outside kubeDir.
func (gui *Gui) kubeFilePath(podName string) string {
	return filepath.Join(gui.kubeDir(), commands.SafeKubeFilename(podName)+".yaml")
}
