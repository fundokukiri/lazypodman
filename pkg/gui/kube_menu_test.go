package gui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/config"
)

func newKubeGui(client commands.PodmanClient, kubeDir string) *Gui {
	gui := newHealthGui(client)
	gui.Config = &config.AppConfig{UserConfig: &config.UserConfig{Gui: config.GuiConfig{KubeDir: kubeDir}}}

	return gui
}

func TestKubeDirDefaultsToUserConfig(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)

	gui := newKubeGui(&fakeHealthPodClient{}, "")

	assert.Equal(t, filepath.Join(base, "lazypodman", "kube"), gui.kubeDir())
}

func TestKubeDirExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	gui := newKubeGui(&fakeHealthPodClient{}, "~/kube")

	assert.Equal(t, filepath.Join(home, "kube"), gui.kubeDir())
}

func TestKubeDirKeepsAbsolutePath(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, "/srv/kube")

	assert.Equal(t, "/srv/kube", gui.kubeDir())
}

func TestKubeSavePrompt(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, "/srv/kube")

	path := filepath.Join("/srv/kube", "web.yaml")

	prompt := gui.kubeSavePrompt(path, false)
	assert.Contains(t, prompt, path)
	assert.Contains(t, prompt, "plaintext")

	overwrite := gui.kubeSavePrompt(path, true)
	assert.Contains(t, overwrite, "File exists")
	assert.Contains(t, overwrite, path)
}

func TestGenerateKubeWithoutPodSpawnsNothing(t *testing.T) {
	client := &fakeHealthPodClient{}
	gui := newKubeGui(client, t.TempDir())

	_, _, _, err := gui.generateKube(nil)

	assert.Error(t, err)
	assert.Equal(t, 0, client.kubeGenerateCalls)
	assert.Equal(t, 0, client.probeSubcommandCalls)
}

func TestGenerateKubeOffSpawnsNothing(t *testing.T) {
	client := &fakeHealthPodClient{}
	gui := newKubeGui(client, t.TempDir())
	gui.PodmanCommand.SetCapabilities(commands.Capabilities{
		CLI: commands.Capability{State: commands.CapabilityAvailable},
		KubeGenerate: commands.Capability{
			State:  commands.CapabilityUnavailable,
			Reason: "no kube subcommand",
		},
	})

	pod := &commands.Pod{Name: "web"}
	_, _, _, err := gui.generateKube(pod)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kube generate off: no kube subcommand")
	assert.Equal(t, 0, client.kubeGenerateCalls)
}

func TestGenerateKubeWritesPreviewPath(t *testing.T) {
	dir := t.TempDir()
	client := &fakeHealthPodClient{kubeYAML: "apiVersion: v1\nkind: Pod\n"}
	gui := newKubeGui(client, dir)

	yaml, path, exists, err := gui.generateKube(&commands.Pod{Name: "my pod"})

	assert.NoError(t, err)
	assert.Contains(t, yaml, "kind: Pod")
	assert.Equal(t, filepath.Join(dir, "my-pod.yaml"), path)
	assert.False(t, exists)

	// An existing file must flip the prompt to overwrite mode.
	assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o600))

	_, _, exists, err = gui.generateKube(&commands.Pod{Name: "my pod"})
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestKubeKeyIsBoundExactlyOnce(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	matches := []*Binding{}
	for _, binding := range gui.podBindings() {
		if binding.Key == 'g' {
			matches = append(matches, binding)
		}
	}

	assert.Len(t, matches, 1, "'g' must be registered exactly once for pods")
	assert.Equal(t, "pods", matches[0].ViewName)

	for _, binding := range gui.staticKeybindings() {
		assert.NotEqual(t, 'g', binding.Key, "'g' must not also be global")
	}
}

func TestKubeOffMessages(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	gui.PodmanCommand.SetCapabilities(commands.Capabilities{
		CLI:  commands.Capability{State: commands.CapabilityAvailable},
		Pods: commands.Capability{State: commands.CapabilityAvailable},
		KubeGenerate: commands.Capability{
			State:  commands.CapabilityUnavailable,
			Reason: "no kube subcommand",
		},
		KubePlay: commands.Capability{
			State:  commands.CapabilityUnavailable,
			Reason: "probe timed out",
		},
	})

	messages := gui.capabilityOffMessages()

	assert.Contains(t, messages, "kube generate off: no kube subcommand")
	assert.Contains(t, messages, "kube play off: probe timed out")
}
