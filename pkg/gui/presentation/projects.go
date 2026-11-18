package presentation

import "github.com/lazypodman/lazypodman/pkg/commands"

func GetProjectDisplayStrings(project *commands.Project) []string {
	return []string{project.Name}
}
