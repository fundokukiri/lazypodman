package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	ogLog "log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/fundokukiri/lazypodman/pkg/commands/ssh"
	"github.com/fundokukiri/lazypodman/pkg/config"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/imdario/mergo"
	"github.com/sasha-s/go-deadlock"
	"github.com/sirupsen/logrus"
)

const (
	// podmanHostEnvKey is the environment variable lazypodman uses to advertise
	// the endpoint it decided to talk to. It is read back by the SSH tunnel
	// handler, which rewrites an `ssh://` host into a local socket.
	podmanHostEnvKey = "CONTAINER_HOST"
)

// PodmanCommand is our main Podman interface
type PodmanCommand struct {
	Log              *logrus.Entry
	OSCommand        *OSCommand
	Tr               *i18n.TranslationSet
	Config           *config.AppConfig
	Client           *client.Client
	InComposeProject bool
	// LocalProjectName is the compose project name for the directory where lazypodman was launched.
	LocalProjectName string
	ErrorChan        chan error
	ContainerMutex   deadlock.Mutex
	ServiceMutex     deadlock.Mutex

	Closers []io.Closer
}

var _ io.Closer = &PodmanCommand{}

// LimitedPodmanCommand is a stripped-down PodmanCommand with just the methods the container/service/image might need
type LimitedPodmanCommand interface {
	NewCommandObject(CommandObject) CommandObject
}

// CommandObject is what we pass to our template resolvers when we are running a custom command. We do not guarantee that all fields will be populated: just the ones that make sense for the current context
type CommandObject struct {
	Podman        string
	PodmanCompose string
	Service       *Service
	Container     *Container
	Image         *Image
	Volume        *Volume
	Network       *Network
	Project       *Project
}

// NewCommandObject takes a command object and returns a default command object with the passed command object merged in
func (c *PodmanCommand) NewCommandObject(obj CommandObject) CommandObject {
	defaultObj := CommandObject{
		Podman:        c.Config.PodmanBinary(),
		PodmanCompose: c.Config.UserConfig.CommandTemplates.PodmanCompose,
	}
	_ = mergo.Merge(&defaultObj, obj)

	// When operating on a specific project, include -p flag so that
	// podman compose targets the correct project.
	if obj.Service != nil && obj.Service.ProjectName != "" {
		defaultObj.PodmanCompose = fmt.Sprintf("%s -p %s", defaultObj.PodmanCompose, obj.Service.ProjectName)
	} else if obj.Project != nil && obj.Project.Name != "" {
		defaultObj.PodmanCompose = fmt.Sprintf("%s -p %s", defaultObj.PodmanCompose, obj.Project.Name)
	}

	return defaultObj
}

// newPodmanClient creates a Podman API client with the given host. Podman
// serves a Docker-compatible API, so we use the Docker client against it.
//
// We avoid using client.FromEnv because it includes WithVersionFromEnv() which
// sets manualOverride=true when DOCKER_API_VERSION is set, preventing API version
// negotiation even when WithAPIVersionNegotiation() is specified.
// Instead, we explicitly configure only what we need, and rely on proper
// API version negotiation to stay compatible with older Podman releases.
func newPodmanClient(podmanHost string) (*client.Client, error) {
	return client.NewClientWithOpts(
		client.WithTLSClientConfigFromEnv(),
		client.WithAPIVersionNegotiation(),
		client.WithHost(podmanHost),
	)
}

// NewPodmanCommand talks to the Podman service and shells out to podman for
// compose-related commands
func NewPodmanCommand(log *logrus.Entry, osCommand *OSCommand, tr *i18n.TranslationSet, config *config.AppConfig, errorChan chan error) (*PodmanCommand, error) {
	podmanHost, err := determinePodmanHost()
	if err != nil {
		ogLog.Printf("> could not determine host %v", err)
	}

	// NOTE: Inject the determined endpoint into the environment. This allows the
	//       `SSHHandler.HandleSSHContainerHost()` to create a local unix socket tunneled
	//       over SSH to the specified ssh host.
	if strings.HasPrefix(podmanHost, "ssh://") {
		os.Setenv(podmanHostEnvKey, podmanHost)
	}

	tunnelCloser, err := ssh.NewSSHHandler(osCommand).HandleSSHContainerHost()
	if err != nil {
		ogLog.Fatal(err)
	}

	// Retrieve the endpoint from the environment which could have been set by
	// the `SSHHandler.HandleSSHContainerHost()` and override `podmanHost`.
	podmanHostFromEnv := os.Getenv(podmanHostEnvKey)
	if podmanHostFromEnv != "" {
		podmanHost = podmanHostFromEnv
	}

	cli, err := newPodmanClient(podmanHost)
	if err != nil {
		ogLog.Fatal(err)
	}

	podmanCommand := &PodmanCommand{
		Log:              log,
		OSCommand:        osCommand,
		Tr:               tr,
		Config:           config,
		Client:           cli,
		ErrorChan:        errorChan,
		InComposeProject: true,
		Closers:          []io.Closer{tunnelCloser},
	}

	podmanCommand.setPodmanComposeCommand(config)

	err = osCommand.RunCommand(
		utils.ApplyTemplate(
			config.UserConfig.CommandTemplates.CheckComposeConfig,
			podmanCommand.NewCommandObject(CommandObject{}),
		),
	)
	if err != nil {
		podmanCommand.InComposeProject = false
		log.Warn(err.Error())
	}

	// When the user passes -p outside of a compose directory, treat it as the
	// local project so the project/services panels still appear and filtering
	// is applied. Inside a compose dir, LocalProjectName is derived from
	// container labels later in RefreshContainersAndServices.
	if !podmanCommand.InComposeProject && config.ProjectName != "" {
		podmanCommand.LocalProjectName = config.ProjectName
	}

	return podmanCommand, nil
}

// IsProjectScoped reports whether lazypodman should be scoped to a single
// compose project — either because we're inside a compose directory or
// because the user passed -p. When false, the project/services panels are
// hidden and all containers are shown in a flat list.
func (c *PodmanCommand) IsProjectScoped() bool {
	return c.InComposeProject || c.Config.ProjectName != ""
}

// setPodmanComposeCommand picks the compose implementation to use. Podman
// ships `podman compose` (which delegates to a compose provider) on newer
// versions, but plenty of users have only the standalone `podman-compose`
// script, so we check both and remember which one works.
func (c *PodmanCommand) setPodmanComposeCommand(config *config.AppConfig) {
	if config.UserConfig.CommandTemplates.PodmanCompose != "podman compose" {
		return
	}

	if err := c.OSCommand.RunCommand("podman compose version"); err == nil {
		return
	}

	// `podman compose` is missing (or has no provider configured); fall back to
	// the standalone podman-compose implementation.
	config.UserConfig.CommandTemplates.PodmanCompose = "podman-compose"
}

func (c *PodmanCommand) Close() error {
	return utils.CloseMany(c.Closers)
}

func (c *PodmanCommand) CreateClientStatMonitor(container *Container) {
	container.MonitoringStats = true
	stream, err := c.Client.ContainerStats(context.Background(), container.ID, true)
	if err != nil {
		// not creating error panel because if we've disconnected from podman we'll
		// have already created an error panel
		c.Log.Error(err)
		container.MonitoringStats = false
		return
	}

	defer stream.Body.Close()

	scanner := bufio.NewScanner(stream.Body)
	for scanner.Scan() {
		data := scanner.Bytes()
		var stats ContainerStats
		_ = json.Unmarshal(data, &stats)

		recordedStats := &RecordedStats{
			ClientStats: stats,
			DerivedStats: DerivedStats{
				CPUPercentage:    stats.CalculateContainerCPUPercentage(),
				MemoryPercentage: stats.CalculateContainerMemoryUsage(),
			},
			RecordedAt: time.Now(),
		}

		container.appendStats(recordedStats, c.Config.UserConfig.Stats.MaxDuration)
	}

	container.MonitoringStats = false
}

func (c *PodmanCommand) RefreshContainersAndServices(currentContainers []*Container) ([]*Container, []*Service, error) {
	c.ServiceMutex.Lock()
	defer c.ServiceMutex.Unlock()

	containers, err := c.GetContainers(currentContainers)
	if err != nil {
		return nil, nil, err
	}

	// Derive services from container labels (covers all projects)
	services := c.GetServicesFromContainers(containers)

	var composeServices []*Service
	if c.InComposeProject {
		composeServices, err = c.GetServices()
		if err != nil {
			c.Log.Warn("Failed to get compose services: " + err.Error())
		}
	}

	// Determine the local project name before merging services, since
	// mergeServices needs it. We match compose service names against container
	// labels to handle cases where the project name differs from the directory
	// name (e.g. a `name:` directive in the compose file).
	if c.LocalProjectName == "" && c.InComposeProject && composeServices != nil {
		for _, ctr := range containers {
			if ctr.ProjectName == "" || ctr.ServiceName == "" {
				continue
			}
			for _, svc := range composeServices {
				if ctr.ServiceName == svc.Name {
					c.LocalProjectName = ctr.ProjectName
					break
				}
			}
			if c.LocalProjectName != "" {
				break
			}
		}
		// Fall back to directory name
		if c.LocalProjectName == "" && c.Config.ProjectDir != "" {
			c.LocalProjectName = filepath.Base(c.Config.ProjectDir)
		}
	}

	// Merge compose services (which include stopped services) with
	// container-derived services from all projects
	if composeServices != nil {
		services = c.mergeServices(services, composeServices)
	}

	c.assignContainersToServices(containers, services)

	return containers, services, nil
}

// GetServicesFromContainers derives services from container labels for all projects
func (c *PodmanCommand) GetServicesFromContainers(containers []*Container) []*Service {
	// Use project+service as key to avoid duplicates
	type serviceKey struct {
		project string
		service string
	}
	seen := make(map[serviceKey]bool)
	services := make([]*Service, 0, len(containers))

	for _, ctr := range containers {
		if ctr.ServiceName == "" || ctr.OneOff {
			continue
		}
		key := serviceKey{project: ctr.ProjectName, service: ctr.ServiceName}
		if seen[key] {
			continue
		}
		seen[key] = true
		services = append(services, &Service{
			Name:          ctr.ServiceName,
			ID:            ctr.ProjectName + "-" + ctr.ServiceName,
			ProjectName:   ctr.ProjectName,
			OSCommand:     c.OSCommand,
			Log:           c.Log,
			PodmanCommand: c,
		})
	}

	return services
}

// mergeServices merges compose services (which may lack ProjectName) with
// container-derived services. Compose services take priority because they
// include services without running containers.
func (c *PodmanCommand) mergeServices(containerServices []*Service, composeServices []*Service) []*Service {
	// Set project name on compose services
	for _, svc := range composeServices {
		if svc.ProjectName == "" {
			svc.ProjectName = c.LocalProjectName
		}
	}

	// Build a set of compose service names for the local project
	composeServiceNames := make(map[string]bool)
	for _, svc := range composeServices {
		composeServiceNames[svc.Name] = true
	}

	// Start with compose services, then add container-derived services
	// that aren't already covered by compose (i.e. from other projects)
	result := make([]*Service, 0, len(composeServices)+len(containerServices))
	result = append(result, composeServices...)

	for _, svc := range containerServices {
		if svc.ProjectName == c.LocalProjectName && composeServiceNames[svc.Name] {
			continue // already covered by compose service
		}
		result = append(result, svc)
	}

	return result
}

// GetProjectNames returns all unique project names from containers
func (c *PodmanCommand) GetProjectNames(containers []*Container) []string {
	seen := make(map[string]bool)
	var names []string
	for _, ctr := range containers {
		if ctr.ProjectName != "" && !seen[ctr.ProjectName] {
			seen[ctr.ProjectName] = true
			names = append(names, ctr.ProjectName)
		}
	}
	sort.Strings(names)
	return names
}

func (c *PodmanCommand) assignContainersToServices(containers []*Container, services []*Service) {
L:
	for _, service := range services {
		for _, ctr := range containers {
			if !ctr.OneOff && ctr.ServiceName == service.Name && ctr.ProjectName == service.ProjectName {
				service.Container = ctr
				continue L
			}
		}
		service.Container = nil
	}
}

// GetContainers gets the podman containers
func (c *PodmanCommand) GetContainers(existingContainers []*Container) ([]*Container, error) {
	c.ContainerMutex.Lock()
	defer c.ContainerMutex.Unlock()

	containers, err := c.Client.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	ownContainers := make([]*Container, 0, len(containers))

	for _, ctr := range containers {
		ctr := ctr

		// Podman runs one hidden "infra" (pause) container per pod. They are
		// noise in a container list, so hide them just like `podman ps` does.
		if isPodmanInfraContainer(ctr) {
			continue
		}
		var newContainer *Container

		// check if we already have data stored against the container
		for _, existingContainer := range existingContainers {
			if existingContainer.ID == ctr.ID {
				newContainer = existingContainer
				break
			}
		}

		// initialise the container if it's completely new
		if newContainer == nil {
			newContainer = &Container{
				ID:            ctr.ID,
				Client:        c.Client,
				OSCommand:     c.OSCommand,
				Log:           c.Log,
				PodmanCommand: c,
				Tr:            c.Tr,
			}
		}

		newContainer.Container = ctr
		// if the container is made with a name label we will use that
		if name, ok := ctr.Labels["name"]; ok {
			newContainer.Name = name
		} else {
			if len(ctr.Names) > 0 {
				newContainer.Name = strings.TrimLeft(ctr.Names[0], "/")
			} else {
				newContainer.Name = ctr.ID
			}
		}
		// podman-compose writes both the Docker-compatible labels and its own
		// io.podman.compose.* ones, so accept whichever we find.
		newContainer.ServiceName = labelValue(ctr.Labels, "com.docker.compose.service", "io.podman.compose.service")
		newContainer.ProjectName = labelValue(ctr.Labels, "com.docker.compose.project", "io.podman.compose.project")
		newContainer.ContainerNumber = labelValue(ctr.Labels, "com.docker.compose.container", "io.podman.compose.container-number")
		newContainer.OneOff = labelValue(ctr.Labels, "com.docker.compose.oneoff") == "True"

		ownContainers = append(ownContainers, newContainer)
	}

	c.SetContainerDetails(ownContainers)

	return ownContainers, nil
}

// GetServices gets services
func (c *PodmanCommand) GetServices() ([]*Service, error) {
	if !c.InComposeProject {
		return nil, nil
	}

	composeCommand := c.Config.UserConfig.CommandTemplates.PodmanCompose
	output, err := c.OSCommand.RunCommandWithOutput(fmt.Sprintf("%s config --services", composeCommand))
	if err != nil {
		return nil, err
	}

	// output looks like:
	// service1
	// service2

	lines := utils.SplitLines(output)
	services := make([]*Service, len(lines))
	for i, str := range lines {
		services[i] = &Service{
			Name:          str,
			ID:            c.LocalProjectName + "-" + str,
			ProjectName:   c.LocalProjectName,
			OSCommand:     c.OSCommand,
			Log:           c.Log,
			PodmanCommand: c,
		}
	}

	return services, nil
}

func (c *PodmanCommand) RefreshContainerDetails(containers []*Container) error {
	c.ContainerMutex.Lock()
	defer c.ContainerMutex.Unlock()

	c.SetContainerDetails(containers)

	return nil
}

// Attaches the details returned from podman inspect to each of the containers
// this contains a bit more info than what you get from the go-docker client
func (c *PodmanCommand) SetContainerDetails(containers []*Container) {
	wg := sync.WaitGroup{}
	for _, ctr := range containers {
		ctr := ctr
		wg.Add(1)
		go func() {
			details, err := c.Client.ContainerInspect(context.Background(), ctr.ID)
			if err != nil {
				c.Log.Error(err)
			} else {
				ctr.Details = details
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

// ViewAllLogs attaches to a subprocess viewing all the logs from podman-compose
func (c *PodmanCommand) ViewAllLogs(project *Project) (*exec.Cmd, error) {
	cmd := c.OSCommand.ExecutableFromString(
		utils.ApplyTemplate(
			c.OSCommand.Config.UserConfig.CommandTemplates.ViewAllLogs,
			c.NewCommandObject(CommandObject{Project: project}),
		),
	)

	c.OSCommand.PrepareForChildren(cmd)

	return cmd, nil
}

// ComposeConfig returns the result of 'podman-compose config'
func (c *PodmanCommand) ComposeConfig() string {
	return c.ComposeConfigForProject(nil)
}

// ComposeConfigForProject returns the result of 'podman-compose config' for a specific project
func (c *PodmanCommand) ComposeConfigForProject(project *Project) string {
	output, err := c.OSCommand.RunCommandWithOutput(
		utils.ApplyTemplate(
			c.OSCommand.Config.UserConfig.CommandTemplates.ComposeConfig,
			c.NewCommandObject(CommandObject{Project: project}),
		),
	)
	if err != nil {
		output = err.Error()
	}
	return output
}
