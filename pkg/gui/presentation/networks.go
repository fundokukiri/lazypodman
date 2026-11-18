package presentation

import "github.com/fundokukiri/lazypodman/pkg/commands"

func GetNetworkDisplayStrings(network *commands.Network) []string {
	return []string{network.Network.Driver, network.Name}
}
