package commands

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
)

// The inspect data below is SYNTHETIC; no real container was captured.
func quadletFixture() *Container {
	ctr := &Container{Name: "web"}

	ctr.Details.ContainerJSONBase = &container.ContainerJSONBase{}
	ctr.Details.Config = &container.Config{
		Image: "docker.io/library/nginx:latest",
		Env:   []string{"B=2", "A=1"},
		Cmd:   []string{"nginx", "-g", "daemon off;"},
	}
	ctr.Details.HostConfig = &container.HostConfig{
		PortBindings: nat.PortMap{
			"80/tcp": []nat.PortBinding{{HostPort: "8080"}},
		},
		NetworkMode: "bridge",
	}
	ctr.Details.Mounts = []container.MountPoint{{Source: "/srv/www", Destination: "/usr/share/nginx/html"}}

	return ctr
}

func TestBuildQuadletContainer(t *testing.T) {
	unit := BuildQuadletContainer(quadletFixture())

	assert.Contains(t, unit, "[Container]")
	assert.Contains(t, unit, "Image=docker.io/library/nginx:latest")
	assert.Contains(t, unit, "ContainerName=web")
	assert.Contains(t, unit, "Environment=A=1 B=2", "env is sorted so the file is stable")
	assert.Contains(t, unit, "Exec=nginx -g daemon off;")
	assert.Contains(t, unit, "PublishPort=8080:80")
	assert.Contains(t, unit, "Volume=/srv/www:/usr/share/nginx/html")
	assert.Contains(t, unit, "Network=bridge")
	assert.Contains(t, unit, "[Service]")
	assert.Contains(t, unit, "[Install]")
}

func TestBuildQuadletContainerRequiresInspect(t *testing.T) {
	assert.Empty(t, BuildQuadletContainer(nil))
	assert.Empty(t, BuildQuadletContainer(&Container{Name: "web"}))
}
