package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docker/go-connections/nat"
)

// BuildQuadletContainer renders a Podman Quadlet `.container` unit from a
// container's inspect data. It is deliberately a small, readable file rather than
// a full mapping of every option: what is emitted is documented and stable.
func BuildQuadletContainer(ctr *Container) string {
	if ctr == nil || ctr.Details.ContainerJSONBase == nil || ctr.Details.Config == nil {
		return ""
	}

	config := ctr.Details.Config
	var builder strings.Builder

	builder.WriteString("[Unit]\nDescription=lazypodman export of " + ctr.Name + "\n\n")
	builder.WriteString("[Container]\n")

	image := config.Image
	if image == "" {
		image = ctr.Container.Image
	}
	builder.WriteString("Image=" + image + "\n")
	builder.WriteString("ContainerName=" + ctr.Name + "\n")

	if len(config.Env) > 0 {
		env := append([]string{}, config.Env...)
		sort.Strings(env)
		builder.WriteString("Environment=" + strings.Join(env, " ") + "\n")
	}

	if len(config.Cmd) > 0 {
		builder.WriteString("Exec=" + strings.Join(config.Cmd, " ") + "\n")
	}

	if ctr.Details.HostConfig != nil {
		builder.WriteString(quadletPorts(ctr.Details.HostConfig.PortBindings))
	}

	for _, mount := range ctr.Details.Mounts {
		if mount.Source != "" && mount.Destination != "" {
			builder.WriteString("Volume=" + mount.Source + ":" + mount.Destination + "\n")
		}
	}

	if ctr.Details.HostConfig != nil && ctr.Details.HostConfig.NetworkMode != "" {
		builder.WriteString("Network=" + string(ctr.Details.HostConfig.NetworkMode) + "\n")
	}

	builder.WriteString("\n[Service]\nRestart=always\n\n[Install]\nWantedBy=default.target\n")

	return builder.String()
}

func quadletPorts(bindings nat.PortMap) string {
	if len(bindings) == 0 {
		return ""
	}

	ports := make([]string, 0, len(bindings))

	for containerPort, hosts := range bindings {
		for _, host := range hosts {
			ports = append(ports, fmt.Sprintf("PublishPort=%s:%s", host.HostPort, containerPort.Port()))
		}
	}

	sort.Strings(ports)

	return strings.Join(ports, "\n") + "\n"
}
