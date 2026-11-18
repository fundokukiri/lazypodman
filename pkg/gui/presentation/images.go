package presentation

import (
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/utils"
)

func GetImageDisplayStrings(image *commands.Image) []string {
	return []string{
		image.Name,
		image.Tag,
		utils.FormatDecimalBytes(int(image.Image.Size)),
	}
}
