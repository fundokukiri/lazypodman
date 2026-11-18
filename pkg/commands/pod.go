package commands

import (
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

	OSCommand     *OSCommand
	PodmanBinary  string
	PodClient     PodmanClient
	Log           *logrus.Entry
	PodmanCommand LimitedPodmanCommand
}

// HasInfra reports whether the pod has an infra container. Pods created with
// `--infra=false` have no InfraId and therefore no ports of their own.
func (p *Pod) HasInfra() bool {
	return p.InfraID != ""
}
