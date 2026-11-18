package presentation

import "github.com/fundokukiri/lazypodman/pkg/commands"

func GetProjectDisplayStrings(project *commands.Project) []string {
	return []string{project.Name}
}
