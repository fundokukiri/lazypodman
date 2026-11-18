package presentation

import "github.com/fundokukiri/lazypodman/pkg/commands"

func GetVolumeDisplayStrings(volume *commands.Volume) []string {
	return []string{volume.Volume.Driver, volume.Name}
}
