package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// Pod is a podman pod: a group of containers that share namespaces. Pods have
// no Docker equivalent, which is why they are fetched with the podman CLI rather
// than through the Docker-compatible service API.
type Pod struct {
	ID                 string            `json:"Id"`
	Name               string            `json:"Name"`
	Status             string            `json:"Status"`
	Created            string            `json:"Created"`
	InfraID            string            `json:"InfraId"`
	NumberOfContainers int               `json:"NumberOfContainers"`
	Labels             map[string]string `json:"Labels"`

	// Ports is the published ports of the pod's infra container, rendered the
	// same way as the containers panel. Filled by joinPods, never by JSON, and
	// empty for pods created with --infra=false.
	Ports string `json:"-"`

	// RunningContainers counts the pod's containers that are running. Filled by
	// joinPods.
	RunningContainers int `json:"-"`

	OSCommand     *OSCommand           `json:"-"`
	PodmanBinary  string               `json:"-"`
	PodClient     PodmanClient         `json:"-"`
	Log           *logrus.Entry        `json:"-"`
	PodmanCommand LimitedPodmanCommand `json:"-"`
}

// HasInfra reports whether the pod has an infra container. Pods created with
// `--infra=false` have no InfraId and therefore no ports of their own.
func (p *Pod) HasInfra() bool {
	return p.InfraID != ""
}

// parsePods parses `podman pod ls --format json`. A host without pods prints an
// empty string or `null`; both mean "no pods", not an error. Unknown fields are
// ignored so a newer Podman stays usable. The returned pods have no runtime
// wiring yet: the caller fills OSCommand/PodClient in.
func parsePods(out []byte) ([]*Pod, error) {
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	var pods []*Pod
	if err := json.Unmarshal([]byte(trimmed), &pods); err != nil {
		return nil, fmt.Errorf("parse `pod ls --format json` output: %w", err)
	}

	return pods, nil
}

// psEntry is the part of `podman ps -a --format json` we need for the
// container-ID -> pod-name mapping. The Docker-compatible API reports PodName as
// empty, which is why this comes from the CLI.
type psEntry struct {
	ID      string `json:"Id"`
	PodName string `json:"PodName"`
}

// parseContainerPodMap parses `podman ps -a --format json` into a map keyed by
// full container ID. Containers without a pod (empty PodName) are skipped: they
// belong to no pod, which the caller represents as a missing key.
func parseContainerPodMap(out []byte) (map[string]string, error) {
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" || trimmed == "null" {
		return map[string]string{}, nil
	}

	var entries []psEntry
	if err := json.Unmarshal([]byte(trimmed), &entries); err != nil {
		return nil, fmt.Errorf("parse `ps -a --format json` output: %w", err)
	}

	podByContainer := make(map[string]string, len(entries))
	for _, entry := range entries {
		if entry.ID == "" || entry.PodName == "" {
			continue
		}

		podByContainer[entry.ID] = entry.PodName
	}

	return podByContainer, nil
}
