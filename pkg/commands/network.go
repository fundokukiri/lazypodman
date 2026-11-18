package commands

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

// Network : A podman Network
type Network struct {
	Name          string
	Network       network.Inspect
	Client        *client.Client
	OSCommand     *OSCommand
	Log           *logrus.Entry
	PodmanCommand LimitedPodmanCommand
}

// RefreshNetworks gets the networks and stores them
func (c *PodmanCommand) RefreshNetworks() ([]*Network, error) {
	networks, err := c.Client.NetworkList(context.Background(), network.ListOptions{})
	if err != nil {
		return nil, err
	}

	ownNetworks := make([]*Network, len(networks))

	for i, nw := range networks {
		ownNetworks[i] = &Network{
			Name:          nw.Name,
			Network:       nw,
			Client:        c.Client,
			OSCommand:     c.OSCommand,
			Log:           c.Log,
			PodmanCommand: c,
		}
	}

	return ownNetworks, nil
}

// PruneNetworks prunes networks
func (c *PodmanCommand) PruneNetworks() error {
	_, err := c.Client.NetworksPrune(context.Background(), filters.Args{})
	return err
}

// Remove removes the network
func (v *Network) Remove() error {
	return v.Client.NetworkRemove(context.Background(), v.Name)
}
