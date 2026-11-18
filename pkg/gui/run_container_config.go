package gui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/mgutz/str"
)

// runFieldKind decides how a schema field is edited and rendered.
type runFieldKind int

const (
	runTextField runFieldKind = iota
	runToggleField
	runListField
	runChoiceField
)

type runField struct {
	// Key is both the state key and the i18n label key.
	Key     string
	Flag    string
	Kind    runFieldKind
	Choices []string
	// Equals renders `--flag=value` instead of `--flag value`.
	Equals bool
	// Default is the value Podman uses when the flag is omitted.
	Default string
	// DefaultOn is the value of a toggle when the flag is omitted.
	DefaultOn bool
}

type runFieldGroup struct {
	Key    string
	Fields []runField
}

// runContainerFields is the root menu. Order is also argv order.
var runContainerFields = []runField{
	{Key: "name", Flag: "--name", Kind: runTextField},
	{Key: "command", Kind: runTextField},
	{Key: "entrypoint", Flag: "--entrypoint", Kind: runTextField},
	{Key: "workdir", Flag: "--workdir", Kind: runTextField},
	{Key: "user", Flag: "-u", Kind: runTextField},
	{Key: "hostname", Flag: "--hostname", Kind: runTextField, Default: "container_id"},
	{Key: "ports", Flag: "-p", Kind: runListField},
	{Key: "volumes", Flag: "-v", Kind: runListField},
	{Key: "env", Flag: "-e", Kind: runListField},
	{Key: "labels", Flag: "-l", Kind: runListField},
	{Key: "network", Flag: "--network", Kind: runTextField, Default: "podman"},
	{Key: "restart", Flag: "--restart", Kind: runChoiceField, Choices: []string{"no", "on-failure", "always", "unless-stopped"}, Default: "no"},
	{Key: "detach", Flag: "-d", Kind: runToggleField, DefaultOn: true},
	{Key: "rm", Flag: "--rm", Kind: runToggleField},
	{Key: "interactive", Flag: "-i", Kind: runToggleField},
	{Key: "tty", Flag: "-t", Kind: runToggleField},
}

// runContainerGroups holds everything behind the Advanced menu, grouped so the
// root menu stays readable.
var runContainerGroups = []runFieldGroup{
	{
		Key: "network_group",
		Fields: []runField{
			{Key: "publish_all", Flag: "--publish-all", Kind: runToggleField},
			{Key: "expose", Flag: "--expose", Kind: runListField},
			{Key: "network_alias", Flag: "--network-alias", Kind: runListField},
			{Key: "ip", Flag: "--ip", Kind: runTextField},
			{Key: "ip6", Flag: "--ip6", Kind: runTextField},
			{Key: "mac", Flag: "--mac-address", Kind: runTextField},
			{Key: "dns", Flag: "--dns", Kind: runListField},
			{Key: "dns_search", Flag: "--dns-search", Kind: runListField},
			{Key: "dns_option", Flag: "--dns-option", Kind: runListField},
			{Key: "add_host", Flag: "--add-host", Kind: runListField},
			{Key: "hosts_file", Flag: "--hosts-file", Kind: runTextField},
			{Key: "ipc", Flag: "--ipc", Kind: runTextField},
			{Key: "pid", Flag: "--pid", Kind: runTextField},
			{Key: "uts", Flag: "--uts", Kind: runTextField},
			{Key: "cgroupns", Flag: "--cgroupns", Kind: runTextField},
			{Key: "pod", Flag: "--pod", Kind: runTextField},
			{Key: "userns", Flag: "--userns", Kind: runChoiceField, Choices: []string{"host", "private", "auto", "keep-id", "nomap"}},
			{Key: "uidmap", Flag: "--uidmap", Kind: runListField},
			{Key: "gidmap", Flag: "--gidmap", Kind: runListField},
			{Key: "subuidname", Flag: "--subuidname", Kind: runTextField},
			{Key: "subgidname", Flag: "--subgidname", Kind: runTextField},
		},
	},
	{
		Key: "storage_group",
		Fields: []runField{
			{Key: "mount", Flag: "--mount", Kind: runListField},
			{Key: "tmpfs", Flag: "--tmpfs", Kind: runListField},
			{Key: "volumes_from", Flag: "--volumes-from", Kind: runListField},
			{Key: "read_only", Flag: "--read-only", Kind: runToggleField},
			{Key: "image_volume", Flag: "--image-volume", Kind: runChoiceField, Choices: []string{"anonymous", "tmpfs", "ignore"}, Default: "anonymous"},
			{Key: "chrootdirs", Flag: "--chrootdirs", Kind: runListField},
		},
	},
	{
		Key: "resources_group",
		Fields: []runField{
			{Key: "cpus", Flag: "--cpus", Kind: runTextField},
			{Key: "cpuset_cpus", Flag: "--cpuset-cpus", Kind: runTextField},
			{Key: "cpu_shares", Flag: "--cpu-shares", Kind: runTextField},
			{Key: "cpu_period", Flag: "--cpu-period", Kind: runTextField},
			{Key: "cpu_quota", Flag: "--cpu-quota", Kind: runTextField},
			{Key: "memory", Flag: "--memory", Kind: runTextField},
			{Key: "memory_reservation", Flag: "--memory-reservation", Kind: runTextField},
			{Key: "memory_swap", Flag: "--memory-swap", Kind: runTextField},
			{Key: "memory_swappiness", Flag: "--memory-swappiness", Kind: runTextField},
			{Key: "shm_size", Flag: "--shm-size", Kind: runTextField},
			{Key: "pids_limit", Flag: "--pids-limit", Kind: runTextField},
			{Key: "ulimit", Flag: "--ulimit", Kind: runListField},
			{Key: "blkio_weight", Flag: "--blkio-weight", Kind: runTextField},
			{Key: "device_read_bps", Flag: "--device-read-bps", Kind: runListField},
			{Key: "device_write_bps", Flag: "--device-write-bps", Kind: runListField},
			{Key: "device_read_iops", Flag: "--device-read-iops", Kind: runListField},
			{Key: "device_write_iops", Flag: "--device-write-iops", Kind: runListField},
			{Key: "oom_score_adj", Flag: "--oom-score-adj", Kind: runTextField},
			{Key: "oom_kill_disable", Flag: "--oom-kill-disable", Kind: runToggleField},
		},
	},
	{
		Key: "security_group",
		Fields: []runField{
			{Key: "privileged", Flag: "--privileged", Kind: runToggleField},
			{Key: "cap_add", Flag: "--cap-add", Kind: runListField},
			{Key: "cap_drop", Flag: "--cap-drop", Kind: runListField},
			{Key: "security_opt", Flag: "--security-opt", Kind: runListField},
			{Key: "seccomp_policy", Flag: "--seccomp-policy", Kind: runTextField},
			{Key: "device", Flag: "--device", Kind: runListField},
			{Key: "device_cgroup_rule", Flag: "--device-cgroup-rule", Kind: runListField},
			{Key: "group_add", Flag: "--group-add", Kind: runListField},
			{Key: "gpus", Flag: "--gpus", Kind: runTextField},
			{Key: "no_new_privileges", Flag: "--security-opt", Kind: runToggleField},
		},
	},
	{
		Key: "health_group",
		Fields: []runField{
			{Key: "health_cmd", Flag: "--health-cmd", Kind: runTextField},
			{Key: "health_interval", Flag: "--health-interval", Kind: runTextField, Default: "30s"},
			{Key: "health_timeout", Flag: "--health-timeout", Kind: runTextField, Default: "30s"},
			{Key: "health_retries", Flag: "--health-retries", Kind: runTextField, Default: "3"},
			{Key: "health_start_period", Flag: "--health-start-period", Kind: runTextField, Default: "0s"},
			{Key: "health_start_interval", Flag: "--health-start-interval", Kind: runTextField, Default: "0s"},
			{Key: "health_startup_cmd", Flag: "--health-startup-cmd", Kind: runTextField},
			{Key: "health_startup_interval", Flag: "--health-startup-interval", Kind: runTextField},
			{Key: "health_startup_retries", Flag: "--health-startup-retries", Kind: runTextField},
			{Key: "health_startup_success", Flag: "--health-startup-success", Kind: runTextField},
			{Key: "health_on_failure", Flag: "--health-on-failure", Kind: runChoiceField, Choices: []string{"none", "kill", "restart", "stop"}, Default: "none"},
			{Key: "health_log_destination", Flag: "--health-log-destination", Kind: runTextField, Default: "local"},
			{Key: "health_max_log_count", Flag: "--health-max-log-count", Kind: runTextField, Default: "5"},
			{Key: "health_max_log_size", Flag: "--health-max-log-size", Kind: runTextField, Default: "500"},
			{Key: "no_healthcheck", Flag: "--no-healthcheck", Kind: runToggleField},
		},
	},
	{
		Key: "runtime_group",
		Fields: []runField{
			{Key: "init", Flag: "--init", Kind: runToggleField},
			{Key: "systemd", Flag: "--systemd", Kind: runChoiceField, Choices: []string{"true", "false", "always"}, Default: "true"},
			{Key: "sdnotify", Flag: "--sdnotify", Kind: runChoiceField, Choices: []string{"container", "conmon", "healthy", "ignore"}},
			{Key: "stop_signal", Flag: "--stop-signal", Kind: runTextField},
			{Key: "stop_timeout", Flag: "--stop-timeout", Kind: runTextField, Default: "10"},
			{Key: "timeout", Flag: "--timeout", Kind: runTextField},
			{Key: "log_driver", Flag: "--log-driver", Kind: runTextField, Default: "journald"},
			{Key: "log_opt", Flag: "--log-opt", Kind: runListField},
			{Key: "pull", Flag: "--pull", Kind: runChoiceField, Choices: []string{"always", "missing", "never", "newer"}, Default: "missing"},
			{Key: "retry", Flag: "--retry", Kind: runTextField},
			{Key: "retry_delay", Flag: "--retry-delay", Kind: runTextField},
			{Key: "platform", Flag: "--platform", Kind: runTextField},
			{Key: "arch", Flag: "--arch", Kind: runTextField},
			{Key: "os", Flag: "--os", Kind: runTextField},
			{Key: "variant", Flag: "--variant", Kind: runTextField},
			{Key: "cidfile", Flag: "--cidfile", Kind: runTextField},
			{Key: "pidfile", Flag: "--pidfile", Kind: runTextField},
			{Key: "conmon_pidfile", Flag: "--conmon-pidfile", Kind: runTextField},
			{Key: "replace", Flag: "--replace", Kind: runToggleField},
			{Key: "requires", Flag: "--requires", Kind: runListField},
			{Key: "tz", Flag: "--tz", Kind: runTextField},
			{Key: "umask", Flag: "--umask", Kind: runTextField, Default: "0022"},
			{Key: "detach_keys", Flag: "--detach-keys", Kind: runTextField},
			{Key: "attach", Flag: "--attach", Kind: runListField},
			{Key: "sig_proxy", Flag: "--sig-proxy", Kind: runChoiceField, Choices: []string{"true", "false"}, Equals: true, Default: "true"},
			{Key: "http_proxy", Flag: "--http-proxy", Kind: runChoiceField, Choices: []string{"true", "false"}, Equals: true, Default: "true"},
			{Key: "tls_verify", Flag: "--tls-verify", Kind: runChoiceField, Choices: []string{"true", "false"}, Equals: true, Default: "true"},
			{Key: "authfile", Flag: "--authfile", Kind: runTextField},
			{Key: "creds", Flag: "--creds", Kind: runTextField},
			{Key: "cert_dir", Flag: "--cert-dir", Kind: runTextField},
		},
	},
	{
		Key: "environment_group",
		Fields: []runField{
			{Key: "env_file", Flag: "--env-file", Kind: runListField},
			{Key: "env_host", Flag: "--env-host", Kind: runToggleField},
			{Key: "env_merge", Flag: "--env-merge", Kind: runListField},
			{Key: "unsetenv", Flag: "--unsetenv", Kind: runListField},
			{Key: "unsetenv_all", Flag: "--unsetenv-all", Kind: runToggleField},
			{Key: "label_file", Flag: "--label-file", Kind: runListField},
			{Key: "annotation", Flag: "--annotation", Kind: runListField},
			{Key: "preserve_fds", Flag: "--preserve-fds", Kind: runTextField},
			{Key: "preserve_fd", Flag: "--preserve-fd", Kind: runListField},
			{Key: "personality", Flag: "--personality", Kind: runTextField},
		},
	},
}

// runContainerState is the editable form. The image is fixed to the selection
// when the menu opens but can still be edited as a field.
type runContainerState struct {
	image     string
	extraArgs string
	text      map[string]string
	toggles   map[string]bool
	lists     map[string][]string
}

func newRunContainerState(image string) *runContainerState {
	state := &runContainerState{
		image:   image,
		text:    map[string]string{},
		toggles: map[string]bool{},
		lists:   map[string][]string{},
	}
	state.toggles["detach"] = true
	state.text["name"] = defaultRunContainerName(image, time.Now())

	return state
}

// defaultRunContainerName builds a readable, unique-enough name from the image.
func defaultRunContainerName(image string, now time.Time) string {
	base := image
	if index := strings.LastIndex(base, "/"); index >= 0 {
		base = base[index+1:]
	}
	if index := strings.Index(base, ":"); index >= 0 {
		base = base[:index]
	}

	base = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			return r
		}
		return '-'
	}, base)
	base = strings.Trim(base, "-._")
	if base == "" {
		base = "container"
	}
	if len(base) > 40 {
		base = base[:40]
	}

	return fmt.Sprintf("%s_%s", base, now.Format("20060102_150405"))
}

// argv builds the exact argument list after "run". It never uses a shell.
func (s *runContainerState) argv(extraArgs string) []string {
	args := []string{"run"}

	appendField := func(field runField) {
		switch field.Kind {
		case runToggleField:
			if !s.toggles[field.Key] {
				return
			}
			if field.Key == "no_new_privileges" {
				args = append(args, "--security-opt", "no-new-privileges")
				return
			}
			args = append(args, field.Flag)
		case runListField:
			for _, value := range s.lists[field.Key] {
				value = strings.TrimSpace(value)
				if value != "" {
					args = append(args, field.Flag, value)
				}
			}
		default:
			value := strings.TrimSpace(s.text[field.Key])
			if value == "" {
				return
			}
			if field.Equals {
				args = append(args, field.Flag+"="+value)
				return
			}
			args = append(args, field.Flag, value)
		}
	}

	for _, field := range runContainerFields {
		if field.Key == "command" {
			continue
		}
		appendField(field)
	}
	for _, group := range runContainerGroups {
		for _, field := range group.Fields {
			appendField(field)
		}
	}

	for _, arg := range str.ToArgv(extraArgs) {
		if arg != "" {
			args = append(args, arg)
		}
	}

	args = append(args, s.image)

	for _, arg := range str.ToArgv(s.text["command"]) {
		if arg != "" {
			args = append(args, arg)
		}
	}

	return args
}

func (s *runContainerState) addListValue(key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	s.lists[key] = append(s.lists[key], value)
}

func (s *runContainerState) removeListValue(key string, index int) {
	values := s.lists[key]
	if index < 0 || index >= len(values) {
		return
	}
	s.lists[key] = append(values[:index], values[index+1:]...)
}

func (s *runContainerState) setListValue(key string, index int, value string) {
	value = strings.TrimSpace(value)
	if value == "" || index < 0 || index >= len(s.lists[key]) {
		return
	}
	s.lists[key][index] = value
}

func runFieldByKey(key string) runField {
	for _, field := range runContainerFields {
		if field.Key == key {
			return field
		}
	}
	for _, group := range runContainerGroups {
		for _, field := range group.Fields {
			if field.Key == key {
				return field
			}
		}
	}

	return runField{}
}
