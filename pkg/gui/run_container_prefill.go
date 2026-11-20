package gui

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// runImageInfo is the read-only part of an image the menu shows: ports and
// volume declarations are suggestions, not flags copied into the command.
type runImageInfo struct {
	Ports    []string
	Volumes  []string
	EnvCount int
	Labels   int
}

func (info runImageInfo) summary() string {
	parts := make([]string, 0, 4)
	if len(info.Ports) > 0 {
		parts = append(parts, "ports "+strings.Join(info.Ports, ", "))
	}
	if len(info.Volumes) > 0 {
		parts = append(parts, "volumes "+strconv.Itoa(len(info.Volumes)))
	}
	if info.EnvCount > 0 {
		parts = append(parts, "env "+strconv.Itoa(info.EnvCount))
	}
	if info.Labels > 0 {
		parts = append(parts, "labels "+strconv.Itoa(info.Labels))
	}

	return strings.Join(parts, " · ")
}

func (gui *Gui) imageRunState(image *commands.Image) *runContainerState {
	state := newRunContainerState(runImageRef(image))
	if gui.PodmanCommand == nil || gui.PodmanCommand.Client == nil {
		return state
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	inspect, err := gui.PodmanCommand.Client.ImageInspect(ctx, image.ID)
	if err != nil {
		if gui.Log != nil {
			gui.Log.Debugf("could not inspect image %s: %v", image.ID, err)
		}

		return state
	}

	applyImageDefaults(state, inspect)
	return state
}

func applyImageDefaults(state *runContainerState, inspect imagetypes.InspectResponse) {
	if inspect.Config == nil {
		return
	}

	applyImageConfig(state, inspect.Config)
	state.imageInfo = imageInfoFromConfig(inspect.Config)
}

func applyImageConfig(state *runContainerState, config *dockerspec.DockerOCIImageConfig) {
	if config == nil {
		return
	}

	// A single entrypoint is representable as --entrypoint. A multi-argument
	// entrypoint is not: --entrypoint takes one argv element and joining it would
	// silently change the command.
	if len(config.Entrypoint) == 1 {
		state.text["entrypoint"] = config.Entrypoint[0]
	}
	if len(config.Cmd) > 0 {
		state.text["command"] = joinRunArgs(config.Cmd)
	}
	if config.WorkingDir != "" {
		state.text["workdir"] = config.WorkingDir
	}
	if config.User != "" {
		state.text["user"] = config.User
	}
	if config.Healthcheck != nil {
		if command := healthCommand(config.Healthcheck.Test); command != "" {
			state.text["health_cmd"] = command
		}
	}
}

func imageInfoFromConfig(config *dockerspec.DockerOCIImageConfig) runImageInfo {
	info := runImageInfo{EnvCount: len(config.Env), Labels: len(config.Labels)}
	for port := range config.ExposedPorts {
		info.Ports = append(info.Ports, port)
	}
	for volume := range config.Volumes {
		info.Volumes = append(info.Volumes, volume)
	}
	sort.Strings(info.Ports)
	sort.Strings(info.Volumes)

	return info
}

func healthCommand(test []string) string {
	if len(test) == 0 || test[0] == "NONE" {
		return ""
	}
	if test[0] == "CMD-SHELL" && len(test) > 1 {
		return test[1]
	}
	if test[0] == "CMD" && len(test) > 1 {
		return joinRunArgs(test[1:])
	}

	return ""
}

func joinRunArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, quoteRunArg(arg))
	}

	return strings.Join(quoted, " ")
}

// applyRunLikeContainer copies the settings that are visible in inspect. The
// name is intentionally not copied: reusing it would collide with the source
// container unless the user asks for --replace.
func applyRunLikeContainer(state *runContainerState, ctr *commands.Container) {
	if !ctr.DetailsLoaded() || ctr.Details.Config == nil {
		return
	}

	config := ctr.Details.Config
	imageConfig := &dockerspec.DockerOCIImageConfig{ImageConfig: ocispecImageConfig(config)}
	if config.Healthcheck != nil {
		imageConfig.Healthcheck = &dockerspec.HealthcheckConfig{Test: config.Healthcheck.Test}
	}
	applyImageConfig(state, imageConfig)

	if config.Hostname != "" {
		state.text["hostname"] = config.Hostname
	}
	if config.Tty {
		state.toggles["tty"] = true
	}
	if config.OpenStdin {
		state.toggles["interactive"] = true
	}
	if ctr.Details.HostConfig != nil {
		if ctr.Details.HostConfig.AutoRemove {
			state.toggles["rm"] = true
		}
		if name := string(ctr.Details.HostConfig.RestartPolicy.Name); name != "" && name != "no" {
			state.text["restart"] = name
		}
		if network := string(ctr.Details.HostConfig.NetworkMode); network != "" && network != "default" && network != "bridge" {
			state.text["network"] = network
		}
	}

	state.lists["env"] = append([]string(nil), config.Env...)
	state.lists["labels"] = mapToRunList(config.Labels)
	state.lists["ports"] = portBindingsToRunList(ctr)
	state.lists["volumes"] = mountsToRunList(ctr)
}

func ocispecImageConfig(config *container.Config) ocispec.ImageConfig {
	imageConfig := ocispec.ImageConfig{}
	imageConfig.Env = config.Env
	imageConfig.Entrypoint = config.Entrypoint
	imageConfig.Cmd = config.Cmd
	imageConfig.WorkingDir = config.WorkingDir
	imageConfig.User = config.User
	imageConfig.Labels = config.Labels
	return imageConfig
}

func mapToRunList(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+values[key])
	}

	return out
}

func portBindingsToRunList(ctr *commands.Container) []string {
	if ctr.Details.HostConfig == nil {
		return nil
	}

	ports := make([]string, 0)
	for port, bindings := range ctr.Details.HostConfig.PortBindings {
		for _, binding := range bindings {
			if binding.HostPort == "" {
				continue
			}
			host := binding.HostPort
			if binding.HostIP != "" {
				host = binding.HostIP + ":" + host
			}
			ports = append(ports, host+":"+string(port))
		}
	}
	sort.Strings(ports)

	return ports
}

func mountsToRunList(ctr *commands.Container) []string {
	mounts := make([]string, 0, len(ctr.Details.Mounts))
	for _, mount := range ctr.Details.Mounts {
		source := mount.Source
		if mount.Type == "volume" && mount.Name != "" {
			source = mount.Name
		}
		if source == "" || mount.Destination == "" {
			continue
		}
		entry := source + ":" + mount.Destination
		if !mount.RW {
			entry += ":ro"
		}
		mounts = append(mounts, entry)
	}
	sort.Strings(mounts)

	return mounts
}

type runIssue struct {
	Message  string
	Blocking bool
}

func validateRunState(state *runContainerState, containers []*commands.Container) []runIssue {
	issues := make([]runIssue, 0)

	published := map[string]bool{}
	for _, spec := range state.lists["ports"] {
		hostPort, proto, err := parseRunHostPort(spec)
		if err != nil {
			issues = append(issues, runIssue{Message: fmt.Sprintf("port %q: %v", spec, err), Blocking: true})
			continue
		}
		if hostPort == 0 {
			continue
		}
		key := fmt.Sprintf("%d/%s", hostPort, proto)
		if published[key] {
			issues = append(issues, runIssue{Message: fmt.Sprintf("host port %d/%s is used twice in this run", hostPort, proto), Blocking: true})
		}
		published[key] = true

		for _, ctr := range containers {
			for _, binding := range ctr.Container.Ports {
				if int(binding.PublicPort) == hostPort && binding.Type == proto {
					issues = append(issues, runIssue{Message: fmt.Sprintf("host port %d/%s is already published by %s", hostPort, proto, ctr.Name), Blocking: true})
					break
				}
			}
		}
	}

	name := strings.TrimSpace(strings.TrimPrefix(state.text["name"], "/"))
	if name != "" && !state.toggles["replace"] {
		for _, ctr := range containers {
			if ctr.Name == name {
				issues = append(issues, runIssue{Message: fmt.Sprintf("a container named %s already exists; enable --replace or choose another name", name), Blocking: true})
				break
			}
		}
	}

	for _, volume := range state.lists["volumes"] {
		parts := strings.Split(volume, ":")
		if len(parts) < 2 {
			continue
		}
		source := parts[0]
		if strings.HasPrefix(source, "/") || strings.HasPrefix(source, ".") {
			if _, err := os.Stat(source); err != nil {
				issues = append(issues, runIssue{Message: fmt.Sprintf("host path %s does not exist", source), Blocking: false})
			}
		}
	}

	if state.toggles["privileged"] {
		issues = append(issues, runIssue{Message: "the container runs privileged", Blocking: false})
	}
	if state.toggles["rm"] && state.text["restart"] != "" && state.text["restart"] != "no" {
		issues = append(issues, runIssue{Message: "--rm is ignored when a restart policy is set", Blocking: false})
	}
	if user := strings.TrimSpace(state.text["user"]); user == "0" || user == "root" {
		issues = append(issues, runIssue{Message: "the container runs as root", Blocking: false})
	}

	return issues
}

func parseRunHostPort(spec string) (int, string, error) {
	proto := "tcp"
	if index := strings.LastIndexByte(spec, '/'); index >= 0 {
		proto = spec[index+1:]
		spec = spec[:index]
	}
	if proto != "tcp" && proto != "udp" && proto != "sctp" {
		return 0, proto, fmt.Errorf("unsupported protocol %q", proto)
	}

	parts := strings.Split(spec, ":")
	containerPart := parts[len(parts)-1]
	containerPort, err := strconv.Atoi(containerPart)
	if err != nil || containerPort < 1 || containerPort > 65535 {
		return 0, proto, fmt.Errorf("invalid container port %q", containerPart)
	}
	if len(parts) == 1 {
		return 0, proto, nil
	}

	hostPart := parts[len(parts)-2]
	hostPort, err := strconv.Atoi(hostPart)
	if err != nil || hostPort < 1 || hostPort > 65535 {
		return 0, proto, fmt.Errorf("invalid host port %q", hostPart)
	}

	return hostPort, proto, nil
}

func validationSummary(issues []runIssue) string {
	errors, warnings := 0, 0
	for _, issue := range issues {
		if issue.Blocking {
			errors++
		} else {
			warnings++
		}
	}
	if errors == 0 && warnings == 0 {
		return ""
	}

	parts := make([]string, 0, 2)
	if errors > 0 {
		parts = append(parts, fmt.Sprintf("%d error(s)", errors))
	}
	if warnings > 0 {
		parts = append(parts, fmt.Sprintf("%d warning(s)", warnings))
	}

	return strings.Join(parts, ", ")
}

func issueMessages(issues []runIssue) string {
	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		prefix := "warning: "
		if issue.Blocking {
			prefix = "error: "
		}
		messages = append(messages, prefix+issue.Message)
	}

	return strings.Join(messages, "\n")
}

func copyRunState(state *runContainerState) *runContainerState {
	if state == nil {
		return nil
	}

	copyState := &runContainerState{
		image:     state.image,
		extraArgs: state.extraArgs,
		imageInfo: state.imageInfo,
		text:      map[string]string{},
		toggles:   map[string]bool{},
		lists:     map[string][]string{},
	}
	for key, value := range state.text {
		copyState.text[key] = value
	}
	for key, value := range state.toggles {
		copyState.toggles[key] = value
	}
	for key, values := range state.lists {
		copyState.lists[key] = append([]string(nil), values...)
	}

	return copyState
}

func (gui *Gui) runCommandPreview(state *runContainerState) string {
	command := formatRunCommand(gui.Config.PodmanBinary(), state.argv(state.extraArgs))
	if len(command) > 90 {
		command = command[:89] + "…"
	}

	return command
}

func (gui *Gui) containersForRunValidation() []*commands.Container {
	if gui.Panels.Containers == nil {
		return nil
	}

	return gui.Panels.Containers.List.GetAllItems()
}

func (gui *Gui) rememberRunState(state *runContainerState) {
	if state == nil || state.image == "" {
		return
	}

	gui.lastRunsMutex.Lock()
	defer gui.lastRunsMutex.Unlock()

	if gui.lastRuns == nil {
		gui.lastRuns = map[string]*runContainerState{}
	}
	gui.lastRuns[state.image] = copyRunState(state)
}

func (gui *Gui) lastRunFor(image string) *runContainerState {
	gui.lastRunsMutex.Lock()
	defer gui.lastRunsMutex.Unlock()

	state, ok := gui.lastRuns[image]
	if !ok {
		return nil
	}

	return copyRunState(state)
}

func (gui *Gui) showRunLikeContainerMenu(state *runContainerState, selected int) error {
	containers := gui.containersForRunValidation()
	items := []*types.MenuItem{
		{
			LabelColumns: gui.runColumns(gui.trString(gui.Tr.RunContainerBack, "back"), "", ""),
			OnPress:      func() error { return gui.showRunContainerMenu(state, selected) },
		},
	}

	for _, ctr := range containers {
		ctr := ctr
		items = append(items, &types.MenuItem{
			LabelColumns: gui.runColumns(ctr.Name, ctr.Container.Image, "enter"),
			OnPress: func() error {
				if !ctr.DetailsLoaded() {
					if err := gui.PodmanCommand.RefreshContainerDetails([]*commands.Container{ctr}); err != nil {
						return gui.createErrorPanel(err.Error())
					}
				}
				applyRunLikeContainer(state, ctr)

				return gui.showRunContainerMenu(state, selected)
			},
		})
	}

	return gui.Menu(CreateMenuOptions{
		Title:        gui.trString(gui.Tr.RunContainerRunLike, "Run like a container"),
		Items:        items,
		OnCancel:     func() error { return gui.showRunContainerMenu(state, selected) },
		SelectedLine: 0,
	})
}

func (gui *Gui) showRunContainerImagePorts(state *runContainerState, returnRow, selected int) error {
	items := []*types.MenuItem{
		{
			LabelColumns: gui.runColumns(gui.trString(gui.Tr.RunContainerBack, "back"), "", ""),
			OnPress:      func() error { return gui.showRunContainerMenu(state, returnRow) },
		},
	}

	for _, port := range state.imageInfo.Ports {
		port := port
		row := len(items)
		items = append(items, &types.MenuItem{
			LabelColumns: gui.runColumns(port, "", "enter"),
			OnPress: func() error {
				return gui.editRunContainerValue(gui.trString(gui.Tr.RunContainerHostPort, "Host port"), "", func(hostPort string) {
					if hostPort != "" {
						state.addListValue("ports", hostPort+":"+port)
					}
				}, func() error { return gui.showRunContainerImagePorts(state, returnRow, row) })
			},
		})
	}

	return gui.Menu(CreateMenuOptions{
		Title:        gui.trString(gui.Tr.RunContainerImagePorts, "Image ports"),
		Items:        items,
		OnCancel:     func() error { return gui.showRunContainerMenu(state, returnRow) },
		SelectedLine: selected,
	})
}

func (gui *Gui) showRunContainerIssues(state *runContainerState, returnRow, selected int) error {
	items := []*types.MenuItem{
		{
			LabelColumns: gui.runColumns(gui.trString(gui.Tr.RunContainerBack, "back"), "", ""),
			OnPress:      func() error { return gui.showRunContainerMenu(state, returnRow) },
		},
	}
	for _, issue := range validateRunState(state, gui.containersForRunValidation()) {
		prefix := "warning"
		if issue.Blocking {
			prefix = "error"
		}
		items = append(items, &types.MenuItem{
			LabelColumns: gui.runColumns(prefix, issue.Message, ""),
		})
	}

	return gui.Menu(CreateMenuOptions{
		Title:        gui.trString(gui.Tr.RunContainerChecks, "Checks"),
		Items:        items,
		OnCancel:     func() error { return gui.showRunContainerMenu(state, returnRow) },
		SelectedLine: selected,
	})
}
