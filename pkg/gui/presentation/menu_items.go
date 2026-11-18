package presentation

import "github.com/fundokukiri/lazypodman/pkg/gui/types"

func GetMenuItemDisplayStrings(menuItem *types.MenuItem) []string {
	return menuItem.LabelColumns
}
