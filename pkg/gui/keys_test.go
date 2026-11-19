package gui

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
	"github.com/jesseduffield/gocui"
	"github.com/stretchr/testify/assert"
)

// fakeEnabledPodClient satisfies the pod client interface with no behaviour: the
// key tests only care whether pods are enabled.
type fakeEnabledPodClient struct{}

func (fakeEnabledPodClient) Enabled() bool   { return true }
func (fakeEnabledPodClient) Reason() string  { return "" }
func (fakeEnabledPodClient) Version() string { return "5.2.0" }
func (fakeEnabledPodClient) PodList(context.Context) ([]*commands.Pod, error) {
	return nil, nil
}

func (fakeEnabledPodClient) ContainerPodMap(context.Context) (map[string]string, error) {
	return nil, nil
}

func (fakeEnabledPodClient) PodInspect(context.Context, string) ([]byte, error) { return nil, nil }
func (fakeEnabledPodClient) PodAction(context.Context, string, string, ...string) error {
	return nil
}
func (fakeEnabledPodClient) PodPrune(context.Context) error { return nil }
func (fakeEnabledPodClient) HealthcheckRun(context.Context, string) (string, int, error) {
	return "", 0, nil
}

func (fakeEnabledPodClient) ProbeHealthcheck(context.Context) commands.Capability {
	return commands.Capability{State: commands.CapabilityAvailable}
}

func (fakeEnabledPodClient) ProbeSubcommand(context.Context, [][]string) ([]string, commands.Capability) {
	return nil, commands.Capability{State: commands.CapabilityUnavailable, Reason: "fake"}
}

func (fakeEnabledPodClient) KubeGenerate(context.Context, []string, string) (string, error) {
	return "", nil
}

func (fakeEnabledPodClient) StreamKube(context.Context, []string, []string, func(string)) error {
	return nil
}

func (fakeEnabledPodClient) PodLogs(context.Context, *commands.Pod, []*commands.Container, int) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func newPodsGui(enabled bool) *Gui {
	podmanCommand := &commands.PodmanCommand{}
	// A nil client answers Enabled() == false, which is the pods-off state.
	if enabled {
		podmanCommand.SetPodClient(fakeEnabledPodClient{})
	}

	return &Gui{PodmanCommand: podmanCommand, Views: Views{}, Tr: &i18n.TranslationSet{}}
}

func TestSidePanelKeysWithoutPods(t *testing.T) {
	gui := newPodsGui(false)

	keys := gui.sidePanelKeys()

	assert.Len(t, keys, 6)
	assert.Equal(t, "project", keys[0].name)
	assert.Equal(t, '1', keys[0].key)
	assert.Equal(t, "services", keys[1].name)
	assert.Equal(t, '2', keys[1].key)
	assert.Equal(t, "containers", keys[2].name)
	assert.Equal(t, '3', keys[2].key)
	assert.Equal(t, "networks", keys[5].name)
	assert.Equal(t, '6', keys[5].key)
}

func TestSidePanelKeysWithPods(t *testing.T) {
	gui := newPodsGui(true)
	// PodsEnabled also requires the view, which only exists after startup.
	gui.Views.Pods = &gocui.View{}

	keys := gui.sidePanelKeys()

	assert.Len(t, keys, 7)
	assert.Equal(t, "pods", keys[2].name)
	assert.Equal(t, '3', keys[2].key)
	assert.Equal(t, "containers", keys[3].name)
	assert.Equal(t, '4', keys[3].key)
	assert.Equal(t, "networks", keys[6].name)
	assert.Equal(t, '7', keys[6].key)
	assert.Equal(t, "[4]", gui.sidePanelTitlePrefix("containers"))
	assert.Empty(t, gui.sidePanelTitlePrefix("menu"))
}

func (fakeEnabledPodClient) StreamPodman(context.Context, []string, func(string)) error {
	return nil
}

func (fakeEnabledPodClient) OpenTerminal(context.Context, string, int, int) (*commands.Terminal, error) {
	return nil, nil
}

func TestGetKeyRendersControlKeys(t *testing.T) {
	cases := map[interface{}]string{
		gocui.KeyCtrlP: "ctrl+p",
		gocui.KeyCtrlA: "ctrl+a",
		gocui.KeyCtrlZ: "ctrl+z",
		' ':            "space",
		'Q':            "Q",
	}

	for key, want := range cases {
		binding := &Binding{Key: key}
		if got := binding.GetKey(); got != want {
			t.Errorf("GetKey(%v) = %q, want %q", key, got, want)
		}
	}
}

// gocui's tcell driver turns the spacebar into KeySpace with an empty rune, so a
// binding declared as the rune ' ' never matches. This test exists because that
// bug shipped once and made multi-select look broken.
func TestSpaceBindingsUseKeySpace(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	found := 0

	for _, binding := range gui.staticKeybindings() {
		if _, isRune := binding.Key.(rune); isRune && binding.Key.(rune) == ' ' {
			t.Errorf("binding %q uses the rune ' ', which gocui never delivers for the spacebar", binding.Description)
		}

		if binding.Key == gocui.KeySpace {
			found++

			if binding.GetKey() != "space" {
				t.Errorf("KeySpace binding renders as %q, want \"space\"", binding.GetKey())
			}
		}
	}

	if found != 2 {
		t.Errorf("expected the menu and containers space bindings, found %d", found)
	}
}

func TestAttachInCurrentTerminalKeyIsBound(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	for _, view := range []string{"containers", "services"} {
		found := false
		for _, binding := range gui.staticKeybindings() {
			if binding.ViewName != view || binding.GetKey() != "A" {
				continue
			}

			found = true
			assert.NotNil(t, binding.Handler)
			assert.Equal(t, gui.Tr.AttachCurrentTerminal, binding.Description)
		}

		assert.True(t, found, "%s must bind uppercase A", view)
	}
}

// E opens the in-app terminal and stays visible in the x options menu; x itself
// must keep opening that menu (the global binding), not a panel-specific one.
func TestExecShellOpensTerminalTab(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	found := false
	for _, binding := range gui.staticKeybindings() {
		if binding.ViewName == "containers" && binding.GetKey() == "ctrl+x" {
			found = true

			if binding.Description != gui.Tr.ExecShell {
				t.Errorf("ctrl+x description = %q, want exec shell", binding.Description)
			}

			if binding.Handler == nil {
				t.Error("ctrl+x has no handler")
			}
		}

		if binding.ViewName == "containers" && (binding.GetKey() == "E" || binding.GetKey() == "x") {
			t.Errorf("containers must not bind %s", binding.GetKey())
		}
	}

	if !found {
		t.Fatal("ctrl+x exec-shell binding for the containers panel is missing")
	}
}
