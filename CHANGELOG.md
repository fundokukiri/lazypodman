# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Quit from the Shell tab**: `ctrl+q` quits Lazypodman while the terminal has
  focus. Plain `q` remains part of the shell input and is not a quit key there.

- **Shell selection everywhere a shell is opened**: `ctrl+x`, `E`, the `x`
  menu's **exec shell**, and `a`/`A`/`F2` now run `zsh` when the container has
  it, otherwise `bash`, and fall back to `sh`. `a`/`A`/`F2` therefore start a
  new `podman exec` shell instead of attaching to the container's main process;
  `ctrl+a` remains the full-log attach path.

- **Run Container menu improvements**: image inspect prefills the command,
  entrypoint, working directory, user and health command; exposed image ports
  can be published with one host-port prompt; **Run like a container** copies
  the visible settings from an existing container; **Restore last run** keeps
  the last submitted form per image for the session. The menu now carries a live
  command preview and a **Checks** row for invalid ports, name or host-port
  conflicts, missing bind paths and privileged mode.

- **Ports tab on the Networks panel**: lists every published host port with its
  container, protocol and network, and marks host-port collisions. It reads the
  container list already in memory.

- **Container files menu** (`v`): list a directory with `podman exec … ls -la`,
  download a file (overwrite confirmation), upload a file (confirmation), and
  show `podman diff`. Arguments are passed as an argv slice.

- **Quadlet manager** (`Q` now opens a menu): export a container or manage
  installed `.container`/`.pod`/`.kube`/`.network`/`.volume` units. Start, stop,
  restart, enable, disable, daemon-reload, journal logs, edit and delete with
  confirmation.

- **Podman connection switcher** in the command palette: lists
  `podman system connection ls` and restarts lazypodman against the chosen
  endpoint. Windows explains the manual `CONTAINER_HOST` step.

- **Diagnose tab**: explains why a container is in its current state using the
  inspect data already loaded: status, exit code, OOM kill, restart count and
  policy, uptime, memory limit, command, image, last healthcheck output, and a
  plain-language likely cause. Common exit codes (0, 1, 125, 126, 127, 137,
  139, 143) have specific explanations. It opens without spawning a process.

- **Search logs across containers**: `ctrl+f` reads the last 500 lines of every
  visible container, groups matches by container in the main panel, and focuses
  the main panel so the results can be scrolled. Each container is bounded to
  4 MiB and a 5 s timeout; the search is a cancellable task.

- **Transcript view for busy Logs tabs**: `ctrl+t` replays the container's log
  stream through a terminal emulator and shows the final screen, so output from
  full-screen programs is readable instead of a mix of cursor moves, carriage
  returns and raw escape sequences. Transcript mode replays up to 5000 lines so
  it has enough history to rebuild the current screen instead of only the last
  prompt. Full-screen output is detected from cursor, erase and alternate-screen
  sequences and switches automatically; `ctrl+t` still toggles it back, and an
  explicit switch to plain logs is remembered for that container.

### Fixed

- **Attach opens in a separate terminal**: `a` (and the `x` menu) now launches
  plain `podman attach` in a new terminal window instead of suspending
  lazypodman. Live output starts immediately at the bottom, with the terminal's
  own scrollback and the TUI stays usable. The launcher honours `$TERMINAL`
  (e.g. `kitty`, `alacritty`, `wezterm`), then tries `xdg-terminal-exec`,
  x-terminal-emulator, GNOME Terminal, Konsole, Xfce Terminal and the common
  lightweight terminals. macOS uses Terminal.app and Windows opens a new
  console window. If no emulator is available, `a` falls back to the current
  terminal. `A` always uses that current-terminal path. The attach process
  sends itself `SIGWINCH` shortly after starting, so containers that only draw
  after a resize no longer leave a black screen until a key is pressed.

- **`ctrl+a` attaches with full logs**: a separate explicit key replays the
  whole container history and then attaches, so `a`/`A` stay instant.

- **Horizontal logs wrap instead of garbling**: lines without a newline are
  wrapped at the view width, in small batches, so the Logs tab still follows
  the bottom without one huge write blocking rendering. A bare `\r` (progress
  output) starts a new display line instead of overwriting the previous text.
  A logical line is split at `logs.maxLineLength` bytes (default 4096) so a
  20-line tail cannot hide a multi-megabyte line. Splitting never drops text and
  follow mode keeps receiving new output; set `maxLineLength: 0` to disable it.
  Terminal cursor/erase sequences are stripped (colour changes are kept), so
  progress output no longer leaves stray control bytes in the text.

- **Switching tabs no longer freezes on the Logs tab**: an incomplete UTF-8
  rune or escape sequence at the end of a read chunk was re-queued forever,
  spinning the old log task. A malformed long escape could do the same. The
  writer now also checks cancellation between batches, so leaving the tab does
  not wait for the whole log chunk to be rendered.

- **The shell caret stays visible**: at the bottom of the Shell tab the
  renderer takes the cursor from the live terminal screen instead of the bottom
  window of the tall scrollback screen. Reverse-video cells on the default
  foreground/background are also rendered as reverse video again, so TUIs that
  hide the hardware cursor and draw the input caret as a reversed block (for
  example free-pi-cli) show the caret, including when that block is the first
  cell on a line.

- **Mouse tab switching stays responsive**: clicking a main-view tab now
  updates the tab highlight immediately and always repaints the tab strip, even
  when the previous Logs stream is still shutting down. Tab clicks also resolve
  the owning side panel from the view stack, so clicking the main frame (which
  focuses `main`) no longer makes later tab label clicks do nothing.

- **A different container starts on the Logs tab**: switching containers no
  longer keeps the tab that was open on the previous one. Open Env on
  container A, select container B, and B starts on Logs.

- **External attach no longer starts blank**: the attach wrapper sends several
  window-size signals over the first few seconds, so an attached TUI redraws
  itself instead of staying empty until the first key press. For the full-log
  attach path the nudges start after the log replay.

- **Attach inside the Shell tab**: `F2` on a container or service
  opens `podman attach` in the in-app shell tab. It replaces the old
  `ctrl+shift+a` binding, which many terminals report as plain `A`. `a`/`A`
  still use an external or the current terminal.

- **Closing the shell leaves the Logs tab populated**: the asynchronous view
  cleanup no longer clears the main view after the Logs task has restored its
  content, so `ctrl+x` shows the existing logs immediately instead of waiting
  for a new line to arrive.

- **Each container keeps its own shell**: opening a shell in another container
  no longer closes the first one. The other shell keeps running in the
  background and its tab comes back when that container is selected again.

- **Shell output keeps its colours**: ANSI foreground/background colours from
  the container PTY are rendered in the live shell and its scrollback instead
  of being flattened to plain text.

- **Vietnamese and other non-ASCII input works in the Shell tab**: composed
  IME characters are now forwarded to the container PTY instead of being
  swallowed by the UI.

- **The selected container stays highlighted while a main tab has focus**:
  clicking Logs/Stats/etc. no longer makes the active side panel's selection
  disappear, so it is still clear which container those tabs belong to. Other
  panels keep their normal, unhighlighted rows.

- **Restarting or stopping a container keeps it selected**: the container is
  re-found by ID after the refresh instead of leaving the cursor on the row it
  used to occupy, so the side panel and main tab stay on the same container
  when it moves between the running and exited groups. Stop and restart now
  also refresh immediately.

- **Returning to the Logs tab no longer reloads it**: switching to Stats,
  Health, Env or another main tab and back restores the last Logs view and
  resumes from the time it was cached, so the tail is not replayed and logs
  written while away are not lost. `ctrl+l` still forces a real full-history
  reload.

- **Stopped containers no longer show an empty Logs tab**: the `logs.since`
  age filter is skipped for stopped containers, so the last `logs.tail` lines
  are shown even when they are older than the window. `ctrl+l` now loads full
  history without the age filter too.

- Logs now have a byte budget (`logs.maxBytes`, default 1 MiB) as well as the
  line cap. The initial tail is held for 120 ms and then drawn once at the
  bottom, so opening Logs no longer replays the whole history visibly. This
  keeps the tab's memory bounded even when every line is 4 KiB wide.
- Logs keep streaming immediately (old behaviour) but the view is capped at
  10.000 lines, trimmed when it passes 20.000, so follow mode no longer grows
  memory without bound. Set `logs.tail` to also limit the initial fetch.

- `space` now actually marks a container (and selects a menu item): gocui
  delivers the spacebar as `KeySpace` with an empty rune, so the previous rune
  `' '` bindings never fired.

### Added

- **Container disk usage in Stats**: opening the Stats tab asks `podman
  inspect --size` once and shows the total rootfs size and the writable layer
  above the graphs. The Stats tab also shows **Uptime**, derived from the
  container's `StartedAt`, formatted as seconds/minutes/hours/days and marked
  `not running` or `unavailable` when it cannot be computed. It also shows
  **RAM usage** as used bytes, plus used/limit and percentage when a memory
  limit is set. If the size
  inspect fails the disk line says "unavailable" instead of blocking the tab.

- **Update an image**: `u` on the Images panel pulls the selected image's
  repository tag again (`podman pull`), refreshing moving tags such as
  `:latest`. The first repo tag is used, then `name:tag`; untagged images show
  an error instead of guessing. After the pull the image ID is compared with
  the old one, so the status line reports either "already up to date" or
  "updated". Podman's output is shown on failure and the list reloads.

- **Commit a container to an image**: `C` on the Containers panel opens a
  menu for the image tag, author, message and `--change` instructions, with
  Advanced options for config, format, iidfile, volumes, quiet and squash. The
  `podman commit` argv is previewed and confirmed, the container ref goes after
  `--`, failures show Podman's output and success reloads the image list.

- **Build an image from a Containerfile**: `B` from any panel opens a
  schema-driven build menu (context, Containerfile, tags, Advanced groups for
  build args, network, resources, security, output/cache and auth/secrets). The
  exact `podman build` argv is previewed and confirmed; failures show Podman's
  output verbatim and success reloads the image list.

- **Run container from an image**: `r` on the Images panel (also the first item
  in the `x` menu) opens a schema-driven run menu. Core values are edited
  inline; ports, volumes, env and other repeatable flags have add/edit/delete
  lists; Advanced groups cover network, storage, resources, security,
  healthcheck, runtime and environment flags, with extra args for anything
  else. Defaults run the image detached, the exact argv is previewed and
  confirmed, and failures show Podman's output verbatim.

- **Shell tab with a real terminal**: `exec shell` (`E`, also reachable from the
  `x` options menu) starts `podman exec -it … /bin/sh` in a PTY and shows it in a
  **shell** tab right after Logs, auto-switched. A VT10x buffer renders
  full-screen tools such as `vim` and `htop`; `ctrl+x` closes the shell and
  removes the tab. Adds `creack/pty` and `hinshun/vt10x`.

- **Integration tests** for Podman-native features: startup check timing and process
  count, socket container list + pod snapshot, kube capability prefixes, kube
  generate, the events stream and Quadlet syntax built from a real inspect
  (`-tags integration`).
- **Startup** spawns one podman process for the CLI check instead of two: the
  `pod logs --help` probe is now lazy (first Pod logs tab), and the two pod-refresh
  calls run concurrently. See [docs/performance.md](docs/performance.md).
- **Events tab** on the containers panel streaming `podman events` verbatim; the
  stream dies with the tab and shows `cli off: <reason>` instead of spawning when
  the CLI is unavailable.
- **Multi-select** with `space` on the containers panel: while containers are
  marked, `s`/`r`/`p`/`d` act on all of them and confirm with the count.
- **Command palette** (`ctrl+p`) listing panel jumps, the kube menu, a healthcheck
  run, `podman auto-update` and quit, using the same handlers as the keys.
- **Quadlet export** (`Q` on the containers panel) writing a readable
  `.container` unit to `gui.quadletDir` (default
  `<user config dir>/containers/systemd`) with mode `0600` after a preview and a
  confirmation naming the full path.
- **`podman auto-update`** entry in the containers bulk menu, confirmed first and
  streamed into the main panel.
- **Kube menu** (`g` on the pods panel): generate a Kubernetes YAML file from a
  pod, or play/down a manifest from `gui.kubeDir` (default
  `<user config dir>/lazypodman/kube`). Generated files are written with mode
  `0600` after a confirmation that names the full path and warns that env vars are
  plaintext; play/down always confirm with the full path and stream podman's output.
  Commands are probed lazily with a fallback to the older `podman generate kube` /
  `podman play kube` syntax. `--replace` and `--build` are not passed.
- **Health** tab on the containers panel (after **Stats**) showing health
  status, failing streak and the recorded probes, all from inspect data the
  socket already provides.
- `t` on the containers panel runs `podman healthcheck run` once. Healthy and
  unhealthy are results shown on the status line (an unhealthy exit is not an
  error); only spawn failures and undocumented exit codes open the error panel.
  Documented exit codes beyond `0` are unverified; anything outside `0`/`1`/`2`
  is treated as an error.
- Per-feature capability state (`unknown`/`available`/`unavailable`): features
  that need the CLI are probed lazily on first use, so startup still runs two
  podman processes. A failed probe adds `health run off: <reason>` to the
  bottom-right corner; a CLI failure shows a single `cli off: <reason>`.
- Pods panel (between Services and Containers) with **Logs**, **Stats**,
  **Containers** and **Config** tabs. Pod logs are merged and prefixed with the
  container name; pod stats are the sum over the pod's containers.
- Pod lifecycle keys: `s` stop, `S` start, `r` restart, `p` pause/unpause,
  `d` remove (with a force option), `b` bulk commands (prune stopped pods),
  `c` custom `podman` commands on the selected pod.
- `gui.showPodColumn` (default `true`): a pod column in the containers panel,
  shown only when pods are available.
- `gui.pods.enabled` (`auto` | `always` | `never`): how hard to try to enable the
  pods feature. Availability needs the `podman` CLI (>= 3.0) on the same store as
  the socket; otherwise the panel is hidden and the reason is shown as
  `pods off: <reason>` in the bottom-right corner.
- `customCommands.pods` / `bulkCommands.pods` config groups, with `{{ .Pod }}`
  template data.

### Changed

- Panel focus keys are derived from one table: with pods enabled the keys are
  Project `1`, Services `2`, Pods `3`, Containers `4`, Images `5`, Volumes `6`,
  Networks `7`; without pods they stay `1`…`6` with no gaps.

## [Fork]

This project started as a Podman-only fork of
[lazydocker](https://github.com/jesseduffield/lazydocker). The original MIT
license and copyright notice are kept in [LICENSE](LICENSE).
