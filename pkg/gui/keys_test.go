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
