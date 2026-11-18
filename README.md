# Lazypodman

**The lazier way to manage everything podman.**

A terminal UI for Podman: containers, images, volumes, networks, compose
projects, logs and stats in one window, every common action one keypress away.

---

## Features

- View the state of your Podman containers and compose services at a glance
- View **pods** with their containers, merged logs, container count and ports
- View and follow logs for a container, a service or a pod
- ASCII graphs of container CPU/memory, or of any stat path you configure
- Attach to a container, exec a shell, view its processes (`top`), config and env
- Start/stop/restart/pause/remove/rebuild containers and services
- Inspect image layers, prune containers, images, volumes and networks
- Custom commands and bulk commands, configured per panel
- Pods, kube generate/play/down, a Health tab with healthcheck runs, a live
  `podman events` tab, multi-select, a command palette, Quadlet export and
  `podman auto-update`
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

Dependencies are vendored, so building needs no network access after cloning:

```sh
git clone https://github.com/fundokukiri/lazypodman.git
cd lazypodman
GOFLAGS=-mod=vendor go build -o lazypodman .
./lazypodman
```

That binary only exists inside the repo directory. To run `lazypodman` from
anywhere, install it somewhere on your `PATH`:

```sh
# a) into the Go bin directory (usually ~/go/bin, no sudo needed)
GOFLAGS=-mod=vendor go install .
export PATH="$PATH:$(go env GOPATH)/bin"     # add this line to ~/.bashrc or ~/.zshrc

# b) into ~/.local/bin (must already be on your PATH)
GOFLAGS=-mod=vendor go build -o ~/.local/bin/lazypodman .

# c) system-wide
GOFLAGS=-mod=vendor go build -o lazypodman .
sudo install -Dm755 lazypodman /usr/local/bin/lazypodman
```

Then check that it resolves from any directory:

```sh
cd ~ && which lazypodman && lazypodman --version
```

If `PATH` changes do not stick, run `source ~/.bashrc` (or open a new terminal).
After pulling new commits, repeat the same build/install command to update it.

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
# after installing the binary on your PATH
echo "alias lzp='lazypodman'" >> ~/.zshrc   # or ~/.bashrc
```

### Uninstall

Nothing runs in the background and no systemd unit is created, so removing
lazypodman means deleting the binary and (optionally) its config.

1. Find where the binary was installed:

```sh
which lazypodman        # e.g. /usr/local/bin/lazypodman or ~/go/bin/lazypodman
```

2. Delete that file, using whichever path matches how you installed it:

```sh
sudo rm /usr/local/bin/lazypodman   # system-wide install
rm ~/go/bin/lazypodman              # go install
rm ~/.local/bin/lazypodman          # ~/.local/bin install
```

3. Delete the configuration directory (this also drops the merged config and any
   state written by lazypodman):

```sh
rm -rf ~/.config/lazypodman                              # Linux
rm -rf ~/Library/Application\ Support/lazypodman         # macOS
rmdir /s /q "%AppData%\lazypodman"                       # Windows
```

If you used a custom `CONFIG_DIR`, delete that directory instead.

4. Remove the `lzp` alias or the `PATH` line you added to `~/.bashrc`,
   `~/.zshrc` or `~/.config/fish/config.fish`, then reload the shell. If you
   built the container image, remove it too:

```sh
podman rmi lazypodman
rm -rf lazypodman      # the git clone, if you no longer need the sources
```

Your containers, images, volumes and networks are untouched: uninstalling
lazypodman does not modify the Podman store.

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

Layout: the left column holds **Project → Services → Pods → Containers → Images
→ Volumes → Networks**; the main panel shows the selected item's **Logs / Stats /
Health / Config / Env / Top / Compose Config / About** tabs. When the pods feature is
unavailable the Pods panel is hidden and the remaining panels keep contiguous
numbers (`1`…`6`).

### Navigation

| Key | Action |
| --- | --- |
| `1`…`7` | focus Projects / Services / Pods / Containers / Images / Volumes / Networks (`6` keys when Pods is off) |
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
`r` restart · `a` attach · `m` logs · `t` run healthcheck · `E` exec shell (in-app terminal) ·
`ctrl+x` close the shell tab ·
`c` custom command · `b` bulk commands · `w` open first published port in the
browser · `space` mark/unmark · `Q` export a Quadlet unit.

The **Health** tab shows the health status, the failing streak and the recorded
probes (newest first) from the container inspect data, so it costs no extra
process or socket call. `t` runs `podman healthcheck run` once and reports
`healthy` or `unhealthy` on the status line; an unhealthy result is a result,
not an error. The run is recorded by Podman: it appends to the health log, can
change the failing streak and may trigger `--health-on-failure` behaviour.

**Services**: `u` up service · `U` up project · `D` down project · `s`/`S`
stop/start · `r`/`R` restart / restart menu · `p` pause · `a` attach · `m` logs ·
`E` exec shell · `c`/`b` custom / bulk command · `w` open port.

**Pods**: `s` stop · `S` start · `r` restart · `p` pause/unpause · `d` remove
(menu, with force) · `b` bulk commands (prune stopped pods) · `c` custom command ·
`g` kube menu (generate, play or down a YAML manifest).
Tabs: **Logs** (merged, prefixed with the container name), **Stats** (sum over
the pod's containers), **Containers**, **Config** (`pod inspect`).

The kube menu keeps manifests in `gui.kubeDir` (default
`<user config dir>/lazypodman/kube`):

- **Generate** exports the selected pod with `podman kube generate`, shows the
  YAML as a preview and saves `<pod>.yaml` with mode `0600` only after a
  confirmation that names the full path and warns that env vars are plaintext.
  An existing file asks before it is overwritten.
- **Play** and **Down** list `*.yaml`/`*.yml` files from that directory whose
  top-level `kind:` makes them kube manifests. Nothing runs automatically: both
  confirm with the full path, then stream the output of `podman kube play`
  (`--down` for down) into the main panel. Cancelling a task kills the process.
- Both features are probed lazily and fall back to the older
  `podman generate kube` / `podman play kube` syntax. No version numbers are
  assumed; if neither works the corner shows `kube generate off:` / `kube play off:`. 
  `--replace` and `--build` are not passed.

**Images, Volumes, Networks**: `d` remove · `c` custom command · `b` bulk
commands (prune).

### Events, multi-select, palette and Quadlet

- `x` opens the usual options menu; its **exec shell** entry (`E`) starts a real
  PTY (`podman exec -it … /bin/sh`) in a new **shell** tab right after **Logs**
  and switches to it. The terminal is a VT10x screen buffer, so `vim`, `htop` and
  `less` work; colours are parsed but drawn as plain text for now. `ctrl+x`
  closes the shell and removes the tab; exiting the shell inside the container
  does the same. Before this the tab does not exist, so nothing is spawned at
  startup.
- The **Events** tab (Containers panel, last tab) streams `podman events`
  verbatim. The stream is tied to the tab: switching away kills the child.
  Without a usable Podman CLI the tab says `cli off: <reason>` and spawns
  nothing.
- `space` marks a container. While anything is marked, `s` (stop), `r`
  (restart), `p` (pause/unpause) and `d` (remove) act on every marked container
  and the confirmation names the count. A successful action clears the marks;
  `space` on a marked row unmarks it. Marks are keyed by container ID, so they
  survive a list refresh.
- `ctrl+p` opens the **command palette**: jump to any panel, the kube menu, a
  healthcheck run, `podman auto-update`, the Quadlet export, or quit. It calls the
  same handlers as the keys, so it cannot drift from them.
- `Q` exports the selected container as a Quadlet `.container` unit. The preview
  is shown first; the file goes to `gui.quadletDir` (default
  `<user config dir>/containers/systemd`, where Podman looks for user units) as
  `0600` after a confirmation naming the full path. The unit contains
  `Image=`, `ContainerName=`, `Environment=` (plaintext!), `Exec=`,
  `PublishPort=`, `Volume=`, `Network=` and a `[Service]`/`[Install]` section.
  It is a readable starting point, not a complete translation of every inspect
  field; the generator does not run `systemctl` or enable anything.
- `podman auto-update` is available in the Containers panel bulk menu (`b`). It
  confirms first, then streams the output into the main panel and refreshes the
  container list. It updates only containers that carry the auto-update label,
  regardless of the current selection.

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
| macOS | `~/Library/Application Support/lazypodman/config.yml` |
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
  showPodColumn: true         # pod column in the containers panel
  pods:
    enabled: "auto"           # auto | always | never

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
- **Healthcheck runs are lazy**: pressing `t` probes `podman healthcheck run`
  once, only then. A missing subcommand is cached for the session; a timeout is
  retried on the next press. Nothing is claimed about a feature before it is
  probed, and the exit-code table shown in the docs is unverified beyond
  `0 = healthy`.
- **Pods**: the pods panel needs the `podman` CLI to talk to the same store as
  the socket. It is enabled only when the binary exists, is Podman ≥ 3.0 and its
  socket path matches the endpoint in use; otherwise the panel is hidden and the
  bottom-right corner shows `pods off: <reason>`. Socket-only setups (for example
  `CONTAINER_HOST=ssh://…` or macOS/Windows `podman machine`) stay fully usable,
  only without pods. `gui.pods.enabled: always` skips the store comparison and
  `never` disables pods outright.
- **Kube**: `g` on the pods panel generates a manifest from a pod or plays/downs a
  manifest from `gui.kubeDir`. The YAML is written `0600`; the directory `0755`.
  The commands are discovered by probing `podman kube generate|play` first and the
  older `podman generate kube` / `podman play kube` second. Warnings: the generated
  file contains env vars in plaintext (no Secrets), and `play`/`down` always ask
  with the full path.
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

See [CONTRIBUTING.md](CONTRIBUTING.md) and [CLAUDE.md](CLAUDE.md). Startup and CLI
spawn costs are measured in [docs/performance.md](docs/performance.md).

Design records for the Podman-native features live in
[docs/superpowers/specs](docs/superpowers/specs) (pods, healthcheck, kube, UX,
Quadlet), with the matching plans in [docs/superpowers/plans](docs/superpowers/plans).
Every behaviour that depends on a real Podman is still waiting for a human run of
[TEST.md](TEST.md).

---

## License

Lazypodman is a fork of [lazydocker](https://github.com/jesseduffield/lazydocker)
by Jesse Duffield and contributors, and is distributed under the same **MIT
License**. The original copyright notice is kept in [LICENSE](LICENSE):

```
MIT License

Copyright (c) 2018 Jesse Duffield

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
