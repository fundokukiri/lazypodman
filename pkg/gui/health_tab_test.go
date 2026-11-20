package gui

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
)

// fakeHealthPodClient counts calls so the tests can prove that a blocked action
// spawns nothing.
type fakeHealthPodClient struct {
	fakeEnabledPodClient

	probeCalls  int
	healthCalls int

	probeCapability commands.Capability
	output          string
	exitCode        int
	runErr          error

	probeSubcommandCalls int
	kubeGenerateCalls    int
	streamKubeCalls      int
	kubeYAML             string
	kubeStreamLines      []string
}

func (f *fakeHealthPodClient) ProbeHealthcheck(context.Context) commands.Capability {
	f.probeCalls++

	return f.probeCapability
}

func (f *fakeHealthPodClient) HealthcheckRun(context.Context, string) (string, int, error) {
	f.healthCalls++

	return f.output, f.exitCode, f.runErr
}

func newHealthGui(client commands.PodmanClient) *Gui {
	podmanCommand := &commands.PodmanCommand{}
	podmanCommand.SetPodClient(client)
	podmanCommand.SetCapabilities(commands.Capabilities{
		CLI:       commands.Capability{State: commands.CapabilityAvailable},
		Pods:      commands.Capability{State: commands.CapabilityAvailable},
		HealthRun: commands.Capability{State: commands.CapabilityUnknown},
	})

	return &Gui{PodmanCommand: podmanCommand, Tr: translations()}
}

func translations() *i18n.TranslationSet {
	set := i18n.GetTranslationSets()["en"]

	return &set
}

func runningContainer(health *container.Health) *commands.Container {
	ctr := &commands.Container{ID: "abc"}
	ctr.Container.State = "running"
	ctr.Details.ContainerJSONBase = &container.ContainerJSONBase{}
	ctr.Details.State = &container.State{Running: true, Health: health}

	return ctr
}

func healthy() *container.Health {
	return &container.Health{Status: container.Healthy, FailingStreak: 0}
}

func TestClassifyHealthcheck(t *testing.T) {
	tr := translations()

	t.Run("zero is healthy", func(t *testing.T) {
		outcome := classifyHealthcheck(tr, "ok\n", 0, nil)

		assert.NoError(t, outcome.err)
		assert.Equal(t, "healthcheck: healthy", outcome.message)
	})

	for _, code := range []int{1, 2} {
		t.Run("documented unhealthy code", func(t *testing.T) {
			outcome := classifyHealthcheck(tr, "probe failed\n", code, nil)

			assert.NoError(t, outcome.err, "unhealthy is a result, not an error")
			assert.Contains(t, outcome.message, "unhealthy")
		})
	}

	t.Run("unknown code is an error with verbatim output", func(t *testing.T) {
		outcome := classifyHealthcheck(tr, "boom: cannot exec\n", 125, nil)

		assert.Error(t, outcome.err)
		assert.Equal(t, "boom: cannot exec", outcome.err.Error())
		assert.Empty(t, outcome.message)
	})

	t.Run("spawn failure is an error", func(t *testing.T) {
		outcome := classifyHealthcheck(tr, "", 0, errors.New("exec: podman: not found"))

		assert.Error(t, outcome.err)
		assert.Contains(t, outcome.err.Error(), "not found")
	})
}

func TestHealthcheckBlocked(t *testing.T) {
	t.Run("no healthcheck", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})
		ctr := runningContainer(&container.Health{Status: container.NoHealthcheck})

		assert.Equal(t, "This container has no healthcheck", gui.healthcheckBlockedReason(ctr))
	})

	t.Run("nil health", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})

		assert.Equal(t, "This container has no healthcheck", gui.healthcheckBlockedReason(runningContainer(nil)))
	})

	t.Run("stopped container", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})
		ctr := runningContainer(healthy())
		ctr.Container.State = "exited"
		ctr.Details.State.Running = false

		assert.Equal(t, "Container is not running", gui.healthcheckBlockedReason(ctr))
	})

	t.Run("list says running but details lag behind", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})
		ctr := runningContainer(healthy())
		ctr.Details.State.Running = false

		assert.Equal(t, "Container is not running", gui.healthcheckBlockedReason(ctr))
	})

	t.Run("cli unavailable", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})
		gui.PodmanCommand.SetCapabilities(commands.Capabilities{
			CLI: commands.Capability{State: commands.CapabilityUnavailable, Reason: "podman CLI not found"},
		})

		assert.Equal(t, "cli off: podman CLI not found", gui.healthcheckBlockedReason(runningContainer(healthy())))
	})

	t.Run("runnable", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})

		assert.Empty(t, gui.healthcheckBlockedReason(runningContainer(healthy())))
	})
}

func TestRunHealthcheck(t *testing.T) {
	t.Run("probe failure is reported, not run", func(t *testing.T) {
		client := &fakeHealthPodClient{
			probeCapability: commands.Capability{State: commands.CapabilityUnavailable, Reason: "no healthcheck subcommand"},
		}
		gui := newHealthGui(client)

		outcome := gui.runHealthcheck(runningContainer(healthy()))

		assert.NoError(t, outcome.err)
		assert.Equal(t, "health run off: no healthcheck subcommand", outcome.message)
		assert.Equal(t, 1, client.probeCalls)
		assert.Equal(t, 0, client.healthCalls)
	})

	t.Run("healthy run", func(t *testing.T) {
		client := &fakeHealthPodClient{
			probeCapability: commands.Capability{State: commands.CapabilityAvailable},
			output:          "ok\n",
		}
		gui := newHealthGui(client)

		outcome := gui.runHealthcheck(runningContainer(healthy()))

		assert.NoError(t, outcome.err)
		assert.Equal(t, "healthcheck: healthy", outcome.message)
		assert.Equal(t, 1, client.healthCalls)
	})

	t.Run("unhealthy run is a result", func(t *testing.T) {
		client := &fakeHealthPodClient{
			probeCapability: commands.Capability{State: commands.CapabilityAvailable},
			output:          "probe failed\n",
			exitCode:        1,
		}
		gui := newHealthGui(client)

		outcome := gui.runHealthcheck(runningContainer(healthy()))

		assert.NoError(t, outcome.err)
		assert.Contains(t, outcome.message, "unhealthy")
	})

	t.Run("unexpected exit code becomes an error", func(t *testing.T) {
		client := &fakeHealthPodClient{
			probeCapability: commands.Capability{State: commands.CapabilityAvailable},
			output:          "cannot exec: no such file\n",
			exitCode:        125,
		}
		gui := newHealthGui(client)

		outcome := gui.runHealthcheck(runningContainer(healthy()))

		assert.Error(t, outcome.err)
		assert.Equal(t, "cannot exec: no such file", outcome.err.Error())
	})
}

func TestHealthTabContents(t *testing.T) {
	t.Run("no healthcheck", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})

		assert.Equal(t, "This container has no healthcheck", gui.healthTabContents(runningContainer(nil)))
	})

	t.Run("status, streak and probes", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})
		health := &container.Health{
			Status:        container.Unhealthy,
			FailingStreak: 3,
			Log: []*container.HealthcheckResult{
				{ExitCode: 0, Output: "ok"},
				{ExitCode: 1, Output: "still failing"},
			},
		}

		contents := gui.healthTabContents(runningContainer(health))

		assert.Contains(t, contents, "Status: unhealthy")
		assert.Contains(t, contents, "Failing streak: 3")
		assert.Contains(t, contents, "still failing")
		assert.Contains(t, contents, "Output")
	})

	t.Run("empty log keeps the table and says there is no output", func(t *testing.T) {
		gui := newHealthGui(&fakeHealthPodClient{})
		health := &container.Health{Status: container.Healthy}

		contents := gui.healthTabContents(runningContainer(health))

		assert.Contains(t, contents, "Time")
		assert.NotContains(t, contents, "(no output)")
	})
}

// healthTabRendersFromMemory proves the tab reading a container's details needs
// no client call at all.
func TestHealthTabRendersFromMemory(t *testing.T) {
	client := &fakeHealthPodClient{}
	gui := newHealthGui(client)

	_ = gui.healthTabContents(runningContainer(healthy()))

	assert.Equal(t, 0, client.healthCalls)
	assert.Equal(t, 0, client.probeCalls)
}

func (f *fakeHealthPodClient) ProbeSubcommand(context.Context, [][]string) ([]string, commands.Capability) {
	f.probeSubcommandCalls++

	return []string{"kube", "play"}, commands.Capability{State: commands.CapabilityAvailable}
}

func (f *fakeHealthPodClient) KubeGenerate(context.Context, []string, string) (string, error) {
	f.kubeGenerateCalls++

	return f.kubeYAML, nil
}

func (f *fakeHealthPodClient) StreamKube(_ context.Context, _, _ []string, onLine func(string)) error {
	f.streamKubeCalls++

	for _, line := range f.kubeStreamLines {
		onLine(line)
	}

	return nil
}

func (f *fakeHealthPodClient) OpenAttachTerminal(_ context.Context, _ string, _, _ int) (*commands.Terminal, error) {
	return nil, nil
}

func (f *fakeHealthPodClient) OpenTerminal(_ context.Context, _ string, _, _ int) (*commands.Terminal, error) {
	return nil, nil
}

func (f *fakeHealthPodClient) StreamPodman(_ context.Context, argv []string, onLine func(string)) error {
	f.streamKubeCalls++

	for _, line := range f.kubeStreamLines {
		onLine(line)
	}

	return nil
}
