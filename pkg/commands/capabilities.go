package commands

import "context"

// CapabilityState is the tri-state answer to "can this Podman-native feature be
// used here". Unknown is a real state: lazy features are not probed at startup so
// that opening lazypodman never pays for a feature the session may not use.
type CapabilityState int

const (
	// CapabilityUnknown means no probe has run yet. Nothing may be claimed about it.
	CapabilityUnknown CapabilityState = iota
	// CapabilityAvailable means the probe (or the CLI check) succeeded.
	CapabilityAvailable
	// CapabilityUnavailable means the probe failed; Reason says why.
	CapabilityUnavailable
)

// Capability is one feature's availability plus the reasons behind it.
type Capability struct {
	State CapabilityState
	// Reason is the short code shown in the "off: <reason>" line.
	Reason string
	// Detail carries podman's own message for the debug log, never for the UI.
	Detail string
	// Transient is true when retrying could succeed (timeout, busy store).
	// A permanent failure (missing subcommand) is never probed again.
	Transient bool
}

// Capabilities is the set of Podman-native features lazypodman may offer.
type Capabilities struct {
	CLI          Capability
	Pods         Capability
	HealthRun    Capability
	KubeGenerate Capability
	KubePlay     Capability
}

// CapabilityFeature names the lazily-probed capabilities.
type CapabilityFeature int

const (
	FeatureHealthRun CapabilityFeature = iota
	FeatureKubeGenerate
	FeatureKubePlay
)

// deriveCapabilities turns the startup CLI check into per-feature capabilities.
// Only CLI and Pods are known at this point: the rest stay Unknown until the user
// asks for them, which is what keeps startup at two podman processes.
func deriveCapabilities(check CLICheck, mode PodsMode) Capabilities {
	caps := Capabilities{
		CLI:          Capability{State: CapabilityUnknown},
		Pods:         Capability{State: CapabilityUnknown},
		HealthRun:    Capability{State: CapabilityUnknown},
		KubeGenerate: Capability{State: CapabilityUnknown},
		KubePlay:     Capability{State: CapabilityUnknown},
	}

	// "never" skips the CLI probe entirely, so nothing is known about the CLI.
	if mode == PodsNever {
		caps.Pods = Capability{State: CapabilityUnavailable, Reason: "disabled in config"}

		return caps
	}

	if check.Enabled {
		caps.CLI = Capability{State: CapabilityAvailable}
		caps.Pods = Capability{State: CapabilityAvailable}

		return caps
	}

	unavailable := Capability{State: CapabilityUnavailable, Reason: check.Reason, Detail: check.Detail}
	caps.CLI = unavailable
	caps.Pods = unavailable
	// The lazy features depend on the same CLI: no point probing them.
	caps.HealthRun = unavailable
	caps.KubeGenerate = unavailable
	caps.KubePlay = unavailable

	return caps
}

// HealthcheckCapability returns the healthcheck capability, probing the CLI on
// first use. Concurrent callers share one probe: the mutex is held across it, and
// the result is cached. A transient failure (timeout) is retried on the next call;
// a permanent one never is.
func (c *PodmanCommand) HealthcheckCapability(ctx context.Context) Capability {
	c.HealthcheckProbeMutex.Lock()
	defer c.HealthcheckProbeMutex.Unlock()

	capability := c.Capabilities().HealthRun
	if capability.State == CapabilityAvailable {
		return capability
	}
	if capability.State == CapabilityUnavailable && !capability.Transient {
		return capability
	}

	// When the CLI itself is known to be unusable, probing a subcommand would be
	// pointless: report the CLI reason the capability already inherited.
	if c.Capabilities().CLI.State == CapabilityUnavailable {
		return capability
	}

	client := c.Pods()

	if client == nil {
		return Capability{State: CapabilityUnavailable, Reason: "podman CLI not found"}
	}

	capability = client.ProbeHealthcheck(ctx)
	c.SetCapability(FeatureHealthRun, capability)

	return capability
}
