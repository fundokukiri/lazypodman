package commands

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"time"
)

// kubeProbeBudget bounds one subcommand probe. It is a variable so tests can
// shrink it.
var kubeProbeBudget = 5 * time.Second

// kubeGenerateVariants and kubePlayVariants hold the syntax the installed Podman
// might accept, newest first. The probe stores whichever one works so later calls
// never guess and no version number is hard-coded.
var (
	kubeGenerateVariants = [][]string{{"kube", "generate"}, {"generate", "kube"}}
	kubePlayVariants     = [][]string{{"kube", "play"}, {"play", "kube"}}
)

// ProbeSubcommand finds the first argv variant this Podman accepts. The answer is
// a capability, never an error: a timeout is transient, a missing subcommand is
// permanent for the session.
func (c *cliPodmanClient) ProbeSubcommand(ctx context.Context, variants [][]string) ([]string, Capability) {
	ctx, cancel := context.WithTimeout(ctx, kubeProbeBudget)
	defer cancel()

	var lastErr error

	for _, variant := range variants {
		argv := append(append([]string{}, variant...), "--help")

		if _, err := c.run(ctx, argv...); err == nil {
			return variant, Capability{State: CapabilityAvailable}
		} else {
			lastErr = err
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, Capability{
				State:     CapabilityUnavailable,
				Reason:    "probe timed out",
				Detail:    lastErr.Error(),
				Transient: true,
			}
		}
	}

	detail := ""
	if lastErr != nil {
		detail = lastErr.Error()
	}

	return nil, Capability{
		State:  CapabilityUnavailable,
		Reason: "no kube subcommand",
		Detail: detail,
	}
}

// KubeGenerate runs the generate command for one pod and returns its YAML.
func (c *cliPodmanClient) KubeGenerate(ctx context.Context, prefix []string, ref string) (string, error) {
	if err := validatePodRef(ref); err != nil {
		return "", err
	}

	argv := append(append([]string{}, prefix...), "--", ref)

	out, err := c.run(ctx, argv...)
	if err != nil {
		return string(out), err
	}

	return string(out), nil
}

// StreamKube runs play/down and hands every line of merged stdout/stderr to
// onLine as it arrives. The child is bound to ctx: cancelling the task kills it
// and the call returns promptly.
func (c *cliPodmanClient) StreamKube(ctx context.Context, prefix, args []string, onLine func(string)) error {
	return c.StreamPodman(ctx, append(append([]string{}, prefix...), args...), onLine)
}

// StreamPodman runs any podman command and streams its merged output line by line.
func (c *cliPodmanClient) StreamPodman(ctx context.Context, argv []string, onLine func(string)) error {
	command := exec.CommandContext(ctx, c.binary, argv...)
	configureProcessGroup(command)
	command.WaitDelay = 2 * time.Second

	reader, writer := io.Pipe()
	command.Stdout = writer
	command.Stderr = writer

	scanDone := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(reader)
		for {
			line, err := reader.ReadString('\n')
			if line != "" {
				onLine(strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"))
			}
			if err == io.EOF {
				scanDone <- nil
				return
			}
			if err != nil {
				scanDone <- err
				return
			}
		}
	}()

	runErr := command.Run()
	// Closing the writer lets the scanner finish even when the process wrote
	// nothing (or failed to start).
	_ = writer.Close()

	scanErr := <-scanDone

	if runErr == nil {
		runErr = scanErr
	}

	if runErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		return runErr
	}

	return nil
}

// kubeCapability is the shared lazy probe behind both kube features. It caches the
// capability with the working argv prefix so the syntax is discovered once.
func (c *PodmanCommand) kubeCapability(
	ctx context.Context,
	feature CapabilityFeature,
	variants [][]string,
) (Capability, []string) {
	c.KubeMutex.Lock()
	defer c.KubeMutex.Unlock()

	capability, prefix := c.kubeFeatureLocked(feature)

	if capability.State == CapabilityAvailable {
		return capability, prefix
	}

	if capability.State == CapabilityUnavailable && !capability.Transient {
		return capability, nil
	}

	if c.Capabilities().CLI.State == CapabilityUnavailable {
		return capability, nil
	}

	client := c.Pods()
	if client == nil {
		return Capability{State: CapabilityUnavailable, Reason: "podman CLI not found"}, nil
	}

	probedPrefix, probedCapability := client.ProbeSubcommand(ctx, variants)
	c.setCapabilityUnlocked(feature, probedCapability)

	if probedCapability.State == CapabilityAvailable {
		if c.kubePrefixes == nil {
			c.kubePrefixes = map[CapabilityFeature][]string{}
		}
		c.kubePrefixes[feature] = probedPrefix
	}

	return probedCapability, probedPrefix
}

func (c *PodmanCommand) kubeFeatureLocked(feature CapabilityFeature) (Capability, []string) {
	switch feature {
	case FeatureKubeGenerate:
		return c.capabilities.KubeGenerate, c.kubePrefixes[FeatureKubeGenerate]
	case FeatureKubePlay:
		return c.capabilities.KubePlay, c.kubePrefixes[FeatureKubePlay]
	default:
		return Capability{State: CapabilityUnavailable, Reason: "unknown feature"}, nil
	}
}

// KubeGenerateCapability reports whether `podman kube generate` works and which
// argv prefix to use.
func (c *PodmanCommand) KubeGenerateCapability(ctx context.Context) (Capability, []string) {
	return c.kubeCapability(ctx, FeatureKubeGenerate, kubeGenerateVariants)
}

// KubePlayCapability reports whether `podman kube play` works and which argv
// prefix to use. Down uses the same prefix plus --down.
func (c *PodmanCommand) KubePlayCapability(ctx context.Context) (Capability, []string) {
	return c.kubeCapability(ctx, FeatureKubePlay, kubePlayVariants)
}

// kubeIsDownArgv keeps the down flag spelling in one place.
func kubeIsDownArgv(prefix []string, path string) []string {
	return append(append([]string{}, prefix...), "--down", path)
}

// kubePlayArgv is separated so tests can assert the exact arguments.
func kubePlayArgv(prefix []string, path string) []string {
	return append(append([]string{}, prefix...), path)
}

// KubePlayArgs is the argument suffix for `kube play <file>`.
func KubePlayArgs(path string) []string {
	return []string{path}
}

// KubeDownArgs is the argument suffix for `kube play --down <file>`.
func KubeDownArgs(path string) []string {
	return []string{"--down", path}
}
