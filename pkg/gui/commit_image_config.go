package gui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/mgutz/str"
)

var commitImageFields = []runField{
	{Key: "image", Kind: runTextField},
	{Key: "author", Flag: "--author", Kind: runTextField},
	{Key: "message", Flag: "--message", Kind: runTextField},
	{Key: "changes", Flag: "--change", Kind: runListField},
}

var commitImageGroups = []runFieldGroup{
	{Key: "commit_options_group", Fields: []runField{
		{Key: "config", Flag: "--config", Kind: runTextField},
		{Key: "format", Flag: "--format", Kind: runChoiceField, Choices: []string{"oci", "docker"}, Default: "oci"},
		{Key: "iidfile", Flag: "--iidfile", Kind: runTextField},
		{Key: "include_volumes", Flag: "--include-volumes", Kind: runToggleField},
		{Key: "quiet", Flag: "--quiet", Kind: runToggleField},
		{Key: "squash", Flag: "--squash", Kind: runToggleField},
	}},
}

func newCommitImageState(ctr *commands.Container) *runContainerState {
	state := &runContainerState{
		text:    map[string]string{"image": defaultCommitImageName(ctr.Name, time.Now())},
		toggles: map[string]bool{"pause": true},
		lists:   map[string][]string{},
	}
	return state
}

func defaultCommitImageName(container string, now time.Time) string {
	base := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			return r
		}
		return '-'
	}, container)
	base = strings.Trim(base, "-._")
	if base == "" {
		base = "container"
	}
	if len(base) > 40 {
		base = base[:40]
	}

	return fmt.Sprintf("%s_%s", base, now.Format("20060102_150405"))
}

func (s *runContainerState) commitArgv(containerRef, extraArgs string) []string {
	args := []string{"commit"}

	for _, field := range commitImageFields {
		if field.Flag == "" {
			continue
		}
		args = appendRunField(args, s, field)
	}
	for _, group := range commitImageGroups {
		for _, field := range group.Fields {
			args = appendRunField(args, s, field)
		}
	}

	if !s.toggles["pause"] {
		args = append(args, "--pause=false")
	}
	args = append(args, str.ToArgv(extraArgs)...)
	args = append(args, "--", containerRef)

	if image := strings.TrimSpace(s.text["image"]); image != "" {
		args = append(args, image)
	}

	return args
}
