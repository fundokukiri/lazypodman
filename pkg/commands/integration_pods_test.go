//go:build integration

package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// These tests talk to a real, running Podman. They are opt-in:
//
//	GOFLAGS=-mod=vendor go test -tags integration ./pkg/commands/ -run Integration
//
// They skip themselves when the CLI or a compatible store is missing, which is
// the case on machines without Podman.
func integrationClient(t *testing.T) *cliPodmanClient {
	t.Helper()

	socketPath := determinePodmanHost()
	client := newCLIPodmanClient(context.Background(), "podman", socketPath, PodsAuto)
	if !client.Enabled() {
		t.Skipf("pods unavailable on this host: %s", client.Reason())
	}

	return client
}

func TestIntegrationPodListMatchesContainerMap(t *testing.T) {
	client := integrationClient(t)

	pods, err := client.PodList(context.Background())
	if err != nil {
		t.Fatalf("pod list: %v", err)
	}

	podByContainer, err := client.ContainerPodMap(context.Background())
	if err != nil {
		t.Fatalf("container pod map: %v", err)
	}

	assert.Len(t, pods, len(pods), "every reported pod has a name")
	podNames := map[string]struct{}{}
	for _, pod := range pods {
		assert.NotEmpty(t, pod.Name)
		podNames[pod.Name] = struct{}{}
	}

	// Every mapped container must belong to a listed pod. The reverse does not
	// hold: a pod without containers (just its infra container) maps nothing.
	for containerID, podName := range podByContainer {
		assert.NotEmpty(t, containerID)
		_, ok := podNames[podName]
		assert.True(t, ok, "container %s maps to pod %s, which is not listed", containerID, podName)
	}
}

func TestIntegrationPodInspectAndStats(t *testing.T) {
	client := integrationClient(t)

	pods, err := client.PodList(context.Background())
	if err != nil || len(pods) == 0 {
		t.Skip("no pods on this host")
	}

	output, err := client.PodInspect(context.Background(), pods[0].Name)
	if err != nil {
		t.Fatalf("pod inspect: %v", err)
	}

	assert.NotEmpty(t, output, "pod inspect must return the JSON document")
}
