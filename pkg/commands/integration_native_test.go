//go:build integration

package commands

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"

	"github.com/fundokukiri/lazypodman/pkg/config"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
	"github.com/fundokukiri/lazypodman/pkg/log"
)

// Real-Podman checks for the Podman-native features. Opt-in:
//
//	GOFLAGS=-mod=vendor go test -tags integration ./pkg/commands/ -run Integration -v
//
// They read whatever containers already exist and create one throwaway container
// for the event stream, so they never touch running workloads.
func TestIntegrationStartupCheckAndTiming(t *testing.T) {
	start := time.Now()

	client := newCLIPodmanClient(context.Background(), "podman", determinePodmanHost(), PodsAuto)
	elapsed := time.Since(start)

	if !client.Enabled() {
		t.Skipf("CLI unavailable on this host: %s", client.Reason())
	}

	t.Logf("startup CLI check: %s, version %s", elapsed, client.Version())

	if elapsed > 3*time.Second {
		t.Errorf("startup CLI check took %s, expected one podman process under 3s", elapsed)
	}
}

func TestIntegrationKubeCapabilitiesAndGenerate(t *testing.T) {
	client := integrationClient(t)

	command := &PodmanCommand{}
	command.SetPodClient(client)

	ctx := context.Background()

	generateCap, generatePrefix := command.KubeGenerateCapability(ctx)
	if generateCap.State != CapabilityAvailable {
		t.Skipf("kube generate unavailable: %s", generateCap.Reason)
	}

	t.Logf("generate prefix: %v", generatePrefix)

	playCap, playPrefix := command.KubePlayCapability(ctx)
	if playCap.State == CapabilityAvailable {
		t.Logf("play prefix: %v, down args: %v", playPrefix, KubeDownArgs("x.yaml"))

		if strings.Join(KubeDownArgs("x.yaml"), " ") != "--down x.yaml" {
			t.Errorf("unexpected down args: %v", KubeDownArgs("x.yaml"))
		}
	}

	// Use the first existing container as the generate target; skip when there is none.
	out, err := client.run(ctx, "ps", "-a", "--format", "{{.Names}}")
	if err != nil {
		t.Fatalf("ps: %v", err)
	}

	names := strings.Fields(string(out))
	if len(names) == 0 {
		t.Skip("no containers to generate kube YAML from")
	}

	yaml, err := client.KubeGenerate(ctx, generatePrefix, names[0])
	if err != nil {
		t.Fatalf("kube generate: %v", err)
	}

	if !strings.Contains(yaml, "kind: Pod") {
		t.Errorf("generated YAML has no pod kind:\n%s", yaml)
	}
}

func TestIntegrationEventsStream(t *testing.T) {
	client := integrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	lines := make(chan string, 16)

	go func() {
		_ = client.StreamPodman(ctx, []string{"events"}, func(line string) {
			select {
			case lines <- line:
			default:
			}
		})
	}()

	// Generate an event with a throwaway container; remove it even if assertions fail.
	createCtx, createCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer createCancel()

	if _, err := client.run(createCtx, "create", "--name", "lazypodman-events-probe", "docker.io/library/busybox:latest", "true"); err != nil {
		t.Skipf("cannot create a probe container: %v", err)
	}

	defer func() {
		_, _ = client.run(context.Background(), "rm", "-f", "lazypodman-events-probe")
	}()

	deadline := time.After(15 * time.Second)

	for {
		select {
		case line := <-lines:
			t.Logf("event: %s", line)

			return
		case <-deadline:
			t.Fatal("no event line arrived within 15s")
		}
	}
}

// TestIntegrationQuadletInstallAcceptsGeneratedUnit checks that the Quadlet file
// built from a real inspect is accepted by podman's own parser. It uses a temporary
// HOME so nothing is installed for the real user.
func TestIntegrationQuadletInstallAcceptsGeneratedUnit(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()

	out, err := client.run(ctx, "ps", "-a", "--format", "{{.Names}}")
	if err != nil {
		t.Fatalf("ps: %v", err)
	}

	names := strings.Fields(string(out))
	if len(names) == 0 {
		t.Skip("no containers to export")
	}

	inspectOut, err := client.run(ctx, "container", "inspect", names[0])
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}

	// `podman inspect` returns a list, like `docker inspect`.
	var inspected []container.InspectResponse
	if err := json.Unmarshal(inspectOut, &inspected); err != nil {
		t.Fatalf("inspect JSON did not match the Docker-compatible shape: %v", err)
	}

	if len(inspected) == 0 {
		t.Fatal("inspect returned no entries")
	}

	details := inspected[0]
	ctr := &Container{Name: strings.TrimPrefix(details.Name, "/")}
	ctr.Details = details

	unit := BuildQuadletContainer(ctr)
	if unit == "" {
		t.Fatal("builder returned an empty unit for real inspect data")
	}

	for _, key := range []string{"[Container]", "Image=", "ContainerName="} {
		if !strings.Contains(unit, key) {
			t.Errorf("generated unit is missing %q:\n%s", key, unit)
		}
	}

	dir := t.TempDir()
	path := filepath.Join(dir, ctr.Name+".container")

	if err := WriteManifestFile(path, []byte(unit), false); err != nil {
		t.Fatalf("write unit: %v", err)
	}

	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))

	if _, err := client.run(ctx, "quadlet", "install", "--reload-systemd=false", path); err != nil {
		// podman resolves specifiers through systemd, which this sandbox does not run.
		// The unit text is still produced and printed above; acceptance needs a
		// machine with a user systemd instance.
		if strings.Contains(err.Error(), "systemd dbus") {
			t.Skipf("podman Quadlet parser needs systemd; generated unit was not validated: %v", err)
		}

		t.Fatalf("podman rejected the generated Quadlet:\n%s\nerror: %v", unit, err)
	}
}

// TestIntegrationSocketAndPodRefresh drives the same startup objects the app uses
// against the real socket: the Docker-compatible client, the CLI client and the pod
// snapshot. It catches API-shape drift that hand-written fixtures cannot.
func TestIntegrationSocketAndPodRefresh(t *testing.T) {
	client := integrationClient(t)

	ctx := context.Background()

	// Create a stopped container through the CLI; no runtime or cgroups needed.
	_, _ = client.run(ctx, "rm", "-f", "lazypodman-socket-probe")
	if _, err := client.run(ctx, "create", "--name", "lazypodman-socket-probe", "docker.io/library/busybox:latest", "true"); err != nil {
		t.Skipf("cannot create a probe container: %v", err)
	}

	defer func() {
		_, _ = client.run(context.Background(), "rm", "-f", "lazypodman-socket-probe")
	}()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))

	appConfig, err := config.NewAppConfig("lazypodman", "", "", "", "", true, nil, "", "")
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	appConfig.UserConfig.Gui.Language = "en"

	tr, err := i18n.NewTranslationSetFromConfig(log.NewLogger(appConfig, "integration"), "en")
	if err != nil {
		t.Fatalf("translations: %v", err)
	}

	command, err := NewPodmanCommand(log.NewLogger(appConfig, "integration"), NewOSCommand(log.NewLogger(appConfig, "integration"), appConfig), tr, appConfig, make(chan error, 1))
	if err != nil {
		t.Fatalf("podman command: %v", err)
	}

	defer func() { _ = command.Close() }()

	if caps := command.Capabilities(); caps.CLI.State != CapabilityAvailable {
		t.Fatalf("CLI not available: %v", caps.CLI)
	}

	listed, err := command.Client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		t.Fatalf("socket container list: %v", err)
	}

	found := false

	for _, ctr := range listed {
		for _, name := range ctr.Names {
			if strings.TrimPrefix(name, "/") == "lazypodman-socket-probe" {
				found = true
			}
		}
	}

	if !found {
		t.Errorf("socket list did not include the created container: %+v", listed)
	}

	state, err := command.RefreshPodState(ctx)
	if err != nil {
		t.Fatalf("pod refresh: %v", err)
	}

	if state == nil {
		t.Fatal("pod refresh returned no snapshot")
	}
}
