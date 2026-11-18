# Lazypodman — Pods as first-class citizens (Phase 1) — design

Date: 2026-11-18
Status: implemented and committed (tasks 1-11; see "Implementation status" below).
Scope: Phase 1 of the Podman-native roadmap. Phases 2–4 (health/Kube, palette/multi-select/events, Quadlet/auto-update) get their own design docs.

## Intent and why

After the Docker→Podman port, lazypodman only speaks the Docker-compatible API. Pods are the thing Podman has that Docker does not, and they are still invisible: a user running `podman pod create` sees containers but not the pod, and has to leave the TUI for every pod operation. That makes lazypodman "lazydocker with a different socket", which is exactly what this fork should stop being.

Goal: a user never leaves the TUI for list / inspect / start / stop / restart / pause / remove / logs / stats of a pod, and can see in the Containers panel which pod a container belongs to.

## Goals

- Pods panel: status, name, container count, infra-container indicator, shared ports.
- Detail tabs: `Config` (pod inspect), `Containers` (the pod's containers), `Stats` (aggregated CPU/RAM), `Logs` (merged, per-container prefix + color).
- Actions: start, stop, restart, pause/unpause, remove (plain and `--force`), prune.
- Pod membership visible in the Containers panel as a column, toggleable in config.
- Every Podman-native feature degrades gracefully (hidden, with a reason in the log) when the `podman` binary or the needed sub-command is missing.

## Non-goals

- No Libpod bindings and no change to the existing socket backend.
- No pod creation from the UI; no Kube/Quadlet/secrets/machine/auto-update (later phases).
- No cached or persisted pod inventory across restarts.
- No configurable-keybindings engine; only the panel focus keys get a single source of truth (approved, limited refactor).
- No grouping/reordering of the Containers list by pod (column only, v1).

## Anti-goals (failure even if the feature works)

- A pod command hitting a different endpoint than the socket backend (e.g. remote/ssh endpoint while CLI talks to the local store) and silently acting on the wrong state.
- Blocking the UI goroutine while streaming logs/stats, or leaving orphaned `podman logs` processes after switching tabs.
- An empty Pods panel when the real cause is a parse failure or a missing binary.
- Removing pods without honouring the existing confirmation setting.

## Constraints

- Vendor-only builds (`GOFLAGS=-mod=vendor`); **no new dependencies** — the CLI wrapper uses only stdlib.
- Go 1.22 module, CI on 1.24; Windows and darwin must keep compiling.
- New user-visible strings only in `pkg/i18n/english.go`; other languages fall back to English.
- Existing patterns stay: `SideListPanel[T]`, `MainTab`, `presentation.GetXDisplayStrings`, `tasks.TaskFunc`.
- Sensitive values: pod inspect output may contain env vars of the pod spec; mask nothing yet but never log values (log only names/IDs).

## Approaches considered

**A. Add `github.com/containers/podman/v5` bindings (Libpod REST).**
Typed pod structs, correct `Pod`/`PodName` fields, streaming endpoints.
Trades off: tens of MB of vendored code and a large dependency graph (`containers/storage`, `containers/image`, `opencontainers/*`) version-locked against `moby/moby` already in `go.mod`; months of upgrade churn for a TUI. Rejected.

**B. CLI JSON wrapper behind a `PodmanClient` interface (chosen).**
`podman pod ls --format json`, `podman pod inspect`, `podman pod logs`/`podman logs`, `podman pod stats --format json`, `podman ps --format json`.
Trades off: requires the `podman` binary and a JSON shape that drifts slightly between versions; one extra process per refresh. In exchange: no dependency risk, works with any Podman ≥ 3, and every limitation is explicit and local.

**C. Docker-compatible API only.**
There are no pod endpoints under `/vX.Y/containers|images|...`; pods exist only under `/libpod`. Not possible without A. Rejected.

Chosen: **B**, per the decision already made.

## Decisions taken (from the conversation, not open)

1. Backend is the CLI wrapper behind `PodmanClient`, with tolerant JSON parsing, golden fixtures, and a fake `podman` script for wrapper tests.
2. Panels renumber: Project=1, Services=2, **Pods=3**, Containers=4, Images=5, Volumes=6, Networks=7. Panel focus keys live in one place; in-app help, README, `docs/Config.md`, keybinding tests and a new `CHANGELOG.md` are updated together.
3. Streaming commands use `exec.CommandContext` with kill-on-close.
4. Verification: skip-if-no-socket integration tests; `-race` in CI only; a human checklist is handed over and the PR states "verified by human" vs "not verified" per item.

## How this coexists with the socket backend

Two backends, one process, no fallback between them:

- **Socket (Docker-compatible API)** — unchanged, still the source for containers, services, images, volumes, networks, stats, logs, events.
- **CLI (`PodmanClient`)** — new, used only for pod-native data: pod list/inspect/actions/logs/stats and the container→pod map.

**Enablement rule (v1).** Podman-native features are enabled only when all of these hold at startup:

1. a `podman` binary is found (config `commandTemplates.podman`, else `podman` on `PATH`);
2. `podman --version` runs within a 5 s timeout and reports ≥ 3.0;
3. the CLI targets the same endpoint as the socket: `podman info --format '{{.Host.RemoteSocket.Path}}'` equals the socket path lazypodman detected (`determinePodmanHost()`), and neither `CONTAINER_HOST`, `PODMAN_HOST` nor `DOCKER_HOST` is set to a remote transport (`ssh://`, `tcp://`) that the local CLI cannot reach.

Detection runs inside `NewPodmanCommand` (app startup), i.e. before views, panels and keybindings are built, so the panel list and the focus keys are fixed before the first frame.

*Builder call:* if `RemoteSocket.Path` is empty (Podman in `--remote` mode, where the CLI has no local store to compare against), treat the check as failed and disable the feature rather than guessing.

If any check fails, the Pods panel is not registered, the pod column is hidden, focus keys compress back to the current layout (1 Project, 2 Services, 3 Containers, …), and exactly one debug line explains why (`Log.Debug`), with no startup error popup.

**The "socket-only" run mode and its limitation.** Running lazypodman in a container with only `/run/podman/podman.sock` mounted (the supported remote mode) has no `podman` binary, so rule 1 fails and everything pod-related is off: no Pods panel, no pod column, focus key 3 falls back to Containers. The same applies to `CONTAINER_HOST=ssh://…`: the SSH tunnel gives us the socket but pod commands would run on the local store, which is worse than hiding the feature. This limitation is documented in README (Pods section) and in the PR description. Containers/services/images/volumes/networks are unaffected in both cases.

## Components and files

### New: `pkg/commands/podman_client.go`

```go
type PodmanClient interface {
    Enabled() bool        // detection passed
    Reason() string       // why it is disabled, for the debug log
    Version() string
    PodList(ctx context.Context) ([]*Pod, error)
    PodInspect(ctx context.Context, name string) ([]byte, error)
    PodAction(ctx context.Context, verb string, name string, extra ...string) error
    PodPrune(ctx context.Context) error
    PodLogs(ctx context.Context, containers []string, tail int) (io.ReadCloser, error) // merged stream; Close() kills the children
    PodStats(ctx context.Context) ([]PodStatsRow, error)
    ContainerPodMap(ctx context.Context) (map[string]string, error) // container ID -> pod name
}
```

- `cliPodmanClient` implements it with `exec.CommandContext`; `cmd.Cancel = func() error { return cmd.Process.Kill() }`, `cmd.WaitDelay = 2 * time.Second`.
- All argv is built with explicit args (never a shell string). Names are passed as separate argv entries, so spaces and quotes are safe.
- `PodLogs` returns a reader over an internal fan-in owned by the client: one `podman logs --follow --tail=<n>` per container, one goroutine per reader, lines tagged with the container name by the consumer. The returned `ReadCloser.Close()` cancels the context and waits for every child process, so the GUI only has to close the reader when leaving the tab or quitting. The GUI never starts a process for logs.

### New: `pkg/commands/pod.go`

- `Pod` struct (minimal, tolerant):

```go
type Pod struct {
    ID                 string            `json:"Id"`
    Name               string            `json:"Name"`
    Status             string            `json:"Status"`
    Created            string            `json:"Created"` // kept raw: RFC3339 on some versions, human text on others
    NumberOfContainers int               `json:"NumberOfContainers"`
    InfraID            string            `json:"InfraId"` // empty when the pod was created with --infra=false
    HasInfra           bool              `json:"-"`       // derived: InfraID != ""
    Labels             map[string]string `json:"Labels"`
}
```

- `parsePods([]byte) ([]*Pod, error)`: tolerates empty output, `null`, `[]`, and unknown fields; returns a descriptive error instead of an empty slice on malformed JSON.
- `PodStatsRow` from `podman pod stats --format json --no-stream` (pod name, CPU %, mem usage/limit); `ContainerPodMap` from `podman ps --format json` (`Id`, `Pod`, `PodName` — the Docker-compatible API returns `PodName` empty, which is why this uses the CLI).
- Infra detection: `InfraId != ""` in the pod list (display-only marker); the infra container is never listed as a pod member or a "real" container.

### New: `pkg/gui/pods_panel.go`, `pkg/gui/presentation/pods.go`

`SideListPanel[*commands.Pod]` with tabs in this order: `Logs`, `Stats`, `Containers`, `Config`.

- Sort: running → degraded → paused → created → exited, then name (mirrors container sort).
- Columns: status (colored), name, `N containers`, infra marker, ports (shared ports of the pod's containers, from the existing container list, not a new API call).
- Logs tab: merged follow stream, default tail 200, at most 8 containers multiplexed (beyond that, show a message and stream only the running ones).
- Stats tab: the pod-level aggregate comes from `podman pod stats --format json --no-stream` polled every 2 s inside a task; the per-container rows reuse the **existing socket stats** of the pod's member containers, so there is only one source of truth for container numbers. Rendered with the existing asciigraph helpers. If `podman pod stats` is unavailable on the installed version, the tab shows the aggregate computed by summing the socket stats instead (capability probe once at startup).
- Containers tab: filter of the existing container panel data by pod membership.
- Config tab: pretty-printed `pod inspect` JSON.

### New: `pkg/gui/keys.go`

Single source of truth for panel focus keys:

```go
type panelKey struct{ Key rune; View string; Tr func(*i18n.TranslationSet) string }
var panelKeys = []panelKey{{'1', "project", ...}, {'2', "services", ...}, {'3', "pods", ...}, ...}
```

- `views.go` sets `TitlePrefix` from it; `keybindings.go` builds the go-to bindings from it.
- When Pods is disabled, the list yields 6 entries and everything shifts back by one — no duplicated key logic.
- In-app bottom line derives from the binding table already, so it stays correct automatically; the test asserts the rendered prefixes and keys match `panelKeys`.

### Changed files

| File | Change |
| --- | --- |
| `pkg/commands/podman.go` | hold `PodmanClient`, construct once at startup, expose `PodmanCommand.Pods()` |
| `pkg/commands/podman_containers.go` | merge `ContainerPodMap` into container summaries (`PodName` field) |
| `pkg/gui/gui.go` | `Panels.Pods`, `setPanels`, `refresh()` goroutine, key registration |
| `pkg/gui/views.go` | `Views.Pods`, title/prefix from `panelKeys` |
| `pkg/gui/view_helpers.go` | `allSidePanels()` includes Pods when enabled |
| `pkg/gui/keybindings.go` | pod bindings (`d`, `s`, `S`, `r`, `p`, `b` prune, `c` custom command) + go-to keys from `panelKeys` |
| `pkg/gui/presentation/containers.go` | optional pod column |
| `pkg/gui/containers_panel.go` | pass pod name into presentation; keep sort untouched |
| `pkg/config/app_config.go` | `gui.showPodColumn` (default true) |
| `pkg/i18n/english.go` | `PodsTitle`, `NoPods`, `FocusPods`, `StartPod`, `StopPod`, `PausePod`, `ContainersInPod`, `PodLogs`, `PodStats`, `PrunePods`, `ConfirmPrunePods`, … |
| `pkg/cheatsheet/generate.go` | title map entry for `pods` |
| `docs/Config.md`, `README.md`, `CHANGELOG.md` (new), `docs/keybindings/*` | keys, config option, feature description, changelog entry |

No changes to `arrangement.go`: the layout is derived from `allSidePanels()`.

## Behaviour details

- **Destructive actions** (`remove`, `remove --force`, `prune`) go through the existing confirmation panel and the existing `gui.returnImmediately`/confirmation setting.
- **Prune** lists what will be pruned (pod names) before confirming; `podman pod prune` only removes stopped pods, so the confirmation says exactly that.
- **Refresh** runs in its own goroutine next to the other reloads; a failure keeps the last good list and shows the error in the status bar, never an empty panel.
- **Custom command** support extends `commands.CommandObject` with `Pod`, so `{{ .Pod.Name }}` works in `commandTemplates` without touching the template engine.
- **Stale selection**: selecting a pod that disappeared between refresh and action returns the "no items" path, no crash.

## Testing

- **Golden fixtures**: `pkg/commands/testdata/pods/{v4.9.json,v5.2.json,empty.json,null.json,malformed.json}` + `testdata/pods/stats_v5.json`; table test asserts the normalized struct and that unknown fields are ignored.
- **Fake `podman` script**: test writes a temp dir with an executable `podman` (skipped on Windows) and points `PATH` at it (`t.Setenv`); the script records its argv to a file and prints the fixture. Asserts: correct argv for list/inspect/action, exit-code handling with podman's stderr surfaced in the error, context cancellation kills the child (no leak, `WaitDelay` respected).
- **Unit**: sort order, presentation cells, pod column toggle, `panelKeys` mapping with and without Pods, `ContainerPodMap` merge.
- **Integration** (`//go:build integration` + automatic skip when `podman` binary or socket is absent): real pod create/start/stop/remove round-trip.
- **CI**: `go test -race ./...` (ubuntu); local runs use `go test ./...` because this environment has no gcc.
- **Cheatsheet**: `go run scripts/cheatsheet/main.go generate` is the docs/keybinding test; CI already checks it is up to date.

## Manual verification checklist (human, rootless Podman)

1. `podman pod create --name demo -p 8080:80`
2. `podman run -d --pod demo --name web docker.io/library/nginx`
3. `podman run -d --pod demo --name side busybox sleep 600`
4. Start lazypodman → Pods panel visible, `demo` listed, `2 containers`, ports `8080->80/tcp`.
5. Containers panel shows `demo` in the pod column; toggle `gui.showPodColumn` off → column gone.
6. Tabs: Logs shows interleaved `web`/`side` lines with prefixes; Stats shows totals + per-container rows; Containers shows the two members; Config shows the inspect JSON.
7. Actions: `s` stop, `S` start, `r` restart, `p` pause/unpause, `d` remove (plain, then `--force`), `b` prune with a stopped pod.
8. Focus keys: `1`…`7` match the new order; `[` / `]` cycle tabs; bottom line help matches.
9. Remote/socket-only mode: `CONTAINER_HOST=ssh://…` and a container with only the socket mounted → Pods panel hidden, Containers still works, exactly one debug line explains why.
10. Version guard: temporary `podman` wrapper that prints `podman version 2.2.1` → Pods hidden.
11. Record for each item: **verified by human (podman <version>)** or **not verified**; paste into the PR.

## What is left to the builder

- Tab order/labels (proposal above) may be adjusted while implementing if the panel gets cramped.
- Polling interval (2 s) and log tail (200) are defaults; a config knob is only added if a reviewer asks.
- Colour assignment for container prefixes: deterministic hash of the container name over the existing theme colours.
- Whether the pod column is on by default: proposal is yes, since it is cheap and it is the feature's point.

## Suggested commit sequence

1. `feat(podman): add a CLI-backed PodmanClient with startup detection`
2. `feat(pods): add the pods panel, tabs and actions`
3. `feat(containers): show pod membership column`
4. `refactor(gui): single source of truth for panel focus keys`
5. `test(pods): golden fixtures, fake podman wrapper, integration tag`
6. `docs(pods): README, Config.md, keybindings, CHANGELOG`
