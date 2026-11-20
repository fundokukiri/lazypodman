# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **Stats is the first tab again**: Containers, Services and Pods open on
  **Stats** instead of **Logs**, so moving through the list no longer asks the
  Podman service for a log tail per container. `m` (or `]`) still reaches Logs
  in one key, and closing the in-app shell falls back to Stats. The in-app
  **Shell** tab still comes first while it exists.

### Fixed

- **The `x` menu is no longer hidden by the shell tab**: every layout pass
  raised the shell view to the very top of the stack, so with a shell tab open
  the options menu (and the confirmation panel) drew underneath it and the key
  looked dead. The shell is now kept above the main view it overlaps and below
  the popups.

### Changed

- **New key layout for getting into a container**: `a` still attaches to the
  container's own process in a separate terminal window. On top of that,
  `ctrl+a` now attaches in the in-app shell tab (was `F2`), `F2` now attaches in
  the current terminal (was `A`), `ctrl+e` opens and closes the shell tab (was
  `ctrl+x`, and it stays the close key inside the tab), and `e` opens a `zsh`
  shell in a separate terminal window (it was `ctrl+e`). `A` and `ctrl+x` are
  gone, and hiding stopped containers moved from `e` to `H` so `h` keeps meaning
  "previous panel".

### Removed

- **Attach with the full logs replayed**: `ctrl+a` used to run `podman logs`
  before attaching, and the new layout has no key left for it. The attach itself
  (`a`, `F2`, `ctrl+a`) is unchanged.

### Fixed

- **The window opened by `a` now outlives lazypodman**: it used to be started
  inside lazypodman's process group and session, so quitting lazypodman with
  `ctrl+c` (or closing the terminal hosting it) signalled the attached window
  too and `podman attach` died with the UI. The terminal launcher now runs in
  its own session, so the container keeps running and the window stays open
  after lazypodman exits.

## [1.0.3] - 2026-11-20

### Changed

- **Logs is the first tab again**: the Containers, Services and Pods panels open
  on **Logs** with Stats second. The in-app **Shell** tab still comes first
  while it exists, so `ctrl+x` and `F2` land on it, and closing the shell falls
  back to Logs.

### Fixed

- **The transcript view no longer leaks memory on the Podman host**: opening the
  Logs tab on a busy container spiked RAM somewhere other than lazypodman's own
  process, because transcript mode ignored `logs.tail` and asked for 5000 lines
  — the Podman service had to read a large log file and hold it. The replay now
  has its own budget, `logs.transcriptTail` (500 by default, 0 to follow
  `tail`), the automatic switch no longer fires on progress bars (which end
  every update with an erase-line and cursor-up), and an idle transcript no
  longer redraws the whole view 7 times a second.

- **`ctrl+l` and `ctrl+t` have a line budget**: the full-history view asks for
  at most `logs.fullTail` lines (10 000 by default, 0 for everything) instead of
  the whole log file, and the transcript view replays at most
  `logs.transcriptTail` lines (500) no matter what `tail` or `ctrl+l` say —
  switching to transcript with the whole history on used to stream every line
  through the terminal emulator and never finish.

- **Full history no longer raises the view budget**: the rendered log is capped
  at `logs.maxBytes` (1 MiB) and 10 000 lines whether or not `ctrl+l` is on. A
  rendered line costs eight bytes per character inside the view, so the previous
  tenfold budget turned one look at a long history into hundreds of megabytes
  that stayed allocated until lazypodman exited.

- **Full history can be scrolled**: `ctrl+l` used to fetch the whole log and then
  keep only the newest megabyte, so scrolling back stopped almost immediately.
  The full-history view keeps ten times more, stops trimming while you are
  scrolled up, and keeps the line you are reading in place when a trim happens
  instead of jumping back to the top.

## [1.0.2] - 2026-11-20

### Added

- **Quit from the Shell tab**: `ctrl+q` quits Lazypodman while the terminal has
  focus. Plain `q` remains part of the shell input and is not a quit key there.

- **Shell selection everywhere a shell is opened**: `ctrl+x`, the `x` menu's
  **exec shell** and the new `ctrl+e` (shell in a separate terminal) run `zsh` when
  the container has it, otherwise `bash`, and fall back to `sh`. `a`, `A` and `F2` keep attaching to the container's own
  process (separate terminal, current terminal, in-app shell tab), so the tab
  shows what the container is running; `ctrl+a` is the same attach with the full
  logs replayed first. A container started without stdin reports that instead of
  opening a dead terminal.

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

- **Stats is the first tab**: the Containers, Services and Pods panels open on
  **Stats**, with Logs second. The in-app **Shell** tab comes first while it
  exists, so `ctrl+x`/`F2` land on it and closing it falls back to Stats.

- **Logs survive a switch**: the Logs tab keeps the rendered view of the last
  four containers, so browsing the list with `j`/`k` and coming back resumes with
  `--since` instead of replaying the whole tail. `ctrl+l`, `ctrl+t` and a
  container restart still force a reload.

### Fixed

- **Shell scrollback no longer panics when the screen shrinks**: the
  scrolled-off-line detector now only compares overlapping rows instead of
  slicing past the new screen.

- **CPU percentages match `podman stats`**: the percentage is the container's
  CPU time divided by the time between two samples, so 250% means 2.5 cores
  busy. The old docker formula scaled by `system_cpu_usage`, which Podman fills
  with the CPU the *host* actually consumed instead of the host's capacity, so
  an idle host reported a 0.3-core container as 240%. A restarted container
  (counter reset) and the unusably short gap before the first streamed sample
  report 0 instead of a bogus number.

- **Pod logs no longer hang with many containers**: the "N containers omitted"
  notice is written inside the closing goroutine, so a pod with more containers
  than the process cap does not deadlock on an unread pipe.

- **Health status no longer freezes**: the Health tab re-reads the container on
  screen once per tick, so it updates on Podman versions that emit no
  `health_status` event, and the tab no longer depends on the deleted per-second
  details refresh.

- **Removing containers shows the real ID**: the confirmation preview said
  `podman rm <id>` with the first character cut off; it now shows the same
  12-character ID as the rest of the app.

- **`$EDITOR` with arguments works**: `EDITOR="code --wait"` or `"vim -u NONE"`
  opens the editor instead of failing on a program named after the whole string.

- **`o` opens IPv6 publishes**: an address of `::` is treated like `0.0.0.0`
  instead of building the link `http://:::8080/`.

- **Byte sizes stay correct above 2 GiB on 32-bit builds**: the size formatters
  take `int64` instead of truncating through `int`.

- **A broken log template no longer panics**: an unparsable template returns an
  empty command instead of `template.Must` panicking at startup.

- **An empty configured command fails cleanly** with "no command" instead of
  panicking on an empty argv slice.

- **SSH tunnels clean up after a failed dial**: the retried command is killed and
  the socket directory removed, so a reconnect does not leave stale sockets.

- **Leaving the log view no longer leaks a signal goroutine**: the interrupt
  watcher exits with the log view context.

- **Container downloads cannot default to the host root**: a container path
  like `/` now suggests `container-root` instead of `/`, and the host root is
  rejected as a copy destination.

- **Pause no longer panics before inspect data loads**: the key shows the
  waiting message instead of dereferencing a nil state.

- **Run checks handle host IPs and paused containers**: the same host port on
  different IPs is allowed, while ports held by paused containers are reported
  as conflicts.

- **Slow Podman calls no longer hold the container lock**: details refresh does
  not take the global lock, and container listing has a timeout.

- **Auto-update output is bounded**: it streams through the same capped log
  writer as Events instead of rebuilding the full output for every line.
  Container stats accept frames larger than 64 KiB.

- **Socket discovery only accepts real sockets**, and the rootless fallback
  path is absolute even when `XDG_RUNTIME_DIR` is unset.

- **Concurrent inspect refreshes no longer race the UI**: container details are
  published through a lock and read as a snapshot, so a background refresh
  cannot expose a half-assigned inspect struct while Logs, Diagnose or the
  shell header renders. Inspect calls now have a timeout instead of hanging a
  refresh forever.

- **Slow Podman no longer freezes the UI**: opening Run Container, the Podman
  connection list and the Quadlet export now run their Podman work in a
  background status task and open the menu/prompt when it finishes. A stuck
  socket no longer blocks the gocui event loop.

- **Events stays bounded**: the Events tab streams through the same bounded log
  writer as Logs instead of rebuilding and re-rendering the full event list on
  every line. A long-running Events tab no longer grows without limit.

- **Port conflicts handle wildcard addresses**: `0.0.0.0`, `::` and an empty
  address conflict with the same port on any interface, and Run validation
  ignores ports from containers that are already stopped.

- **Transcript detection ignores cursor hiding alone**: progress bars that only
  hide the cursor no longer switch the Logs tab to a mostly empty terminal
  replay. Alternate-screen and multiple cursor/erase sequences still trigger it.

- **Log cache resets across a container restart**: the cached Logs view is tied
  to the container's start time, so a restart no longer appends the new run to
  the previous run's output.

- **Long lines in streamed Podman output are read completely**: `podman events`
  and `auto-update` output no longer deadlock when one line is larger than the
  old 1 MiB scanner limit.

- **Large container file copies are not cut off at 30 seconds**: `podman cp`
  runs without the short command timeout used for `ls`/`diff`, so a large
  download or upload can finish.

- **Quadlet export quotes values and keeps named volumes named**: newlines in
  env/command values cannot inject unit directives, and `Volume=` uses the
  volume name rather than Podman's internal host path.

- **Shell shortcuts open a separate terminal**: `a` starts a new `podman
  exec` shell in a new terminal window, and `A` starts one in the current
  terminal. The launcher honours `$TERMINAL` (e.g. `kitty`, `alacritty`,
  `wezterm`), then tries `xdg-terminal-exec`, x-terminal-emulator, GNOME
  Terminal, Konsole, Xfce Terminal and the common lightweight terminals. macOS
  uses Terminal.app and Windows opens a new console window. If no emulator is
  available, `a` falls back to the current terminal.

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
