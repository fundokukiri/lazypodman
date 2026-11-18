# Batch 1 — Healthcheck + per-feature capabilities (design)

Status: awaiting approval. Phase 2 of the Podman-native roadmap; kube is batches
2a/2b and gets its own design.

## Intent and why

The containers panel already colours `healthy` / `unhealthy` / `starting` from
`Details.State.Health`, but a user whose probe flaps sees a colour and nothing
else: not which probe result failed, not when, not the probe's own output. Podman
also has no key in lazypodman to re-run the probe, so verifying a fix means
dropping to a shell.

Batch 1 adds both: a **Health** tab built from data the socket already returns
(no new podman process), and a `t` key that runs `podman healthcheck run` on the
selected running container.

It also replaces the single "pods panel enabled" gate with a **capability
struct**, because healthcheck-run and kube each need the CLI for different
reasons. Without that, hiding or showing features would be guesses.

## Goals

- Health tab per container: status, failing streak, last probes (time, exit
  code, output), newest first.
- `t` runs the probe once; the result panel distinguishes *unhealthy* (a result)
  from *cannot run* (a failure).
- One capability object, computed once at startup, exposing for each feature
  whether it is available and why not.
- Off-reasons keep appearing on the existing bottom-right information line.

## Non-goals

- Creating or editing healthchecks (`--health-cmd`, `--health-interval`,
  `--health-retries`). Read-only.
- Pod-level or service-level health aggregation.
- Kube (2a/2b), Events, palette.
- Historical health graphs.

## Approaches considered

1. **Socket tab only, no trigger.** Cheapest, but the user still cannot re-run a
   probe, which is the reason they open a health tab.
2. **New CLI call for the log** (`podman inspect --format '{{json .State.Health}}'`).
   Duplicates data the socket already has and costs a process per refresh.
3. **Socket tab + CLI trigger** (chosen). One new process only when the user
   presses `t`, and none while browsing.

## Capabilities

New `pkg/commands/capabilities.go`:

```go
type Capability struct {
    Available bool
    Reason    string // short code for the "off: <reason>" line
    Detail    string // podman's own message, for the debug log
}

type Capabilities struct {
    CLI          Capability // binary + parseable version + same store
    Pods         Capability // CLI && gui.pods.enabled != never
    HealthRun    Capability // CLI && `podman healthcheck run --help` works
    KubeGenerate Capability // batch 2a
    KubePlay     Capability // batch 2b
}
```

- The existing `CLICheck`/`detectCLI` already answers `CLI`; it becomes the one
  probe everything else derives from. `Pods` keeps today's semantics exactly
  (`PodsMode` still applies), so Phase 1 behaviour does not change.
- **Healthcheck capability is probed lazily**, on the first `t` press, then
  cached. Rationale: we just cut startup to two podman processes and will not add
  a third for a feature most sessions never use. Until probed, `HealthRun` is
  "unknown": `t` shows a waiting status, runs the probe, then either the result
  or `health run off: <reason>`.
- Kube capabilities are probed once in batch 2a/2b, with the documented fallbacks
  (`podman kube play|generate` then `podman play kube|generate kube`). No version
  numbers are hard-coded anywhere.
- Concurrency: `Capabilities` is built once before the GUI starts and only the
  healthcheck probe mutates it later, under one mutex with single-flight, so two
  fast `t` presses run one probe, not two. Readers take a copy.
- `PodmanClient` gains `HealthcheckRun`; the fakes in `pkg/commands` and
  `pkg/gui` tests grow the method, which is the only interface change in this
  batch.
- Off-line rule: when the CLI is unavailable, the information line shows a single
  `cli off: <reason>` (pods, health and kube all share it). When the CLI is fine
  but one feature fails its probe, that feature adds its own
  `health run off: <reason>`. At most one CLI-level line plus one per failed
  feature; no five-line wall.

## Health tab

Position: after **Stats** (tab order Logs, Stats, Health, Env, Config, Top).

Rendering, from `container.Details.State.Health` (socket, refreshed by the
existing 1 s details loop):

```
Healthy · failing streak 0 · 12 probes
TIME      EXIT  OUTPUT
10:42:01  0     probe ok
10:41:31  1     dial tcp 127.0.0.1:8080: connect: refused
```

- Newest first, last 20 entries (`Log` is oldest-first and bounded by podman).
- Timestamp from `End` (falls back to `Start`), local `HH:MM:SS`; a zero time
  renders `--:--:--` rather than a 0001 date.
- Output: newlines become `⏎`, runes truncated at 200 with `…`; empty output
  shows `(no output)`.
- Exit 0 green, 1/2 red, other codes yellow (probe error).
- No healthcheck (`Health == nil` or `Status == none`): the tab says
  `This container has no healthcheck`, no error.
- A ticker task (1 s) re-reads the already-in-memory `Details`, so it stays live
  without I/O. No new goroutine per tab beyond the existing ticker pattern.

## `t` — run the probe once

- Key choice: **`t`** ("test"). Verified free: `h`/`l` are previous/next panel on
  every side panel, `k`/`j` are list up/down, `H`/`K`/`J`/`L` are global main
  scrolling, `x`/`X`/`q`/`?` are global. `t` is unbound globally and on
  containers. Kube's future `k` is taken by list-up, so batch 2 will pick another
  key and say which.
- Preconditions checked **before** any process:
  - not running (`Container.Container.State != "running"` or
    `Details.State.Running == false`) → message `Container is not running`.
  - no healthcheck → message `This container has no healthcheck`.
  - CLI/HealthRun unavailable → `health run off: <reason>`.
- Run through a new `PodmanClient.HealthcheckRun(ctx, ref)` returning
  `(output string, exitCode int, err error)`. `exec.CommandContext`, no UI
  goroutine, `gui.WithWaitingStatus(Tr.RunningHealthcheckStatus)` while it runs.
- Result classification (the point of the requirement):
  - exit 0 → popup titled **healthy**, body = probe output.
  - exit 1 or 2 → popup titled **unhealthy**, body = probe output. This is a
    *result*, shown exactly like healthy, not an error.
  - process could not run / other exit code → error panel with podman's stderr
    verbatim.
- Destructive-action rule does not apply: `t` changes no state, so no
  confirmation prompt (same as restart/pause today). It refreshes container
  details afterwards so the list colour updates too.

## Config

No new keys in batch 1. `gui.pods.enabled` keeps its meaning for the pods panel
only.

## Files

| File | Change |
| --- | --- |
| `pkg/commands/capabilities.go` (+ test) | `Capability`, `Capabilities`, derivation from `CLICheck`, lazy healthcheck probe (single-flight) |
| `pkg/commands/podman.go` | hold `Capabilities`, expose `Capabilities()`; `CLICheck` retained |
| `pkg/commands/podman_cli.go` (+ test) | `HealthcheckRun`; probe helper for `healthcheck run` |
| `pkg/gui/keys.go` | capability-aware helpers replacing `PodsEnabled` for pods only |
| `pkg/gui/pods_panel.go`, `views.go` | read capabilities instead of `Pods()` |
| `pkg/gui/containers_panel.go` (+ new `health_tab.go`) | Health tab, `t` handler |
| `pkg/gui/presentation/health.go` (+ test) | table rendering, truncation, exit-code colours |
| `pkg/gui/keybindings.go` | `t` binding + description |
| `pkg/i18n/english.go` | health tab/result/reason strings |
| `docs/keybindings/*` | regenerated |
| `README.md`, `docs/Config.md`, `CHANGELOG.md` | key + tab + off-line description |
| `pkg/commands/testdata/health/` | synthetic health-log fixtures + README stating SYNTHETIC |

## Tests

- `parseHealthLog` / renderer: synthetic entries covering exit 0/1/2/other,
  multiline output, empty output, 250-char output, unknown timestamps, zero
  entries, 20-entry cap.
- Capability derivation: CLI down → Pods/HealthRun/Kube off with the CLI reason;
  `pods.enabled: never` → Pods off, HealthRun still fine; lazy probe success and
  failure (missing subcommand).
- `t` handler: fake client returning `(out, 0)`, `(out, 1)`, `(out, 125)`, and an
  `exec.ErrNotFound`-style error, asserting popup vs error panel classification,
  plus the "not running" and "no healthcheck" short-circuits (no call made).
- Key table test: `t` appears exactly once for `containers` and nowhere else.
- All new tests use synthetic data; nothing here is captured from a real Podman.

## Manual checklist additions (batch 1)

Appended to the Phase 1 checklist, same three-level reporting:

1. `podman run -d --name hc-ok --health-cmd 'exit 0' --health-interval 2s docker.io/library/busybox sleep 600` → tab shows probes with exit 0, status healthy; `t` → healthy popup.
2. `podman run -d --name hc-bad --health-cmd 'exit 1' --health-interval 2s …` → tab shows exit 1 rows in red; `t` → **unhealthy popup, not an error**; list colour follows.
3. Container without a healthcheck → tab says there is none; `t` shows the same message and spawns no process.
4. Stopped container with health history → tab still lists the last probes; `t` says it is not running.
5. `pgrep -a podman` before/after pressing `t`: one process appears, none remains afterwards.
6. Socket-only setup (`CONTAINER_HOST=ssh://…`): no `t` effect beyond `cli off: <reason>`; Health tab still renders from inspect.

## Verification levels

- **verified by human (podman x.y)** — only checklist items a person actually ran.
- **tested against synthetic fixtures only** — everything in this batch until
  then; Podman's health log shape is documented but not captured here.
- **not verified** — anything else, including whether `podman healthcheck run`
  exists on the user's version; that is exactly what the lazy probe decides.

## Left to the builder

Exact i18n wording, table column widths, whether the result popup reuses
`createPopupPanel` or a slimmer helper, and the tab's empty-state colour.
