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
  since: '60m' # set to '' to show all logs
  tail: '' # set to 200 to show last 200 lines of logs
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

## Replacements

You can add replacements like so:

```yaml
replacements:
  imageNamePrefixes:
    '123456789012.dkr.ecr.us-east-1.amazonaws.com': '<prod>'
    '923456789999.dkr.ecr.us-east-1.amazonaws.com': '<dev>'
```
