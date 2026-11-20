# User Config:

## Opening The User Config

The location of the user config will differ depending on your OS. You can open it via lazypodman by opening the application, clicking on the 'project' panel at the top left and pressing 'o' (or pressing 'e' if your files open in vim).

Changes to the user config will only take place after closing and re-opening lazypodman

### Locations:

- OSX: `~/Library/Application Support/lazypodman/config.yml`
- Linux: `~/.config/lazypodman/config.yml`
- Windows: `C:\Users\<User>\AppData\Roaming\lazypodman\config.yml`

JSON schema is available for `config.yml` so that IntelliSense in Visual Studio Code
(completion and error checking) is automatically enabled when the [YAML Red Hat][yaml]
extension is installed. However, note that automatic schema detection only works
if your config file is in one of the standard paths mentioned above. If you
override the path to the file, you can still make IntelliSense work by adding

```yaml
# yaml-language-server: $schema=https://json.schemastore.org/lazypodman.json
```

to the top of your config file or via [Visual Studio Code settings.json config][settings].

[yaml]: https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml
[settings]: https://github.com/redhat-developer/vscode-yaml#associating-a-schema-to-a-glob-pattern-via-yamlschemas

## Default:

```yml
gui:
  scrollHeight: 2
  language: "auto" # one of 'auto' | 'en' | 'pl' | 'nl' | 'de' | 'tr'
  border: "rounded" # one of 'rounded' | 'single' | 'double' | 'hidden'
  theme:
    activeBorderColor:
      - green
      - bold
    inactiveBorderColor:
      - white
    selectedLineBgColor:
      - blue
    optionsTextColor:
      - blue
  returnImmediately: false
  wrapMainPanel: true
  # Side panel width as a ratio of the screen's width
  sidePanelWidth: 0.333
  # Determines whether we show the bottom line (the one containing keybinding
  # info and the status of the app).
  showBottomLine: true
  # When true, increases vertical space used by focused side panel,
  # creating an accordion effect
  expandFocusedSidePanel: false
  # Determines which screen mode will be used on startup
  screenMode: "normal" # one of 'normal' | 'half' | 'fullscreen'
  # Determines the style of the container status and container health display in the
  # containers panel. "long": full words (default), "short": one or two characters,
  # "icon": unicode emoji.
  containerStatusHealthStyle: "long"
logs:
  timestamps: false
  since: '60m' # set to '' to show all logs; ignored for stopped containers
  # tail limits how much history is fetched when a log tab opens (empty = all).
  # Default is '20'; press ctrl+l in the Logs tab to load the whole history.
  tail: '20'
  # maxLineLength splits one logical line into display segments. Default 4096
  # prevents a single horizontal line from making a small tail slow to render.
  # 0 = no split.
  maxLineLength: 4096
  # maxBytes bounds the rendered text kept in the Logs tab (newest lines win).
  # Default 1048576 (1 MiB); 0 = only the 10000-line cap applies.
  maxBytes: 1048576
commandTemplates:
  podman: podman # The podman executable used for attach/exec (default: podman)
  podmanCompose: podman compose # Determines the compose command to run, referred to as .PodmanCompose in commandTemplates
  restartService: '{{ .PodmanCompose }} restart {{ .Service.Name }}'
  up:  '{{ .PodmanCompose }} up -d'
  down: '{{ .PodmanCompose }} down'
  downWithVolumes: '{{ .PodmanCompose }} down --volumes'
  upService:  '{{ .PodmanCompose }} up -d {{ .Service.Name }}'
  startService: '{{ .PodmanCompose }} start {{ .Service.Name }}'
  stopService: '{{ .PodmanCompose }} stop {{ .Service.Name }}'
  serviceLogs: '{{ .PodmanCompose }} logs --since=60m --follow {{ .Service.Name }}'
  viewServiceLogs: '{{ .PodmanCompose }} logs --follow {{ .Service.Name }}'
  rebuildService: '{{ .PodmanCompose }} up -d --build {{ .Service.Name }}'
  recreateService: '{{ .PodmanCompose }} up -d --force-recreate {{ .Service.Name }}'
  allLogs: '{{ .PodmanCompose }} logs --tail=300 --follow'
  viewAlLogs: '{{ .PodmanCompose }} logs'
  composeConfig: '{{ .PodmanCompose }} config'
  checkComposeConfig: '{{ .PodmanCompose }} config --quiet'
  serviceTop: '{{ .PodmanCompose }} top {{ .Service.Name }}'
oS:
  openCommand: open {{filename}}
  openLinkCommand: open {{link}}
stats:
  graphs:
    - caption: CPU (%)
      statPath: DerivedStats.CPUPercentage
      color: blue
    - caption: Memory (%)
      statPath: DerivedStats.MemoryPercentage
      color: green
```

## To see what all of the config options mean, and what other options you can set, see [here](https://godoc.org/github.com/fundokukiri/lazypodman/pkg/config)

## Color Attributes:

For color attributes you can choose an array of attributes (with max one color attribute)
The available attributes are:

- default
- black
- red
- green
- yellow
- blue
- magenta
- cyan
- white
- bold
- reverse # useful for high-contrast
- underline

## Pods

The pods panel is detected automatically: it is shown when the `podman` CLI exists,
is at least version 3.0 and talks to the same store as the socket lazypodman uses
(Linux, same host). The Docker-compatible API has no pod endpoint, so pod data
comes from the CLI.

```yaml
gui:
  showPodColumn: true   # pod column in the containers panel (default true)
  pods:
    enabled: "auto"     # auto | always | never
```

- `auto` (default): run every check, disable pods when one fails.
- `always`: skip the CLI/socket store comparison. Useful when the check is slow
  on a rootless start; pods are then assumed to live in the same store.
- `never`: never enable pods, never run the CLI.

When pods are off the panel is hidden, the containers panel loses its pod column
and the bottom-right corner shows `pods off: <reason>`. Nothing else changes:
containers, services, images, volumes and networks keep working through the socket.
Pods custom commands and bulk commands live under `customCommands.pods` and
`bulkCommands.pods`, and can use `{{ .Pod.Name }}` / `{{ .Pod.ID }}`.

## Custom Commands

You can add custom commands like so:

```yaml
customCommands:
  containers:
    - name: bash
      attach: true
      command: '{{ .Podman }} exec -it {{ .Container.ID }} bash'
      serviceNames: []
```

You may use the following go templates (such as `{{ .Container.ID }}` above) in your commands:
- `{{ .Podman }}`: the podman binary (default: `podman`)
- `{{ .PodmanCompose }}`: the compose command (default: `podman compose`, falling back to `podman-compose`) 
- [`{{ .Container }}`](https://pkg.go.dev/github.com/fundokukiri/lazypodman@v0.20.0/pkg/commands#Container) and its fields. For example: `{{ .Container.Container.ImageID }}`
- [`{{ .Service }}`](https://pkg.go.dev/github.com/fundokukiri/lazypodman@v0.20.0/pkg/commands#Service) and its fields. For example: `{{ .Service.Name }}`


## Kube

The kube menu (`g` on the pods panel) generates a manifest from a pod and plays or
downs manifests from one directory.

```yaml
gui:
  kubeDir: ""   # default: <user config dir>/lazypodman/kube
```

`~` is expanded. The directory is both the save target for generate and the source
for play/down.

- **Generate** writes `<pod-name>.yaml` with mode `0600` (directory `0755`), and
  only after a confirmation showing the full path. If the file exists the prompt
  says so; cancelling leaves it untouched. The generated YAML contains environment
  variables in plaintext and no Secrets, which the prompt warns about.
- **Play / Down** list `*.yaml` and `*.yml` files (non-recursive) whose first 20
  lines contain a top-level `kind:`. They never run automatically; both confirm
  with the full path, and down says the pods will be stopped and removed. Output
  streams into the main panel and a cancelled task kills the podman process.
- The command syntax is probed lazily: `podman kube generate|play` first, then the
  older `podman generate kube` / `podman play kube`. Nothing is assumed about the
  version; when neither works the corner shows `kube generate off:` / `kube play off:`
  only after a probe has actually failed. `--replace` and `--build` are never
  passed in this version.

## Healthcheck

Every container panel has a **Health** tab, after **Stats**. It reads the health
state and probe history from the container inspect data the socket already
provides, so opening it starts no process and makes no extra call.

`t` (containers panel) runs `podman healthcheck run` once for the selected
container. Two conditions are checked before any process starts: the container
must be running and must have a healthcheck (`--health-cmd`).

- A healthy run shows `healthcheck: healthy` on the status line.
- An unhealthy run shows `healthcheck: unhealthy (exit 1)`. That is a **result**,
  not an error, so no error panel appears.
- A run that cannot execute, or an exit code outside the documented table, opens
  the error panel with Podman's output verbatim.

The documented exit codes are `0` healthy, `1` unhealthy and `2` reserved
(unhealthy). Only `0` is verified here; `1` and `2` come from Podman's
documentation, and any other code is treated as an error rather than guessed at.

**Side effect**: a manual run is recorded by Podman. It appends to the health log,
can change the failing streak and the reported status, and can trigger
`--health-on-failure` behaviour if the container was started with it.

Running the probe needs the CLI, so the first `t` press probes
`podman healthcheck run --help` once. A missing subcommand is cached for the
session (and the corner then shows `health run off: <reason>`); a timeout is
retried on the next press. Before a probe has failed nothing is claimed, which is
why no off line appears on startup.

## Replacements

You can add replacements like so:

```yaml
replacements:
  imageNamePrefixes:
    '123456789012.dkr.ecr.us-east-1.amazonaws.com': '<prod>'
    '923456789999.dkr.ecr.us-east-1.amazonaws.com': '<dev>'
```

## Quadlet

`Q` on the containers panel exports the selected container as a Quadlet
`.container` unit. The unit is built from container inspect data, shown as a
preview first, and written as `0600` only after a confirmation naming the full
path.

```yaml
gui:
  quadletDir: ""   # default: <user config dir>/containers/systemd
```

`~` is expanded. The default is the directory Podman reads user units from, so
`systemctl --user daemon-reload` then `systemctl --user start <name>` can pick the
file up. lazypodman itself never calls `systemctl`.

The generated file contains `Image=`, `ContainerName=`, `Environment=` (values are
plaintext, the prompt warns about it; each value is quoted and newlines are
flattened), `Exec=`, `PublishPort=`, `Volume=` (named volumes keep their name),
`Network=` plus `[Service]` and `[Install]` sections. It is a readable starting
point, not a full translation of every inspect field.

## Auto-update

The Containers bulk menu (`b`) has an `Auto-update containers` entry that runs
`podman auto-update` after a confirmation and streams the output into the main
panel. It acts on every container that carries the auto-update label, not on the
current selection or marks. There is no config key; remove the entry with
`bulkCommands.containers` overrides if you do not want it.
