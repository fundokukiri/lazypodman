package commands

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/go-errors/errors"
	"github.com/sasha-s/go-deadlock"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

// Container : A podman Container
type Container struct {
	Name            string
	ServiceName     string
	ContainerNumber string // might make this an int in the future if need be

	// OneOff tells us if the container is just a job container or is actually bound to the service
	OneOff      bool
	ProjectName string
	ID          string

	// PodName is the pod this container belongs to, filled from the podman CLI
	// snapshot. Empty for standalone containers and whenever pods are disabled.
	PodName     string
	Container   container.Summary
	Client      *client.Client
	OSCommand   *OSCommand
	Log         *logrus.Entry
	StatHistory []*RecordedStats
	// Details is written by the background inspect refresh. Read it through
	// DetailsSnapshot so the UI never sees a partly assigned struct.
	Details         container.InspectResponse
	MonitoringStats atomic.Bool
	PodmanCommand   LimitedPodmanCommand
	Tr              *i18n.TranslationSet

	detailsMutex sync.RWMutex
	StatsMutex   deadlock.Mutex

	// Disk usage is fetched with `podman inspect --size` when the Stats tab
	// opens, not on every stats tick.
	DiskSizeMutex  deadlock.Mutex
	DiskSizeRw     int64
	DiskSizeRootFs int64
	DiskSizeKnown  bool
}

// DetailsSnapshot returns a consistent copy of the last inspect response.
func (c *Container) DetailsSnapshot() container.InspectResponse {
	c.detailsMutex.RLock()
	defer c.detailsMutex.RUnlock()

	return c.Details
}

func (c *Container) SetDetails(details container.InspectResponse) {
	c.detailsMutex.Lock()
	defer c.detailsMutex.Unlock()

	c.Details = details
}

func (c *Container) SetDiskSize(rw, rootFs int64) {
	c.DiskSizeMutex.Lock()
	defer c.DiskSizeMutex.Unlock()

	c.DiskSizeRw = rw
	c.DiskSizeRootFs = rootFs
	c.DiskSizeKnown = true
}

func (c *Container) DiskSize() (rw int64, rootFs int64, known bool) {
	c.DiskSizeMutex.Lock()
	defer c.DiskSizeMutex.Unlock()

	return c.DiskSizeRw, c.DiskSizeRootFs, c.DiskSizeKnown
}

// Remove removes the container
func (c *Container) Remove(options container.RemoveOptions) error {
	c.Log.Warn(fmt.Sprintf("removing container %s", c.Name))
	if err := c.Client.ContainerRemove(context.Background(), c.ID, options); err != nil {
		if strings.Contains(err.Error(), "Stop the container before attempting removal or force remove") {
			return ComplexError{
				Code:    MustStopContainer,
				Message: err.Error(),
				frame:   xerrors.Caller(1),
			}
		}
		return err
	}

	return nil
}

// Start starts the container
func (c *Container) Start() error {
	c.Log.Warn(fmt.Sprintf("starting container %s", c.Name))
	return c.Client.ContainerStart(context.Background(), c.ID, container.StartOptions{})
}

// Stop stops the container
func (c *Container) Stop() error {
	c.Log.Warn(fmt.Sprintf("stopping container %s", c.Name))
	return c.Client.ContainerStop(context.Background(), c.ID, container.StopOptions{})
}

// Pause pauses the container
func (c *Container) Pause() error {
	c.Log.Warn(fmt.Sprintf("pausing container %s", c.Name))
	return c.Client.ContainerPause(context.Background(), c.ID)
}

// Unpause unpauses the container
func (c *Container) Unpause() error {
	c.Log.Warn(fmt.Sprintf("unpausing container %s", c.Name))
	return c.Client.ContainerUnpause(context.Background(), c.ID)
}

// Restart restarts the container
func (c *Container) Restart() error {
	c.Log.Warn(fmt.Sprintf("restarting container %s", c.Name))
	return c.Client.ContainerRestart(context.Background(), c.ID, container.StopOptions{})
}

// CanAttach reports why a container cannot be attached to, or nil when it can.
func (c *Container) CanAttach() error {
	details := c.DetailsSnapshot()
	if details.ContainerJSONBase == nil {
		return errors.New(c.Tr.WaitingForContainerInfo)
	}
	if details.Config == nil {
		return errors.New(c.Tr.WaitingForContainerInfo)
	}

	// verify that we can in fact attach to this container
	if !details.Config.OpenStdin {
		return errors.New(c.Tr.UnattachableContainerError)
	}

	if c.Container.State == "exited" {
		return errors.New(c.Tr.CannotAttachStoppedContainerError)
	}

	return nil
}

// Top returns process information
func (c *Container) Top(ctx context.Context) (container.TopResponse, error) {
	detail, err := c.Inspect()
	if err != nil {
		return container.TopResponse{}, err
	}

	// check container status
	if !detail.State.Running {
		return container.TopResponse{}, errors.New("container is not running")
	}

	return c.Client.ContainerTop(ctx, c.ID, []string{})
}

// PruneContainers prunes containers
func (c *PodmanCommand) PruneContainers() error {
	_, err := c.Client.ContainersPrune(context.Background(), filters.Args{})
	return err
}

// Inspect returns details about the container
func (c *Container) Inspect() (container.InspectResponse, error) {
	return c.Client.ContainerInspect(context.Background(), c.ID)
}

// RenderTop returns details about the container
func (c *Container) RenderTop(ctx context.Context) (string, error) {
	result, err := c.Top(ctx)
	if err != nil {
		return "", err
	}

	return utils.RenderTable(append([][]string{result.Titles}, result.Processes...))
}

// DetailsLoaded tells us whether we have yet loaded the details for a container.
// Sometimes it takes some time for a container to have its details loaded
// after it starts.
func (c *Container) DetailsLoaded() bool {
	return c.DetailsSnapshot().ContainerJSONBase != nil
}
