package commands

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// writeKubeScript writes a POSIX podman stub with an arbitrary body.
func writeKubeScript(t *testing.T, body string) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("fake podman script requires a POSIX shell")
	}

	path := filepath.Join(t.TempDir(), "podman")
	requireNoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755))

	return path
}

func TestProbeSubcommandFallsBackToOldSyntax(t *testing.T) {
	binary := writeKubeScript(t, `
case "$1 $2" in
  "kube generate") exit 2 ;;
  "generate kube") echo "old syntax"; exit 0 ;;
esac
exit 2
`)

	prefix, capability := (&cliPodmanClient{binary: binary}).ProbeSubcommand(
		context.Background(), kubeGenerateVariants)

	assert.Equal(t, CapabilityAvailable, capability.State)
	assert.Equal(t, []string{"generate", "kube"}, prefix)
}

func TestProbeSubcommandMissingIsPermanent(t *testing.T) {
	binary := writeKubeScript(t, "exit 2\n")

	prefix, capability := (&cliPodmanClient{binary: binary}).ProbeSubcommand(
		context.Background(), kubePlayVariants)

	assert.Nil(t, prefix)
	assert.Equal(t, CapabilityUnavailable, capability.State)
	assert.Equal(t, "no kube subcommand", capability.Reason)
	assert.False(t, capability.Transient)
}

func TestProbeSubcommandTimeoutIsTransient(t *testing.T) {
	oldBudget := kubeProbeBudget
	kubeProbeBudget = 50 * time.Millisecond
	t.Cleanup(func() { kubeProbeBudget = oldBudget })

	binary := writeKubeScript(t, "sleep 5\n")

	_, capability := (&cliPodmanClient{binary: binary}).ProbeSubcommand(
		context.Background(), kubePlayVariants)

	assert.Equal(t, CapabilityUnavailable, capability.State)
	assert.True(t, capability.Transient)
}

func TestKubeGenerateReturnsYAML(t *testing.T) {
	binary := writeKubeScript(t, "printf 'apiVersion: v1\\nkind: Pod\\n'\n")

	yaml, err := (&cliPodmanClient{binary: binary}).KubeGenerate(
		context.Background(), []string{"kube", "generate"}, "web")

	requireNoError(t, err)
	assert.Contains(t, yaml, "kind: Pod")
}

func TestKubeGenerateSurfacesStderr(t *testing.T) {
	binary := writeKubeScript(t, "echo 'no such pod: web' >&2\nexit 125\n")

	_, err := (&cliPodmanClient{binary: binary}).KubeGenerate(
		context.Background(), []string{"kube", "generate"}, "web")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such pod: web")
}

func TestKubeGenerateRejectsFlagLikeRef(t *testing.T) {
	client := &cliPodmanClient{binary: "podman"}

	_, err := client.KubeGenerate(context.Background(), []string{"kube", "generate"}, "--all")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid pod reference")
}

func TestStreamKubeDeliversLinesInOrder(t *testing.T) {
	binary := writeKubeScript(t, "echo first\necho second >&2\necho third\n")

	var mu sync.Mutex
	lines := []string{}

	err := (&cliPodmanClient{binary: binary}).StreamKube(
		context.Background(),
		[]string{"kube", "play"},
		[]string{"/tmp/x.yaml"},
		func(line string) {
			mu.Lock()
			defer mu.Unlock()
			lines = append(lines, line)
		},
	)

	requireNoError(t, err)
	assert.Equal(t, []string{"first", "second", "third"}, lines)
}

func TestStreamKubeCancelKillsChild(t *testing.T) {
	binary := writeKubeScript(t, "echo started\nsleep 5\n")

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{}, 1)
	done := make(chan error, 1)

	go func() {
		done <- (&cliPodmanClient{binary: binary}).StreamKube(
			ctx,
			[]string{"kube", "play"},
			[]string{"/tmp/x.yaml"},
			func(line string) {
				if strings.Contains(line, "started") {
					select {
					case started <- struct{}{}:
					default:
					}
				}
			},
		)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("the child never started")
	}

	cancel()

	select {
	case err := <-done:
		assert.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("cancelling did not stop the child promptly")
	}
}

func TestKubeCapabilityCachesPrefix(t *testing.T) {
	binary := writeKubeScript(t, `
case "$1 $2" in
  "kube play") echo "--help"; exit 0 ;;
esac
exit 2
`)

	client := &cliPodmanClient{binary: binary}
	podmanCommand := &PodmanCommand{}
	podmanCommand.SetPodClient(client)
	podmanCommand.SetCapabilities(deriveCapabilities(CLICheck{Enabled: true}, PodsAuto))

	capability, prefix := podmanCommand.KubePlayCapability(context.Background())
	assert.Equal(t, CapabilityAvailable, capability.State)
	assert.Equal(t, []string{"kube", "play"}, prefix)

	// A second press uses the cache: no probe, same prefix.
	capability, prefix = podmanCommand.KubePlayCapability(context.Background())
	assert.Equal(t, CapabilityAvailable, capability.State)
	assert.Equal(t, []string{"kube", "play"}, prefix)
	assert.Equal(t, []string{"kube", "play", "--down", "/tmp/x.yaml"}, kubeIsDownArgv(prefix, "/tmp/x.yaml"))
	assert.Equal(t, []string{"kube", "play", "/tmp/x.yaml"}, kubePlayArgv(prefix, "/tmp/x.yaml"))
}

func TestKubeCapabilityStopsWhenCLIUnavailable(t *testing.T) {
	client := &fakePodClient{enabled: false}
	podmanCommand := &PodmanCommand{}
	podmanCommand.SetPodClient(client)
	podmanCommand.SetCapabilities(deriveCapabilities(CLICheck{Reason: "podman CLI not found"}, PodsAuto))

	capability, prefix := podmanCommand.KubeGenerateCapability(context.Background())

	assert.Equal(t, CapabilityUnavailable, capability.State)
	assert.Nil(t, prefix)
	assert.Equal(t, 0, client.probeSubcommandCalls)
}

func TestStreamPodmanHandlesLongLines(t *testing.T) {
	binary, _, stdoutFile, _ := writeFakePodmanScript(t)
	line := strings.Repeat("x", 2<<20)
	requireNoError(t, os.WriteFile(stdoutFile, []byte(line+"\n"), 0o644))

	client := &cliPodmanClient{binary: binary}
	var got []string

	err := client.StreamPodman(context.Background(), []string{"events"}, func(line string) {
		got = append(got, line)
	})
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Len(t, got[0], 2<<20)
}
