# Lazypodman

**The lazier way to manage everything podman.**

A terminal UI for Podman: containers, images, volumes, networks, compose
projects, logs and stats in one window, every common action one keypress away.

---

## Features

- View the state of your Podman containers and compose services at a glance
- View **pods** with their containers, merged logs, container count and ports
- View and follow logs for a container, a service or a pod, with a bounded live
  view, full-history toggle and jump-to-newest
- ASCII graphs of container CPU/memory, disk usage (total and writable layer),
  RAM usage, uptime, or of any stat path you configure
- Attach to a container's own process: `a` in a separate terminal window (or
  via `TERMINAL`), `A` in the current terminal, `F2` in the in-app shell tab.
  `x` or `ctrl+x` instead opens a new shell (`zsh` when the container has it,
  otherwise `bash`, then `sh`), so you can look around without touching the
  running process; `ctrl+e` opens that shell in a separate terminal window.
  `ctrl+a` attaches with the full logs replayed first.
  View processes (`top`), config and env in their own tabs
- Start/stop/restart/pause/remove/rebuild containers and services, run a new
  container from an image, build an image from a Containerfile, or commit a
  container to an image with fully editable menus
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
→ Volumes → Networks**; the main panel shows the selected item's **Logs /
Stats / Diagnose / Health / Env / Config / Top / Events / Compose Config /
About** tabs.
The Networks panel has **Ports / Config** tabs. When the pods feature is
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
| `ctrl+q` | quit even while the in-app shell has focus (where `q` is sent to the shell) |
| `+` / `_` | screen mode (normal / half / fullscreen) |
| `o` / `e` | open / edit the config (Projects panel) |
| `B` | build an image from a Containerfile (any panel) |

### Panel keys

**Containers**: `d` remove · `e` show/hide stopped · `p` pause · `s` stop ·
`r` restart · `a` attach in a new terminal · `A` attach in this terminal ·
`ctrl+e` exec shell in a new terminal · `ctrl+a` attach with full logs ·
`F2` attach in the shell tab ·
`m` logs · `ctrl+f` search logs across containers · `ctrl+t` transcript view ·
`t` run healthcheck · `C` commit to an image · `v` container files ·
`Q` Quadlet menu · `ctrl+x` open/close the in-app shell tab ·
`c` custom command · `b` bulk commands · `w` open first published port in the
browser · `space` mark/unmark · `Q` export a Quadlet unit.

The **Ports** tab on the Networks panel lists every published host port with
its container, protocol and network, marks host-port collisions, and needs no
extra process.

**Container files** (`v`) opens a small menu for listing a directory
(`podman exec … ls -la`), downloading a file (with an overwrite confirmation),
uploading a file after a confirmation, and showing `podman diff`. Paths are
argv arguments, never a shell string.

The **Diagnose** tab explains the container's current state from the inspect
data already in memory: status, exit code, OOM kill, restart count and policy,
uptime, memory limit, command, image, last healthcheck output, and a plain
language "likely cause". Exit codes 0, 1, 125, 126, 127, 137, 139 and 143 have
specific explanations; other codes show the number and point at Logs. It spawns
no process.

The **Health** tab shows the health status, the failing streak and the recorded
probes (newest first) from the container inspect data, so it costs no extra
process or socket call. `t` runs `podman healthcheck run` once and reports
`healthy` or `unhealthy` on the status line; an unhealthy result is a result,
not an error. The run is recorded by Podman: it appends to the health log, can
change the failing streak and may trigger `--health-on-failure` behaviour.

**Services**: `u` up service · `U` up project · `D` down project · `s`/`S`
stop/start · `r`/`R` restart / restart menu · `p` pause · `a` attach in a new
terminal · `A` attach in this terminal · `ctrl+e` exec shell in a new terminal ·
`F2` attach in the shell tab ·
`m` logs · `ctrl+x` open/close shell · `c`/`b` custom / bulk command · `w` open port.

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

**Images**: `r` run · `u` update · `d` remove · `c` custom command · `b` bulk
commands (prune). **Volumes, Networks**: `d` remove · `c` custom command · `b` bulk
commands (prune).

### Logs

- Opening logs shows the **last 20 lines** by default (`logs.tail: 20`). Change
  it in the config (for example `tail: '200'`, or `''` for everything).
- Output streams live. The view is capped at 10 000 lines (trimmed at 20 000)
  and at `logs.maxBytes`, so a very chatty container cannot grow memory without
  bound. `ctrl+l` (full history) raises both budgets tenfold, so the history you
  asked for stays scrollable; while you are scrolled up the view stops trimming
  for the same reason and keeps its place when it eventually has to.
- `ctrl+l` reloads the current logs with the **full history**, and `ctrl+l` again
  goes back to the default tail. The choice is remembered per container.
- `ctrl+t` switches the Logs tab between plain text and a **transcript view**.
  Transcript mode replays the stream through a terminal emulator and shows the
  final screen, which keeps the layout of full-screen programs readable instead
  of showing their raw cursor and erase sequences. Full-screen output is
  detected automatically and switches the tab for you (entering the alternate
  screen, clearing the screen, or repainting with cursor moves; a progress bar
  that only erases a line does not count); `ctrl+t` toggles it back, and that
  choice is remembered per container.
- Transcript mode replays `logs.transcriptTail` lines (500 by default) instead
  of the 20-line tail, because the last 20 lines are usually only the final
  prompt. Setting it to 0 replays only `tail`. Replaying thousands of lines
  makes the Podman service scan a large log file, which is where a memory
  spike on a busy container comes from.
- `G` jumps to the newest line and resumes following. Scrolling up pauses
  following; the scrollbar on the right shows where you are.
- `m` opens logs for the selected container or service; closing the in-app
  shell lands on the first tab, Logs.

### In-app shell

- Select a container (or service) and press `x` → **exec shell**, or press
  `ctrl+x` directly. A real PTY opens in the **Shell** tab, which comes first
  while it exists, and the app switches to it. Lazypodman uses `zsh` if the
  container has it, otherwise `bash`, then `sh`. `ctrl+e` runs the same shell in
  a separate terminal window, which also works on containers that cannot be
  attached to.
- The terminal is a VT10x screen, so `vim`, `htop` and `less` work. The line above
  it shows the container name (yellow), image (magenta), status (green/red),
  short ID (grey) and pod when there is one.
- `F2` attaches to the container's own process in this same tab, instead of
  opening a shell there. A container started without stdin (`-i`) cannot be
  attached to, so `F2` reports that instead of opening a dead terminal.
  `ctrl+a` is the same attach with the full logs replayed first.
- Each container can keep its own shell. Opening a shell in another container
  leaves the first one running in the background; going back to that container
  shows its shell tab again.
- `ctrl+x` closes the shell, wipes the view and reloads the logs; exiting the
  shell inside the container closes the tab too. The tab only exists while a
  shell is open, so nothing is spawned at startup.
- `PgUp`/`PgDn` or the mouse wheel page through up to **2000 lines** of
  scrollback (including bursts such as `cat` of a long file). A thumb appears in
  the last column; it hides while you read old output and the caret comes back at
  the bottom, or as soon as you type/`Enter`.
- `ctrl+o` leaves the terminal focused but keeps the shell running, so `[`/`]`,
  the palette and the rest of the UI work again. `ctrl+q` quits Lazypodman from
  the shell; plain `q` is forwarded to the program running inside the shell. `esc` is forwarded to the app
  inside the shell.
- Full-screen apps use their own screen buffer; paging keys belong to the app
  there, so scrollback is available for normal shell output.

### Run a new container

- Select an image and press `r` (also the first item in the `x` menu) to open the
  run menu. Untouched rows show the value Podman will use, such as `on`,
  `off`, `no` or `missing`; empty values show `-`. **Name** stays `auto`, so
  Podman picks a name unless you type one. Pressing **Run** without changes
  runs that image detached with those values.
- `enter` edits a value, `space` toggles a switch. Ports, volumes, env, labels and
  the other repeatable flags open a list where `enter` edits or deletes a value
  and **add** creates one. Menu navigation wraps: down on the last row goes to
  the first, up on the first goes to the last.
- **Image defaults**: the menu inspects the selected image and prefills the
  command, entrypoint, working directory, user and health command it declares.
  Exposed ports are listed and can be published with one host-port prompt.
- **Run like a container**: copy the visible settings (ports, mounts, env,
  labels, restart policy, tty/stdin) from any existing container, then edit
  before running. The container name is not copied.
- **Restore last run**: re-open the last submitted form for this image during
  the session.
- The menu shows the current command preview and a **Checks** row for invalid
  ports, name or host-port conflicts, missing bind paths, privileged mode and
  other warnings. Blocking problems stop the run before the confirmation.
- **Advanced** groups the less common flags: network/DNS, storage, resources,
  security, healthcheck, runtime/image and environment/labels. Anything not in
  the menu can be typed into **Extra args**.
- Before anything runs, the full `podman run` command is shown in a confirmation
  panel. Arguments are passed as an argv slice (never through a shell), and a
  failure shows Podman's own output. Success refreshes containers and services.

### Update an image

- Press `u` on an image to pull its tag again with `podman pull <tag>`, so a
  moving tag such as `:latest` is refreshed. The first repository tag is used,
  falling back to `name:tag`; an untagged image shows an error instead of
  guessing a registry.
- The pull runs with a status line. Podman's own error output is shown on
  failure. Afterwards the image ID is compared with the previous one, and the
  status line says either **image already up to date: `<ref>`** or **image
  updated: `<ref>`**. Existing containers keep running on the old image until
  they are recreated.

### Commit a container to an image

- Press `C` on a container (also in the `x` menu, and `C` on a marked selection)
  to open the commit menu. **Image** is prefilled as
  `<container>_<timestamp>` so the result has a name; **Author**, **Message** and
  **Instructions** (`--change`) are optional.
- **Advanced** covers config file, format, image ID file, include volumes, quiet
  and squash. Anything else goes in **Extra args**. The container is passed
  after `--`, so a name can never be mistaken for a flag.
- The exact `podman commit` command is previewed and confirmed. Failures show
  Podman's output verbatim; success reloads the image list.

### Build an image

- Press `B` from any panel to open the build menu; it does not use the selected
  image. **Context** is the build
  directory (`.` by default), **Containerfile** stays `auto` so Podman finds the
  `Containerfile`/`Dockerfile`, and **Tags** is a repeatable `-t` list.
- **Advanced** groups build options, build args/labels, network/DNS, resources,
  security, output/cache and auth/secrets. Repeatable flags open add/edit/delete
  lists; anything unmodelled goes in **Extra args**.
- **Build** previews the exact `podman build` command and asks for confirmation.
  Arguments are passed as an argv slice (never through a shell). A failed build
  shows Podman's own output; a successful one reloads the image list.

### Events, multi-select, palette and Quadlet

- The **Events** tab (Containers panel, last tab) streams `podman events`
  verbatim. The stream is tied to the tab: switching away kills the child.
  Without a usable Podman CLI the tab says `cli off: <reason>` and spawns
  nothing.
- `space` marks a container. While anything is marked, `s` (stop), `r`
  (restart), `p` (pause/unpause) and `d` (remove) act on every marked container
  and the confirmation names the count. A successful action clears the marks;
  `space` on a marked row unmarks it. Marks are keyed by container ID, so they
  survive a list refresh.
- `ctrl+f` opens a **log search** across every visible container. It reads the
  last 500 lines of each container (bounded to 4 MiB per container and a 5 s
  timeout), groups matches by container and focuses the main panel so the
  results can be scrolled. The search runs as a cancellable task, so leaving the
  panel stops it. Opening a container replaces the results with that container's
  normal tab.
- `ctrl+p` opens the **command palette**: jump to any panel, the kube menu, a
  healthcheck run, a log search, Podman connections, `podman auto-update`, the
  Quadlet menu, or quit. It calls the same handlers as the keys, so it cannot drift from them.
- `Q` opens the **Quadlet** menu: export the selected container, or manage the
  units already installed in `gui.quadletDir`. The manager lists `.container`,
  `.pod`, `.kube`, `.network` and `.volume` files, then offers start, stop,
  restart, enable, disable, `systemctl --user daemon-reload`, journal logs, edit
  and delete (with confirmation). It never enables anything automatically.
- Exporting a container writes a Quadlet `.container` unit. The preview
  is shown first; the file goes to `gui.quadletDir` (default
  `<user config dir>/containers/systemd`, where Podman looks for user units) as
  `0600` after a confirmation naming the full path. The unit contains
  `Image=`, `ContainerName=`, `Environment=` (plaintext, quoted, newlines
  flattened), `Exec=`, `PublishPort=`, `Volume=` (named volumes keep their name),
  `Network=` and a `[Service]`/`[Install]` section.
  It is a readable starting point, not a complete translation of every inspect
  field; the generator does not run `systemctl` or enable anything.
- The command palette's **Podman connections** entry lists
  `podman system connection ls` and restarts lazypodman with the chosen endpoint
  (`CONTAINER_HOST`/`DOCKER_HOST`). A restart is required because every panel
  and log stream belongs to the old store. On Windows the menu explains the
  manual environment variable instead.
- `podman auto-update` is available in the Containers panel bulk menu (`b`). It
  confirms first, then streams the output into the main panel and refreshes the
  container list. It updates only containers that carry the auto-update label,
  regardless of the current selection.

The full generated list for every language lives in
[docs/keybindings](/docs/keybindings) and can be regenerated with
`GOFLAGS=-mod=vendor go run ./scripts/cheatsheet generate`.

### Typical workflows

1. **Debug a failing service**: select it, `m` for logs, `ctrl+l` if you need the
   full history, `]` for the Stats tab, `ctrl+x` to open a shell.
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
  since: '60m'                # '' = all logs (stopped containers ignore it)
  tail: '20'                  # default tail; '' = full history (ctrl+l toggles)
  maxLineLength: 4096         # split long horizontal lines; 0 = keep whole (slower)
  maxBytes: 1048576           # rendered log kept in the Logs tab; 0 = no byte cap
  transcriptTail: 500         # lines replayed in the transcript view; 0 = use tail

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
`CONFIG_DIR`, `TERMINAL` (attach terminal emulator) and `DEBUG=TRUE`.

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

See [CLAUDE.md](CLAUDE.md) for the build rules. Startup and CLI spawn costs are
measured in [docs/performance.md](docs/performance.md).

Design records for the Podman-native features live in
[docs/superpowers/specs](docs/superpowers/specs) (pods, healthcheck, kube, UX,
Quadlet), with the matching plans in [docs/superpowers/plans](docs/superpowers/plans).
Every behaviour that depends on a real Podman is still waiting for a human run on
a real rootless Podman.

---

## Credits

Lazypodman is a fork of [lazydocker](https://github.com/jesseduffield/lazydocker),
created by Jesse Duffield and contributors. The original project is MIT licensed
and its copyright notice is preserved in [CREDITS.md](CREDITS.md).

---

## License

Lazypodman itself is distributed under the **MIT License**:

```
MIT License

Copyright (c) 2026 Fundoku

This project is a fork of lazydocker. Portions are 
Copyright (c) 2018 Jesse Duffield, see CREDITS.md.

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

See [LICENSE](LICENSE) for the canonical file and [CREDITS.md](CREDITS.md) for
upstream attribution.
