# Lazypodman

**The lazier way to manage everything podman.**

A terminal UI for Podman: containers, images, volumes, networks, compose
projects, logs and stats in one window, every common action one keypress away.

---

## Features

- View the state of your Podman containers and compose services at a glance
- View and follow logs for a container or a service
- ASCII graphs of container CPU/memory, or of any stat path you configure
- Attach to a container, exec a shell, view its processes (`top`), config and env
- Start/stop/restart/pause/remove/rebuild containers and services
- Inspect image layers, prune containers, images, volumes and networks
- Custom commands and bulk commands, configured per panel
- Mouse support, filterable lists, 8 UI languages

---

## Requirements

| Component | Version / note |
| --- | --- |
| Podman | >= **4.0** with the service API (socket) running |
| Compose (optional) | `podman compose` (Podman >= 4.1) or `podman-compose` |
| Go (to build) | >= **1.22** |

Lazypodman speaks the Docker-compatible API that Podman serves, so enable the
socket once:

```sh
systemctl --user enable --now podman.socket   # rootless (most common)
sudo systemctl enable --now podman.socket     # rootful
```

On macOS and Windows start the Podman machine first:

```sh
podman machine start
podman system connection list   # shows the endpoint for CONTAINER_HOST
```

---

## Installation

### Build from source

Dependencies are vendored, so no network access is needed after cloning:

```sh
git clone https://github.com/fundokukiri/lazypodman.git
cd lazypodman
GOFLAGS=-mod=vendor go build -o lazypodman .
./lazypodman
```

Install into `$GOBIN`/`$GOPATH/bin`:

```sh
GOFLAGS=-mod=vendor go install -mod=vendor
```

### Container image

```sh
podman build -t lazypodman .

podman run --rm -it \
  --security-opt label=disable \
  -v /run/podman/podman.sock:/run/podman/podman.sock \
  -e CONTAINER_HOST=unix:///run/podman/podman.sock \
  -v ./config:/root/.config/lazypodman \
  lazypodman
```

The included `docker-compose.yml` sets up the same thing:

```sh
podman compose run --rm lazypodman   # or: podman-compose run --rm lazypodman
```

### Alias

```sh
echo "alias lzp='lazypodman'" >> ~/.zshrc   # or ~/.bashrc
```

---

## Finding the Podman service

In order of decreasing precedence:

| # | Source | Example |
| --- | --- | --- |
| 1 | `PODMAN_HOST` (lazypodman-specific) | `unix:///run/user/1000/podman/podman.sock` |
| 2 | `CONTAINER_HOST` (Podman CLI variable) | `tcp://127.0.0.1:8888` |
| 3 | `DOCKER_HOST` (still honored) | `unix:///run/podman/podman.sock` |
| 4 | a running rootless socket | `$XDG_RUNTIME_DIR/podman/podman.sock`, `/run/user/<uid>/podman/podman.sock` |
| 5 | a running rootful socket | `/run/podman/podman.sock` |
| 6 | platform default | Windows: `npipe:////./pipe/podman-machine-default` |

- A bare path is accepted and turned into a URL:
  `CONTAINER_HOST=/run/user/1000/podman/podman.sock`.
- `ssh://` endpoints are tunneled to a local socket automatically; the remote
  socket is taken from the URL (default `/run/podman/podman.sock`):
  ```sh
  CONTAINER_HOST=ssh://user@host/run/user/1000/podman/podman.sock lazypodman
  ```
- Docker contexts (`~/.docker/contexts`) are **not** used.

---

## Usage

Run it in a directory that contains a compose file to get the projects and
services panels:

```sh
cd /path/to/compose/project
lazypodman
```

Flags:

| Flag | Meaning |
| --- | --- |
| `-f, --file` | alternate compose files (repeatable) |
| `-p, --project` | compose project name |
| `-c, --config` | print the merged config |
| `-d, --debug` | debug logging |
| `--version`, `-h` | version and help |

Layout: the left column holds **Project → Services → Containers → Images →
Volumes → Networks**; the main panel shows the selected item's **Logs / Stats /
Config / Env / Top / Compose Config / About** tabs.

### Navigation

| Key | Action |
| --- | --- |
| `1`…`6` | focus Projects / Services / Containers / Images / Volumes / Networks |
| `tab` / `shift+tab` | next / previous panel (or `l` / `h`) |
| `↑` `↓` / `k` `j` | move in the list |
| `enter` | focus the main panel |
| `[` / `]` | previous / next tab |
| `/` | filter the list |
| `esc` | close panel / go back |
| `q` | quit |
| `+` / `_` | screen mode (normal / half / fullscreen) |
| `o` / `e` | open / edit the config (Projects panel) |

### Panel keys

**Containers**: `d` remove · `e` show/hide stopped · `p` pause · `s` stop ·
`r` restart · `a` attach · `m` logs · `E` exec shell · `c` custom command ·
`b` bulk commands · `w` open first published port in the browser.

**Services**: `u` up service · `U` up project · `D` down project · `s`/`S`
stop/start · `r`/`R` restart / restart menu · `p` pause · `a` attach · `m` logs ·
`E` exec shell · `c`/`b` custom / bulk command · `w` open port.

**Images, Volumes, Networks**: `d` remove · `c` custom command · `b` bulk
commands (prune).

The full generated list for every language lives in
[docs/keybindings](/docs/keybindings) and can be regenerated with
`GOFLAGS=-mod=vendor go run ./scripts/cheatsheet generate`.

### Typical workflows

1. **Debug a failing service**: select it, `m` for logs, `]` for the Stats tab,
   `E` to open a shell.
2. **Redeploy after a code change**: select the service, `R`, pick
   recreate/rebuild (driven by `commandTemplates.recreateService` /
   `rebuildService`).
3. **Reclaim disk space**: Images / Volumes / Networks panel, `b`, prune.
4. **Check the rendered compose file**: select the project, open the
   **Compose Config** tab.

---

## Configuration

| OS | Config file |
| --- | --- |
| Linux | `~/.config/lazypodman/config.yml` |
| macOS | `~/Library/Application Support/lazypodman/config.yml` (older installs: `.../jesseduffield/...`) |
| Windows | `%AppData%\\lazypodman\\config.yml` |

- Override the directory with `CONFIG_DIR`.
- Open it from the app: Projects panel, `o` (open) or `e` (edit).
- `lazypodman -c` prints the config after merging with the defaults.
- Changes take effect after a restart. Full reference:
  [docs/Config.md](/docs/Config.md).

Key groups:

```yaml
gui:
  language: "auto"            # auto | en | de | es | fr | nl | pl | pt | tr | zh
  border: "rounded"           # rounded | single | double | hidden
  theme:
    activeBorderColor: [green, bold]
    inactiveBorderColor: [default]
    selectedLineBgColor: [blue]
    optionsTextColor: [blue]
  sidePanelWidth: 0.3333
  showBottomLine: true
  wrapMainPanel: true
  returnImmediately: false
  screenMode: "normal"        # normal | half | fullscreen
  containerStatusHealthStyle: "long"   # long | short | icon

logs:
  timestamps: false
  since: '60m'                # '' = all logs
  tail: ''                    # e.g. 200 = last 200 lines

commandTemplates:
  podman: podman               # CLI used for attach/exec (e.g. podman-remote)
  podmanCompose: podman compose # or "podman-compose", or extra -f flags
  up: '{{ .PodmanCompose }} up -d'
  down: '{{ .PodmanCompose }} down'
  upService: '{{ .PodmanCompose }} up -d {{ .Service.Name }}'
  stopService: '{{ .PodmanCompose }} stop {{ .Service.Name }}'
  restartService: '{{ .PodmanCompose }} restart {{ .Service.Name }}'
  rebuildService: '{{ .PodmanCompose }} up -d --build {{ .Service.Name }}'
  recreateService: '{{ .PodmanCompose }} up -d --force-recreate {{ .Service.Name }}'
  serviceLogs: '{{ .PodmanCompose }} logs --since=60m --follow {{ .Service.Name }}'
  allLogs: '{{ .PodmanCompose }} logs --tail=300 --follow'
  composeConfig: '{{ .PodmanCompose }} config'
  checkComposeConfig: '{{ .PodmanCompose }} config --quiet'
  serviceTop: '{{ .PodmanCompose }} top {{ .Service.Name }}'

stats:
  maxDuration: 3m
  graphs:
    - caption: CPU (%)
      statPath: DerivedStats.CPUPercentage
      color: cyan
    - caption: Memory (%)
      statPath: DerivedStats.MemoryPercentage
      color: green
```

Custom and bulk commands are Go templates with `{{ .Podman }}`,
`{{ .PodmanCompose }}`, `{{ .Container }}`, `{{ .Service }}`, `{{ .Image }}`,
`{{ .Volume }}`, `{{ .Network }}` and `{{ .Project }}`:

```yaml
customCommands:
  containers:
    - name: bash
      attach: true
      command: '{{ .Podman }} exec -it {{ .Container.ID }} bash'

bulkCommands:
  images:
    - name: prune dangling
      command: 'podman image prune -f'
```

Environment variables: `PODMAN_HOST`, `CONTAINER_HOST`, `DOCKER_HOST`,
`CONFIG_DIR`, and `DEBUG=TRUE`.

---

## Podman specifics

- **Podman only**: endpoints come from `PODMAN_HOST`, `CONTAINER_HOST`,
  `DOCKER_HOST`, then a running rootless or rootful Podman socket. The Docker
  context store is never read.
- **Hidden infra containers**: the per-pod `-infra` (pause) containers are
  filtered out of the containers panel, like `podman ps` does.
- **Compose flexibility**: `podman compose` is used when available, otherwise
  `podman-compose`; both `com.docker.compose.*` and `io.podman.compose.*`
  labels are understood.
- **Configurable CLI**: `commandTemplates.podman` (default `podman`) is used for
  attach/exec and shown in menus, so `podman-remote` works.
- **Compose keys**: `commandTemplates.podmanCompose` and
  `commandTemplates.composeConfig`.
- **Safer stats**: missing `system_cpu_usage` deltas or an unlimited memory
  limit report `0%` instead of `Inf`/`NaN`.

---

## Troubleshooting

**"Can't reach the podman service socket"** — the service is not running. Start
`podman.socket` (see [Requirements](#requirements)) or export `CONTAINER_HOST`
pointing at your socket.

**"connection to the podman service failed"** — check that `podman info` works as
the same user. For rootless Podman run without `sudo` and make sure
`XDG_RUNTIME_DIR` is set.

**Empty services panel** — run lazypodman from the compose project directory (or
pass `-p`), and make sure `podman compose config --quiet` (or
`podman-compose config --quiet`) succeeds there.

**No logs or stats** — only the last 60 minutes of logs are fetched by default;
change `logs.since` in the config.

**Cannot select text with the mouse** — hold `option`/`alt` while dragging, or
set `gui.ignoreMouseEvents: true`.

---

## Development

```sh
GOFLAGS=-mod=vendor go build ./...
GOFLAGS=-mod=vendor go test ./...
GOFLAGS=-mod=vendor go vet ./...
gofmt -l main.go pkg/ scripts/

# regenerate docs/keybindings after changing i18n strings
GOFLAGS=-mod=vendor go run ./scripts/cheatsheet generate
```

See [CONTRIBUTING.md](CONTRIBUTING.md) and [CLAUDE.md](CLAUDE.md).

---

## License

MIT. See [LICENSE](LICENSE) for the full text and copyright notice.
