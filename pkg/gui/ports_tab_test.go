package gui

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func portContainer(name string, ports ...container.Port) *commands.Container {
	ctr := &commands.Container{Name: name, ID: name}
	ctr.Container.Ports = ports

	return ctr
}

func TestCollectPublishedPortsMarksHostPortConflicts(t *testing.T) {
	first := portContainer("web", container.Port{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"})
	second := portContainer("api", container.Port{IP: "0.0.0.0", PrivatePort: 3000, PublicPort: 8080, Type: "tcp"})
	unique := portContainer("db", container.Port{IP: "127.0.0.1", PrivatePort: 5432, PublicPort: 5432, Type: "tcp"})

	ports := collectPublishedPorts([]*commands.Container{first, second, unique})

	assert.Len(t, ports, 3)
	conflicts := 0
	for _, port := range ports {
		if port.Conflict {
			conflicts++
		}
	}
	assert.Equal(t, 2, conflicts)
}

func TestCollectPublishedPortsTreatsWildcardAsConflict(t *testing.T) {
	all := portContainer("web", container.Port{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"})
	loopback := portContainer("api", container.Port{IP: "127.0.0.1", PrivatePort: 3000, PublicPort: 8080, Type: "tcp"})

	ports := collectPublishedPorts([]*commands.Container{all, loopback})

	assert.Len(t, ports, 2)
	for _, port := range ports {
		assert.True(t, port.Conflict)
	}
}

func TestRenderPublishedPortsNamesTheConflicts(t *testing.T) {
	ports := collectPublishedPorts([]*commands.Container{
		portContainer("web", container.Port{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"}),
		portContainer("api", container.Port{IP: "0.0.0.0", PrivatePort: 3000, PublicPort: 8080, Type: "tcp"}),
	})

	rendered := utils.Decolorise(renderPublishedPorts(translations(), ports, 2))

	assert.Contains(t, rendered, "host-port conflict")
	assert.Contains(t, rendered, "web (conflict)")
	assert.Contains(t, rendered, "api (conflict)")
}

func TestRenderPublishedPortsSaysWhenThereAreNone(t *testing.T) {
	assert.Equal(t, translations().PortNonePublished, renderPublishedPorts(translations(), nil, 0))
}
