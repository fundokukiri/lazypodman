package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapabilitiesDerivation(t *testing.T) {
	enabled := CLICheck{Enabled: true, Version: "5.2.2"}
	disabled := CLICheck{Reason: "podman CLI not found"}

	t.Run("enabled leaves lazy features unknown", func(t *testing.T) {
		caps := deriveCapabilities(enabled, PodsAuto)

		assert.Equal(t, CapabilityAvailable, caps.CLI.State)
		assert.Equal(t, CapabilityAvailable, caps.Pods.State)
		assert.Equal(t, CapabilityUnknown, caps.HealthRun.State)
		assert.Equal(t, CapabilityUnknown, caps.KubeGenerate.State)
		assert.Equal(t, CapabilityUnknown, caps.KubePlay.State)
	})

	t.Run("cli failure is inherited by every feature", func(t *testing.T) {
		caps := deriveCapabilities(disabled, PodsAuto)

		for name, cap := range map[string]Capability{
			"CLI": caps.CLI, "Pods": caps.Pods, "HealthRun": caps.HealthRun,
			"KubeGenerate": caps.KubeGenerate, "KubePlay": caps.KubePlay,
		} {
			assert.Equal(t, CapabilityUnavailable, cap.State, name)
			assert.Equal(t, "podman CLI not found", cap.Reason, name)
		}
	})

	t.Run("never skips the CLI probe and only disables pods", func(t *testing.T) {
		caps := deriveCapabilities(CLICheck{Reason: "disabled in config"}, PodsNever)

		assert.Equal(t, CapabilityUnknown, caps.CLI.State)
		assert.Equal(t, CapabilityUnavailable, caps.Pods.State)
		assert.Equal(t, "disabled in config", caps.Pods.Reason)
		assert.Equal(t, CapabilityUnknown, caps.HealthRun.State)
	})
}

func TestPodmanCommandCapabilities(t *testing.T) {
	podmanCommand := &PodmanCommand{}
	podmanCommand.SetCapabilities(deriveCapabilities(CLICheck{Enabled: true}, PodsAuto))

	podmanCommand.SetCapability(FeatureHealthRun, Capability{
		State:  CapabilityUnavailable,
		Reason: "no healthcheck subcommand",
	})

	caps := podmanCommand.Capabilities()
	assert.Equal(t, CapabilityUnavailable, caps.HealthRun.State)
	assert.Equal(t, "no healthcheck subcommand", caps.HealthRun.Reason)

	// The returned value is a copy: mutating it must not leak back.
	caps.HealthRun.Reason = "mutated"
	assert.Equal(t, "no healthcheck subcommand", podmanCommand.Capabilities().HealthRun.Reason)
}
