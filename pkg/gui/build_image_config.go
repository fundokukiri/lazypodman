package gui

import (
	"strings"

	"github.com/mgutz/str"
)

// buildImageFields are the options shown on the first build screen. They are
// ordered the way they appear on the command line.
var buildImageFields = []runField{
	{Key: "context", Kind: runTextField, Default: "."},
	{Key: "containerfile", Flag: "--file", Kind: runTextField, Default: "auto"},
	{Key: "tags", Flag: "--tag", Kind: runListField},
}

// buildImageGroups hold the less common `podman build` flags.
var buildImageGroups = []runFieldGroup{
	{Key: "build_options_group", Fields: []runField{
		{Key: "format", Flag: "--format", Kind: runChoiceField, Choices: []string{"oci", "docker"}, Default: "oci"},
		{Key: "target", Flag: "--target", Kind: runTextField},
		{Key: "platform", Flag: "--platform", Kind: runTextField},
		{Key: "pull", Flag: "--pull", Kind: runChoiceField, Choices: []string{"missing", "always", "never", "newer"}, Default: "missing"},
		{Key: "jobs", Flag: "--jobs", Kind: runTextField},
		{Key: "no_cache", Flag: "--no-cache", Kind: runToggleField},
		{Key: "all_platforms", Flag: "--all-platforms", Kind: runToggleField},
		{Key: "squash", Flag: "--squash", Kind: runToggleField},
		{Key: "quiet", Flag: "--quiet", Kind: runToggleField},
	}},
	{Key: "build_args_group", Fields: []runField{
		{Key: "build_arg", Flag: "--build-arg", Kind: runListField},
		{Key: "build_arg_file", Flag: "--build-arg-file", Kind: runTextField},
		{Key: "build_env", Flag: "--env", Kind: runListField},
		{Key: "build_label", Flag: "--label", Kind: runListField},
		{Key: "build_annotation", Flag: "--annotation", Kind: runListField},
		{Key: "unsetenv", Flag: "--unsetenv", Kind: runListField},
		{Key: "unsetlabel", Flag: "--unsetlabel", Kind: runListField},
		{Key: "unsetannotation", Flag: "--unsetannotation", Kind: runListField},
	}},
	{Key: "build_network_group", Fields: []runField{
		{Key: "build_network", Flag: "--network", Kind: runTextField},
		{Key: "add_host", Flag: "--add-host", Kind: runListField},
		{Key: "dns", Flag: "--dns", Kind: runListField},
		{Key: "dns_search", Flag: "--dns-search", Kind: runListField},
		{Key: "dns_option", Flag: "--dns-option", Kind: runListField},
	}},
	{Key: "build_resources_group", Fields: []runField{
		{Key: "cpuset_cpus", Flag: "--cpuset-cpus", Kind: runTextField},
		{Key: "memory", Flag: "--memory", Kind: runTextField},
		{Key: "memory_swap", Flag: "--memory-swap", Kind: runTextField},
		{Key: "shm_size", Flag: "--shm-size", Kind: runTextField},
		{Key: "cpu_shares", Flag: "--cpu-shares", Kind: runTextField},
		{Key: "cgroup_parent", Flag: "--cgroup-parent", Kind: runTextField},
		{Key: "ulimit", Flag: "--ulimit", Kind: runListField},
		{Key: "isolation", Flag: "--isolation", Kind: runTextField},
	}},
	{Key: "build_security_group", Fields: []runField{
		{Key: "security_opt", Flag: "--security-opt", Kind: runListField},
		{Key: "cap_add", Flag: "--cap-add", Kind: runListField},
		{Key: "cap_drop", Flag: "--cap-drop", Kind: runListField},
		{Key: "userns", Flag: "--userns", Kind: runTextField},
		{Key: "userns_uid_map", Flag: "--userns-uid-map", Kind: runListField},
		{Key: "userns_gid_map", Flag: "--userns-gid-map", Kind: runListField},
		{Key: "group_add", Flag: "--group-add", Kind: runListField},
		{Key: "device", Flag: "--device", Kind: runListField},
	}},
	{Key: "build_output_group", Fields: []runField{
		{Key: "from", Flag: "--from", Kind: runTextField},
		{Key: "output", Flag: "--output", Kind: runListField},
		{Key: "iidfile", Flag: "--iidfile", Kind: runTextField},
		{Key: "metadata_file", Flag: "--metadata-file", Kind: runTextField},
		{Key: "manifest", Flag: "--manifest", Kind: runTextField},
		{Key: "cache_from", Flag: "--cache-from", Kind: runListField},
		{Key: "cache_to", Flag: "--cache-to", Kind: runListField},
		{Key: "cache_ttl", Flag: "--cache-ttl", Kind: runTextField},
		{Key: "timestamp", Flag: "--timestamp", Kind: runTextField},
		{Key: "source_date_epoch", Flag: "--source-date-epoch", Kind: runTextField},
		{Key: "ignorefile", Flag: "--ignorefile", Kind: runTextField},
	}},
	{Key: "build_auth_group", Fields: []runField{
		{Key: "authfile", Flag: "--authfile", Kind: runTextField},
		{Key: "cert_dir", Flag: "--cert-dir", Kind: runTextField},
		{Key: "creds", Flag: "--creds", Kind: runTextField},
		{Key: "secret", Flag: "--secret", Kind: runListField},
		{Key: "ssh", Flag: "--ssh", Kind: runListField},
		{Key: "build_volume", Flag: "--volume", Kind: runListField},
		{Key: "build_mount", Flag: "--mount", Kind: runListField},
	}},
}

func newBuildImageState() *runContainerState {
	return &runContainerState{
		text:    map[string]string{"context": "."},
		toggles: map[string]bool{},
		lists:   map[string][]string{},
	}
}

// buildArgv builds the exact argument list after "build". It never uses a shell.
func (s *runContainerState) buildArgv(extraArgs string) []string {
	args := []string{"build"}

	appendField := func(field runField) {
		switch field.Kind {
		case runToggleField:
			if s.toggles[field.Key] {
				args = append(args, field.Flag)
			}
		case runListField:
			for _, value := range s.lists[field.Key] {
				value = strings.TrimSpace(value)
				if value != "" {
					args = append(args, field.Flag, value)
				}
			}
		case runTextField, runChoiceField:
			value := strings.TrimSpace(s.text[field.Key])
			if value != "" {
				args = append(args, field.Flag, value)
			}
		}
	}

	for _, field := range buildImageFields {
		if field.Flag == "" {
			continue // context is positional
		}
		appendField(field)
	}
	for _, group := range buildImageGroups {
		for _, field := range group.Fields {
			appendField(field)
		}
	}

	args = append(args, str.ToArgv(extraArgs)...)

	context := strings.TrimSpace(s.text["context"])
	if context != "" {
		args = append(args, context)
	}

	return args
}

func buildFieldByKey(key string) runField {
	for _, field := range buildImageFields {
		if field.Key == key {
			return field
		}
	}
	for _, group := range buildImageGroups {
		for _, field := range group.Fields {
			if field.Key == key {
				return field
			}
		}
	}

	return runField{}
}
