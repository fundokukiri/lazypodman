package i18n

// TranslationSet is a set of localised strings for a given language
type TranslationSet struct {
	NotEnoughSpace                             string
	ProjectTitle                               string
	MainTitle                                  string
	GlobalTitle                                string
	Navigate                                   string
	Menu                                       string
	MenuTitle                                  string
	Execute                                    string
	Scroll                                     string
	Close                                      string
	Quit                                       string
	ErrorTitle                                 string
	NoViewMachingNewLineFocusedSwitchStatement string
	OpenConfig                                 string
	EditConfig                                 string
	ConfirmQuit                                string
	ConfirmUpProject                           string
	ErrorOccurred                              string
	ConnectionFailed                           string
	UnattachableContainerError                 string
	WaitingForContainerInfo                    string
	CannotAttachStoppedContainerError          string
	CannotAccessPodmanSocketError              string
	CannotKillChildError                       string

	Cancel                      string
	CustomCommandTitle          string
	BulkCommandTitle            string
	Remove                      string
	HideStopped                 string
	ForceRemove                 string
	RemoveWithVolumes           string
	MustForceToRemoveContainer  string
	Confirm                     string
	Return                      string
	FocusMain                   string
	LcFilter                    string
	StopContainer               string
	RestartingStatus            string
	StartingStatus              string
	StoppingStatus              string
	UppingProjectStatus         string
	UppingServiceStatus         string
	PausingStatus               string
	RemovingStatus              string
	DowningStatus               string
	RunningCustomCommandStatus  string
	RunningBulkCommandStatus    string
	RemoveService               string
	UpService                   string
	Stop                        string
	Pause                       string
	Restart                     string
	Down                        string
	DownWithVolumes             string
	Start                       string
	Rebuild                     string
	Recreate                    string
	PreviousContext             string
	NextContext                 string
	Attach                      string
	AttachCurrentTerminal       string
	AttachOpenedInTerminal      string
	NoExternalTerminal          string
	ViewLogs                    string
	UpProject                   string
	DownProject                 string
	ServicesTitle               string
	ContainersTitle             string
	StandaloneContainersTitle   string
	TopTitle                    string
	ImagesTitle                 string
	VolumesTitle                string
	NetworksTitle               string
	NoContainers                string
	NoContainer                 string
	NoImages                    string
	NoVolumes                   string
	NoNetworks                  string
	NoServices                  string
	RemoveImage                 string
	RemoveVolume                string
	RemoveNetwork               string
	RemoveWithoutPrune          string
	RemoveWithoutPruneWithForce string
	RemoveWithForce             string
	PruneImages                 string
	PruneContainers             string
	PruneVolumes                string
	PruneNetworks               string
	ConfirmPruneContainers      string
	ConfirmStopContainers       string
	ConfirmRemoveContainers     string
	ConfirmPruneImages          string
	ConfirmPruneVolumes         string
	ConfirmPruneNetworks        string
	PruningStatus               string
	StopService                 string
	PressEnterToReturn          string
	DetachFromContainerShortCut string
	StopAllContainers           string
	RemoveAllContainers         string
	ViewRestartOptions          string
	ExecShell                   string
	RunCustomCommand            string
	RunContainer                string
	RunContainerTitle           string
	RunContainerAdvanced        string
	RunContainerBack            string
	RunContainerAdd             string
	RunContainerEdit            string
	RunContainerDelete          string
	RunContainerEmpty           string
	RunContainerItems           string
	RunContainerRun             string
	RunContainerExtraArgs       string
	RunContainerConfirmTitle    string
	RunContainerConfirm         string
	RunContainerRunningStatus   string
	RunContainerOn              string
	RunContainerOff             string
	RunContainerImage           string
	RunContainerDefault         string
	RunContainerUnset           string
	RunContainerContainerID     string
	RunContainerAuto            string
	RunContainerLabels          map[string]string
	BuildImage                  string
	UpdateImage                 string
	UpdatingImageStatus         string
	UpdateImageNoTag            string
	ImageAlreadyUpToDate        string
	ImageUpdated                string
	BuildImageTitle             string
	BuildImageConfirmTitle      string
	BuildImageConfirm           string
	BuildImageRunningStatus     string
	BuildImageLabels            map[string]string
	CommitImage                 string
	CommitImageTitle            string
	CommitImageConfirmTitle     string
	CommitImageConfirm          string
	CommitImageRunningStatus    string
	CommitImageLabels           map[string]string
	ViewBulkCommands            string
	FilterList                  string
	OpenInBrowser               string
	SortContainersByState       string

	LogsTitle                   string
	ConfigTitle                 string
	EnvTitle                    string
	ComposeConfigTitle          string
	StatsTitle                  string
	CreditsTitle                string
	ContainerConfigTitle        string
	ContainerEnvTitle           string
	NothingToDisplay            string
	NoContainerForService       string
	CannotDisplayEnvVariables   string
	CannotManageNonLocalService string

	No  string
	Yes string

	LcNextScreenMode string
	LcPrevScreenMode string
	FilterPrompt     string

	FocusProjects   string
	FocusServices   string
	FocusPods       string
	FocusContainers string
	FocusImages     string
	FocusVolumes    string
	FocusNetworks   string

	PodsTitle                string
	PodsOff                  string
	NoPods                   string
	PodStatsTitle            string
	PodContainersTitle       string
	NoPodContainers          string
	RemovePod                string
	RemovePodForce           string
	StartPod                 string
	StopPod                  string
	RestartPod               string
	PausePod                 string
	UnpausePod               string
	ConfirmStartPod          string
	ConfirmStopPod           string
	ConfirmRestartPod        string
	ConfirmRemovePod         string
	ConfirmPausePod          string
	ConfirmUnpausePod        string
	PrunePods                string
	ConfirmPrunePods         string
	PodLogsOmittedContainers string

	HealthTitle                string
	HealthTimeHeader           string
	HealthExitHeader           string
	HealthOutputHeader         string
	HealthStatusLabel          string
	HealthFailingStreakLabel   string
	NoHealthcheck              string
	ContainerNotRunning        string
	RunningHealthcheckStatus   string
	RunHealthcheck             string
	HealthcheckHealthyResult   string
	HealthcheckUnhealthyResult string
	HealthRunOff               string
	CliOff                     string
	NoProbeOutput              string

	EventsTitle string

	PaletteTitle            string
	AutoUpdateContainers    string
	ConfirmAutoUpdate       string
	AutoUpdateHeader        string
	QuadletExport           string
	QuadletSavedStatus      string
	QuadletPlaintextWarning string
	QuadletNeedsDetails     string
	QuadletTitle            string

	ToggleMark             string
	ContainerShellTitle    string
	ShellTitle             string
	ShellHeader            string
	ShellHeaderPod         string
	AllLogs                string
	RecentLogs             string
	JumpToLatest           string
	ShellCloseHint         string
	ShellExited            string
	CloseShell             string
	LeaveShell             string
	MarkedCount            string
	MarksCleared           string
	StopContainersCount    string
	RestartContainersCount string
	PauseContainersCount   string
	RemoveContainersCount  string

	KubeMenuTitle        string
	KubeGenerateMenu     string
	KubePlayMenu         string
	KubeDownMenu         string
	KubeGenerateOff      string
	KubePlayOff          string
	KubeProbingStatus    string
	KubeGeneratingStatus string
	KubePlaintextWarning string
	KubeSavePrompt       string
	KubeOverwritePrompt  string
	KubeSavedStatus      string
	KubeNoFiles          string
	KubePlayConfirm      string
	KubeDownConfirm      string
	KubePlayHeader       string
	KubeDownHeader       string
}

func englishSet() TranslationSet {
	return TranslationSet{
		PruningStatus:              "pruning",
		RemovingStatus:             "removing",
		RestartingStatus:           "restarting",
		StartingStatus:             "starting",
		StoppingStatus:             "stopping",
		UppingServiceStatus:        "upping service",
		UppingProjectStatus:        "upping project",
		DowningStatus:              "downing",
		PausingStatus:              "pausing",
		RunningCustomCommandStatus: "running custom command",
		RunningBulkCommandStatus:   "running bulk command",

		NoViewMachingNewLineFocusedSwitchStatement: "No view matching newLineFocused switch statement",

		ErrorOccurred:                     "An error occurred! Please create an issue at https://github.com/fundokukiri/lazypodman/issues",
		ConnectionFailed:                  "connection to the podman service failed. Make sure it is running (e.g. `systemctl --user start podman.socket`) and that CONTAINER_HOST/PODMAN_HOST points at it",
		UnattachableContainerError:        "Container does not support attaching. You must either run the service with the '-it' flag or use `stdin_open: true, tty: true` in the podman-compose.yml file",
		WaitingForContainerInfo:           "Cannot proceed until podman gives us more information about the container. Please retry in a few moments.",
		CannotAttachStoppedContainerError: "You cannot attach to a stopped container, you need to start it first (which you can actually do with the 'r' key) (yes I'm too lazy to do this automatically for you) (pretty cool that I get to communicate one-on-one with you in the form of an error message though)",
		CannotAccessPodmanSocketError:     "Can't reach the podman service socket. Lazypodman looked at $XDG_RUNTIME_DIR/podman/podman.sock, then /run/podman/podman.sock.\nStart the service with `systemctl --user start podman.socket` (rootless) or `systemctl start podman.socket` (rootful), or set CONTAINER_HOST to your podman endpoint.",
		CannotKillChildError:              "Waited three seconds for child process to stop. There may be an orphan process that continues to run on your system.",

		Confirm: "Confirm",

		Return:                      "return",
		FocusMain:                   "focus main panel",
		LcFilter:                    "filter list",
		Navigate:                    "navigate",
		Execute:                     "execute",
		Close:                       "close",
		Quit:                        "quit",
		Menu:                        "menu",
		MenuTitle:                   "Menu",
		Scroll:                      "scroll",
		OpenConfig:                  "open lazypodman config",
		EditConfig:                  "edit lazypodman config",
		Cancel:                      "cancel",
		Remove:                      "remove",
		HideStopped:                 "hide/show stopped containers",
		ForceRemove:                 "force remove",
		RemoveWithVolumes:           "remove with volumes",
		RemoveService:               "remove containers",
		UpService:                   "up service",
		Stop:                        "stop",
		Pause:                       "pause",
		Restart:                     "restart",
		Down:                        "down project",
		DownWithVolumes:             "down project with volumes",
		Start:                       "start",
		Rebuild:                     "rebuild",
		Recreate:                    "recreate",
		PreviousContext:             "previous tab",
		NextContext:                 "next tab",
		Attach:                      "attach",
		AttachCurrentTerminal:       "attach in this terminal",
		AttachOpenedInTerminal:      "attach opened in a separate terminal",
		NoExternalTerminal:          "no terminal emulator found; set the TERMINAL environment variable or install xterm, kitty, alacritty, wezterm, foot, konsole or gnome-terminal",
		ViewLogs:                    "view logs",
		UpProject:                   "up project",
		DownProject:                 "down project",
		RemoveImage:                 "remove image",
		RemoveVolume:                "remove volume",
		RemoveNetwork:               "remove network",
		RemoveWithoutPrune:          "remove without deleting untagged parents",
		RemoveWithoutPruneWithForce: "remove (forced) without deleting untagged parents",
		RemoveWithForce:             "remove (forced)",
		PruneContainers:             "prune exited containers",
		PruneVolumes:                "prune unused volumes",
		PruneNetworks:               "prune unused networks",
		PruneImages:                 "prune unused images",
		StopAllContainers:           "stop all containers",
		RemoveAllContainers:         "remove all containers (forced)",
		ViewRestartOptions:          "view restart options",
		ExecShell:                   "exec shell",
		RunCustomCommand:            "run predefined custom command",
		RunContainer:                "run container",
		RunContainerTitle:           "Run container",
		RunContainerAdvanced:        "Advanced",
		RunContainerBack:            "back",
		RunContainerAdd:             "add",
		RunContainerEdit:            "edit",
		RunContainerDelete:          "delete",
		RunContainerEmpty:           "-",
		RunContainerItems:           "items",
		RunContainerRun:             "Run",
		RunContainerExtraArgs:       "Extra args",
		RunContainerConfirmTitle:    "Confirm",
		RunContainerConfirm:         "Run this container?",
		RunContainerRunningStatus:   "Starting container",
		RunContainerOn:              "on",
		RunContainerOff:             "off",
		RunContainerImage:           "Image",
		RunContainerDefault:         "default",
		RunContainerUnset:           "-",
		RunContainerContainerID:     "container ID",
		RunContainerAuto:            "auto",
		ViewBulkCommands:            "view bulk commands",
		FilterList:                  "filter list",
		OpenInBrowser:               "open in browser (first port is http)",
		SortContainersByState:       "sort containers by state",

		GlobalTitle:                 "Global",
		MainTitle:                   "Main",
		ProjectTitle:                "Project",
		ServicesTitle:               "Services",
		ContainersTitle:             "Containers",
		StandaloneContainersTitle:   "Standalone Containers",
		ImagesTitle:                 "Images",
		VolumesTitle:                "Volumes",
		NetworksTitle:               "Networks",
		CustomCommandTitle:          "Custom Command:",
		BulkCommandTitle:            "Bulk Command:",
		ErrorTitle:                  "Error",
		LogsTitle:                   "Logs",
		ConfigTitle:                 "Config",
		EnvTitle:                    "Env",
		ComposeConfigTitle:          "Compose Config",
		TopTitle:                    "Top",
		StatsTitle:                  "Stats",
		CreditsTitle:                "About",
		ContainerConfigTitle:        "Container Config",
		ContainerEnvTitle:           "Container Env",
		NothingToDisplay:            "Nothing to display",
		NoContainerForService:       "No logs to show; service is not associated with a container",
		CannotDisplayEnvVariables:   "Something went wrong while displaying environment variables",
		CannotManageNonLocalService: "This service belongs to a different compose project. Run lazypodman from that project's directory to manage it.",

		NoContainers: "No containers",
		NoContainer:  "No container",
		NoImages:     "No images",
		NoVolumes:    "No volumes",
		NoNetworks:   "No networks",
		NoServices:   "No services",

		ConfirmQuit:                 "Are you sure you want to quit?",
		ConfirmUpProject:            "Are you sure you want to 'up' your podman compose project?",
		MustForceToRemoveContainer:  "You cannot remove a running container unless you force it. Do you want to force it?",
		NotEnoughSpace:              "Not enough space to render panels",
		ConfirmPruneImages:          "Are you sure you want to prune all unused images?",
		ConfirmPruneContainers:      "Are you sure you want to prune all stopped containers?",
		ConfirmStopContainers:       "Are you sure you want to stop all containers?",
		ConfirmRemoveContainers:     "Are you sure you want to remove all containers?",
		ConfirmPruneVolumes:         "Are you sure you want to prune all unused volumes?",
		ConfirmPruneNetworks:        "Are you sure you want to prune all unused networks?",
		StopService:                 "Are you sure you want to stop this service's containers?",
		StopContainer:               "Are you sure you want to stop this container?",
		PressEnterToReturn:          "Press enter to return to lazypodman (this prompt can be disabled in your config by setting `gui.returnImmediately: true`)",
		DetachFromContainerShortCut: "By default, to detach from the container press ctrl-p then ctrl-q",

		No:  "no",
		Yes: "yes",

		LcNextScreenMode: "next screen mode (normal/half/fullscreen)",
		LcPrevScreenMode: "prev screen mode",
		FilterPrompt:     "filter",

		FocusProjects:   "focus projects panel",
		FocusServices:   "focus services panel",
		FocusPods:       "focus pods panel",
		FocusContainers: "focus containers panel",
		FocusImages:     "focus images panel",
		FocusVolumes:    "focus volumes panel",
		FocusNetworks:   "focus networks panel",

		PodsTitle:                "Pods",
		PodsOff:                  "pods off:",
		NoPods:                   "No pods",
		PodStatsTitle:            "Stats (sum of containers)",
		PodContainersTitle:       "Containers",
		NoPodContainers:          "This pod has no containers",
		RemovePod:                "Remove",
		RemovePodForce:           "Remove (force)",
		StartPod:                 "Start",
		StopPod:                  "Stop",
		RestartPod:               "Restart",
		PausePod:                 "Pause",
		UnpausePod:               "Unpause",
		ConfirmStartPod:          "Start pod?",
		ConfirmStopPod:           "Stop pod?",
		ConfirmRestartPod:        "Restart pod?",
		ConfirmRemovePod:         "Remove pod? Its containers are removed too.",
		ConfirmPausePod:          "Pause pod?",
		ConfirmUnpausePod:        "Unpause pod?",
		PrunePods:                "Prune stopped pods",
		ConfirmPrunePods:         "Prune all stopped pods?",
		PodLogsOmittedContainers: "only the first containers are streamed",

		HealthTitle:                "Health",
		HealthTimeHeader:           "Time",
		HealthExitHeader:           "Exit",
		HealthOutputHeader:         "Output",
		HealthStatusLabel:          "Status",
		HealthFailingStreakLabel:   "Failing streak",
		NoHealthcheck:              "This container has no healthcheck",
		ContainerNotRunning:        "Container is not running",
		RunningHealthcheckStatus:   "Running healthcheck",
		RunHealthcheck:             "Run healthcheck (recorded in the podman health log)",
		HealthcheckHealthyResult:   "healthcheck: healthy",
		HealthcheckUnhealthyResult: "healthcheck: unhealthy (exit %d)",
		HealthRunOff:               "health run off:",
		CliOff:                     "cli off:",
		NoProbeOutput:              "(no output)",
		EventsTitle:                "Events",
		PaletteTitle:               "Command palette",
		AutoUpdateContainers:       "Auto-update containers (podman auto-update)",
		ConfirmAutoUpdate:          "Run podman auto-update?",
		AutoUpdateHeader:           "# podman auto-update",
		QuadletExport:              "Export Quadlet .container file",
		QuadletSavedStatus:         "Saved %s",
		QuadletPlaintextWarning:    "This unit contains environment variables in plaintext.",
		QuadletNeedsDetails:        "Container details are not loaded yet",
		QuadletTitle:               "Quadlet",
		ToggleMark:                 "mark/unmark container",
		ContainerShellTitle:        "container shell",
		ShellTitle:                 "shell",
		ShellHeader:                "container: %s · image: %s · status: %s · id: %s",
		ShellHeaderPod:             " · pod: %s",
		AllLogs:                    "load all logs",
		RecentLogs:                 "load recent logs",
		JumpToLatest:               "jump to latest logs",
		ShellCloseHint:             "shell open: ctrl+x closes it",
		ShellExited:                "shell exited",
		CloseShell:                 "close shell",
		LeaveShell:                 "leave shell, keep it running",
		MarkedCount:                "%d selected",
		MarksCleared:               "selection cleared",
		StopContainersCount:        "Stop %d containers?",
		RestartContainersCount:     "Restart %d containers?",
		PauseContainersCount:       "Pause/unpause %d containers?",
		RemoveContainersCount:      "Remove %d containers?",

		KubeMenuTitle:        "Kube",
		KubeGenerateMenu:     "Generate kube YAML for this pod",
		KubePlayMenu:         "Play kube YAML…",
		KubeDownMenu:         "Down kube YAML…",
		KubeGenerateOff:      "kube generate off:",
		KubePlayOff:          "kube play off:",
		KubeProbingStatus:    "Checking kube support",
		KubeGeneratingStatus: "Generating kube YAML",
		KubePlaintextWarning: "This file contains environment variables in plaintext; no Secrets are created.",
		KubeSavePrompt:       "Save to %s?",
		KubeOverwritePrompt:  "File exists: %s Overwrite it?",
		KubeSavedStatus:      "Saved %s",
		KubeNoFiles:          "No kube YAML files in %s",
		KubePlayConfirm:      "Play the pods defined in:\n%s",
		KubeDownConfirm:      "Stop and remove the pods defined in:\n%s",
		KubePlayHeader:       "# playing %s",
		KubeDownHeader:       "# down %s",

		BuildImage:               "build image from Containerfile",
		BuildImageTitle:          "Build image",
		BuildImageConfirmTitle:   "Confirm build",
		BuildImageConfirm:        "Build this image?",
		BuildImageRunningStatus:  "Building image",
		UpdateImage:              "update image",
		UpdatingImageStatus:      "updating image",
		UpdateImageNoTag:         "This image has no repository tag to pull",
		ImageAlreadyUpToDate:     "image already up to date: %s",
		ImageUpdated:             "image updated: %s",
		CommitImage:              "commit to image",
		CommitImageTitle:         "Commit container",
		CommitImageConfirmTitle:  "Confirm commit",
		CommitImageConfirm:       "Commit this container to an image?",
		CommitImageRunningStatus: "Committing container",
		CommitImageLabels: map[string]string{
			"image": "Image", "author": "Author", "message": "Message",
			"changes": "Instructions", "commit_options_group": "Options",
			"config": "Config file", "format": "Format", "iidfile": "Image ID file",
			"include_volumes": "Include volumes", "quiet": "Quiet", "squash": "Squash layers",
		},
		BuildImageLabels: map[string]string{
			"context": "Context", "containerfile": "Containerfile", "tags": "Tags",
			"build_options_group": "Build options", "build_args_group": "Build args & labels",
			"build_network_group": "Network & DNS", "build_resources_group": "Resources",
			"build_security_group": "Security", "build_output_group": "Output & cache",
			"build_auth_group": "Auth & secrets",
			"format":           "Format", "target": "Target stage", "platform": "Platform",
			"jobs": "Jobs", "no_cache": "No cache", "all_platforms": "All platforms",
			"squash": "Squash layers", "quiet": "Quiet",
			"build_arg": "Build arg", "build_arg_file": "Build arg file",
			"build_env": "Build env", "build_label": "Build label",
			"build_annotation": "Build annotation", "unsetenv": "Unset env",
			"unsetlabel": "Unset label", "unsetannotation": "Unset annotation",
			"build_network": "Build network", "add_host": "Add host", "dns": "DNS",
			"dns_search": "DNS search", "dns_option": "DNS option",
			"cpuset_cpus": "CPU set", "memory": "Memory", "memory_swap": "Memory swap",
			"shm_size": "SHM size", "cpu_shares": "CPU shares",
			"cgroup_parent": "Cgroup parent", "ulimit": "Ulimit", "isolation": "Isolation",
			"security_opt": "Security opt", "cap_add": "Cap add", "cap_drop": "Cap drop",
			"userns": "User namespace", "userns_uid_map": "User namespace UID map",
			"userns_gid_map": "User namespace GID map", "group_add": "Group add",
			"device": "Device", "from": "Replace FROM", "output": "Output",
			"iidfile": "Image ID file", "metadata_file": "Metadata file",
			"manifest": "Manifest", "cache_from": "Cache from", "cache_to": "Cache to",
			"cache_ttl": "Cache TTL", "timestamp": "Timestamp",
			"source_date_epoch": "Source date epoch", "ignorefile": "Ignore file",
			"authfile": "Auth file", "cert_dir": "Cert dir", "creds": "Credentials",
			"secret": "Secret", "ssh": "SSH", "build_volume": "Build volume",
			"build_mount": "Build mount",
		},
		RunContainerLabels: map[string]string{
			"image": "Image",
			"name":  "Name", "command": "Command", "entrypoint": "Entrypoint",
			"workdir": "Working dir", "user": "User", "hostname": "Hostname",
			"ports": "Ports", "volumes": "Volumes", "env": "Env", "labels": "Labels",
			"network": "Network", "restart": "Restart", "detach": "Detach",
			"rm": "Auto-remove", "interactive": "Interactive", "tty": "TTY",
			"network_group": "Network & DNS", "storage_group": "Storage",
			"resources_group": "Resources", "security_group": "Security",
			"health_group": "Healthcheck", "runtime_group": "Runtime & image",
			"environment_group": "Environment & labels",
			"publish_all":       "Publish all ports", "expose": "Expose", "network_alias": "Network alias",
			"ip": "Static IPv4", "ip6": "Static IPv6", "mac": "MAC address", "dns": "DNS",
			"dns_search": "DNS search", "dns_option": "DNS option", "add_host": "Add host",
			"hosts_file": "Hosts file", "ipc": "IPC namespace", "pid": "PID namespace",
			"uts": "UTS namespace", "cgroupns": "Cgroup namespace", "pod": "Pod",
			"userns": "User namespace", "uidmap": "UID map", "gidmap": "GID map",
			"subuidname": "Subuid name", "subgidname": "Subgid name",
			"mount": "Mount", "tmpfs": "Tmpfs", "volumes_from": "Volumes from",
			"read_only": "Read-only rootfs", "image_volume": "Image volume",
			"chrootdirs": "Chroot dirs", "cpus": "CPUs", "cpuset_cpus": "CPU set",
			"cpu_shares": "CPU shares", "cpu_period": "CPU period", "cpu_quota": "CPU quota",
			"memory": "Memory", "memory_reservation": "Memory reservation",
			"memory_swap": "Memory swap", "memory_swappiness": "Memory swappiness",
			"shm_size": "SHM size", "pids_limit": "PIDs limit", "ulimit": "Ulimit",
			"blkio_weight": "Block IO weight", "device_read_bps": "Device read BPS",
			"device_write_bps": "Device write BPS", "device_read_iops": "Device read IOPS",
			"device_write_iops": "Device write IOPS", "oom_score_adj": "OOM score adj",
			"oom_kill_disable": "Disable OOM killer", "privileged": "Privileged",
			"cap_add": "Cap add", "cap_drop": "Cap drop", "security_opt": "Security opt",
			"seccomp_policy": "Seccomp policy", "device": "Device",
			"device_cgroup_rule": "Device cgroup rule", "group_add": "Group add",
			"gpus": "GPUs", "no_new_privileges": "No new privileges",
			"health_cmd": "Health command", "health_interval": "Health interval",
			"health_timeout": "Health timeout", "health_retries": "Health retries",
			"health_start_period": "Health start period", "health_start_interval": "Health start interval",
			"health_startup_cmd": "Startup health command", "health_startup_interval": "Startup health interval",
			"health_startup_retries": "Startup health retries", "health_startup_success": "Startup health successes",
			"health_on_failure": "Health on failure", "health_log_destination": "Health log destination",
			"health_max_log_count": "Health log max count", "health_max_log_size": "Health log max size",
			"no_healthcheck": "Disable healthcheck", "init": "Init", "systemd": "Systemd",
			"sdnotify": "Sdnotify", "stop_signal": "Stop signal", "stop_timeout": "Stop timeout",
			"timeout": "Run timeout", "log_driver": "Log driver", "log_opt": "Log option",
			"pull": "Pull policy", "retry": "Pull retries", "retry_delay": "Pull retry delay",
			"platform": "Platform", "arch": "Architecture", "os": "OS", "variant": "Variant",
			"cidfile": "CID file", "pidfile": "PID file", "conmon_pidfile": "Conmon PID file",
			"replace": "Replace same name", "requires": "Requires", "tz": "Timezone",
			"umask": "Umask", "detach_keys": "Detach keys", "attach": "Attach",
			"sig_proxy": "Signal proxy", "http_proxy": "HTTP proxy", "tls_verify": "TLS verify",
			"authfile": "Auth file", "creds": "Credentials", "cert_dir": "Cert dir",
			"env_file": "Env file", "env_host": "Host env", "env_merge": "Env merge",
			"unsetenv": "Unset env", "unsetenv_all": "Unset all env",
			"label_file": "Label file", "annotation": "Annotation",
			"preserve_fds": "Preserve FDs", "preserve_fd": "Preserve FD",
			"personality": "Personality",
		},
	}
}
