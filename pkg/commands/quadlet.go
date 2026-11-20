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
	if ctr == nil {
		return ""
	}

	details := ctr.DetailsSnapshot()
	if details.ContainerJSONBase == nil || details.Config == nil {
		return ""
	}

	config := details.Config
	var builder strings.Builder

	builder.WriteString("[Unit]\nDescription=lazypodman export of " + quadletValue(ctr.Name) + "\n\n")
	builder.WriteString("[Container]\n")

	image := config.Image
	if image == "" {
		image = ctr.Container.Image
	}
	builder.WriteString("Image=" + quadletValue(image) + "\n")
	builder.WriteString("ContainerName=" + quadletValue(ctr.Name) + "\n")

	if len(config.Env) > 0 {
		env := append([]string{}, config.Env...)
		sort.Strings(env)
		for _, value := range env {
			builder.WriteString("Environment=" + quadletValue(value) + "\n")
		}
	}

	if len(config.Cmd) > 0 {
		args := make([]string, 0, len(config.Cmd))
		for _, arg := range config.Cmd {
			args = append(args, quadletValue(arg))
		}
		builder.WriteString("Exec=" + strings.Join(args, " ") + "\n")
	}

	if details.HostConfig != nil {
		builder.WriteString(quadletPorts(details.HostConfig.PortBindings))
	}

	for _, mount := range details.Mounts {
		if mount.Source == "" || mount.Destination == "" {
			continue
		}

		source := mount.Source
		if mount.Type == "volume" && mount.Name != "" {
			source = mount.Name
		}
		builder.WriteString("Volume=" + quadletValue(source+":"+mount.Destination) + "\n")
	}

	if details.HostConfig != nil && details.HostConfig.NetworkMode != "" {
		builder.WriteString("Network=" + quadletValue(string(details.HostConfig.NetworkMode)) + "\n")
	}

	builder.WriteString("\n[Service]\nRestart=always\n\n[Install]\nWantedBy=default.target\n")

	return builder.String()
}

// quadletValue keeps a value on one line and quotes it for systemd's unit-file
// parser, so a newline in a container env value cannot inject a new directive.
func quadletValue(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return "\"" + value + "\""
}

func quadletPorts(bindings nat.PortMap) string {
	if len(bindings) == 0 {
		return ""
	}

	ports := make([]string, 0, len(bindings))

	for containerPort, hosts := range bindings {
		for _, host := range hosts {
			ports = append(ports, fmt.Sprintf("PublishPort=%s:%s", quadletValue(host.HostPort), containerPort.Port()))
		}
	}

	sort.Strings(ports)

	return strings.Join(ports, "\n") + "\n"
}
