# Batch 1 — Healthcheck Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Execution method chosen by the human partner: **native** (one agent, this session).

**Goal:** Add a pod-level Health tab built from socket inspect data plus a `t` key that runs `podman healthcheck run` on the selected container, behind a tri-state per-feature capability object that replaces the single pods-enabled gate.

**Architecture:** `pkg/commands/capabilities.go` derives `CLI`/`Pods` from the existing startup `CLICheck` and keeps `HealthRun`/`Kube*` in the tri-state `Unknown` until a feature is first used. `HealthcheckRun` lives on `PodmanClient` (CLI, `exec.CommandContext`) and returns `(output, exitCode, error)`; the GUI classifies the code, shows a 10 s non-blocking flash for results and an error panel for failures. The Health tab renders `Details.State.Health` (socket) with a 1 s ticker, so browsing costs no process.

**Tech Stack:** Go 1.22, gocui, Docker-compatible API client (`pkg/commands`), podman CLI for one-shot actions.

**Spec:** `docs/superpowers/specs/2026-11-18-healthcheck-design.md`

## Global Constraints

- Podman-only; never fall back to the Docker daemon.
- Never log or display secret values; probe output is container stdout/stderr only.
- Never block the UI goroutine; no podman call on it.
- New user-visible strings only in `pkg/i18n/english.go`.
- No `podman` process on startup for this batch (capabilities are lazy).
- Exit-code table beyond `0 = healthy` is **UNVERIFIED**; unknown codes are errors.
- Fixtures/tests are synthetic until a human records real output.
- Verification report uses exactly: `verified by human (podman x.y)`, `tested against synthetic fixtures only`, `not verified`.
- Every task ends green: `gofmt -s -l main.go pkg/ scripts/`, `GOFLAGS=-mod=vendor go vet ./pkg/...`, `GOFLAGS=-mod=vendor go test ./...`, `golangci-lint run ./pkg/...`.

## Review Focus

1. Pressing `t` on a container whose inspect lags behind the list (list says running, details say stopped) must not panic or skip silently — show the "not running" message and spawn nothing.
2. A container removed between selection and run (`no such container`) must surface podman's stderr verbatim in the error panel.
3. A probe producing non-UTF8 or very large output must be truncated safely by runes, with no multi-megabyte write to the view.
4. A hanging `podman healthcheck run` must time out, leave the UI responsive, and mark the capability transient (so a later `t` retries).
5. Two rapid `t` presses must run one capability probe and must not interleave two error/result fates (second press reuses the first result or is ignored while running).

---

### Task 1: Capability object and derivation from the CLI check

**Files:**
- Create: `pkg/commands/capabilities.go`, `pkg/commands/capabilities_test.go`
- Modify: `pkg/commands/podman.go` (add `capabilities`, `Capabilities()`, `SetCapability()`)

**Interfaces:**
- Produces:
  - `type CapabilityState int` with `CapabilityUnknown`, `CapabilityAvailable`, `CapabilityUnavailable`
  - `type Capability struct { State CapabilityState; Reason string; Detail string; Transient bool }`
  - `type Capabilities struct { CLI, Pods, HealthRun, KubeGenerate, KubePlay Capability }`
  - `func deriveCapabilities(check CLICheck, mode PodsMode) Capabilities`
  - `func (c *PodmanCommand) Capabilities() Capabilities` (copy under mutex)
  - `func (c *PodmanCommand) SetCapability(feature CapabilityFeature, cap Capability)` where `type CapabilityFeature int` (const `FeatureHealthRun`, `FeatureKubeGenerate`, `FeatureKubePlay`)
- Consumes: `CLICheck`, `PodsMode` from Task 2 of Phase 1.

- [ ] **Step 1: Write the failing test**

`capabilities_test.go`, table:
- enabled check + `PodsAuto` → `CLI`/`Pods` available, `HealthRun`/`Kube*` `CapabilityUnknown`.
- disabled check (`Reason: "podman CLI not found"`) → all five `CapabilityUnavailable`, every `Reason` equals the CLI reason.
- enabled check + `PodsNever` → `Pods.Unavailable` with `"disabled in config"`, `CLI` available, `HealthRun` unknown.
- `SetCapability(FeatureHealthRun, Capability{State: CapabilityUnavailable, Reason: "no healthcheck subcommand"})` then `Capabilities().HealthRun` reflects it, and mutating the returned copy does not change the stored one.

Run: `GOFLAGS=-mod=vendor go test ./pkg/commands/ -run TestCapabilities -v`
Expected: FAIL — `undefined: CapabilityState`.

- [ ] **Step 2: Implement `pkg/commands/capabilities.go`**

`deriveCapabilities` sets `CLI.State` from `check.Enabled`, copies `check.Reason`/`check.Detail` into every derived capability that piggybacks on the CLI, and leaves lazy features `CapabilityUnknown`. `Pods` additionally honours `mode == PodsNever` with reason `"disabled in config"`.

- [ ] **Step 3: Run the tests**

Run: `GOFLAGS=-mod=vendor go test ./pkg/commands/ -run TestCapabilities -v`
Expected: PASS.

- [ ] **Step 4: Wire into `NewPodmanCommand`**

After `SetPodClient(<-cliCheck)`, store `deriveCapabilities(client.check, podsMode)`; keep the debug log but move it to the CLI capability reason.

- [ ] **Step 5: Full check and commit**

Run the four global commands; commit `feat(pods): add per-feature capability states`.

---

### Task 2: Lazy healthcheck probe and `HealthcheckRun`

**Files:**
- Modify: `pkg/commands/podman_client.go` (interface), `pkg/commands/podman_cli.go`, `pkg/commands/podman_cli_test.go`, `pkg/commands/pod_test.go`, `pkg/gui/keys_test.go` (fakes)

**Interfaces:**
- Produces on `PodmanClient`:
  - `HealthcheckRun(ctx context.Context, ref string) (output string, exitCode int, err error)` — spawn/exit failure returns `err`; a real exit status returns its code with `err == nil`.
  - `ProbeHealthcheck(ctx context.Context) Capability` — one `podman healthcheck run --help`; exit status 0 → available; `exec.ErrNotFound` or non-zero → unavailable, `Transient: false`; context deadline → unavailable, `Transient: true`.
- Consumes: `Capability` from Task 1.

- [ ] **Step 1: Write failing tests** with the existing fake-script helper:
- script exiting 0/1/125: `HealthcheckRun` returns the code, output contains stdout+stderr, `err == nil`.
- script killed by a closed process (`FAKE_EXIT` unset but binary removed): `err != nil`.
- `ProbeHealthcheck` against a script printing `--help` text → available; script exiting 2 → unavailable with a non-empty reason; a `hang: true` style stub with a 50 ms budget → `Transient` true.

Run: `GOFLAGS=-mod=vendor go test ./pkg/commands/ -run 'TestHealthcheckRun|TestProbeHealthcheck' -v`
Expected: FAIL — method not defined.

- [ ] **Step 2: Implement**

`HealthcheckRun` validates the ref (reuse `validatePodRef`), runs `podman healthcheck run -- <ref>` with `exec.CommandContext`, a 10 s timeout (`healthcheckBudget`) and returns `(stdout+stderr, exitCode, nil)` for `*exec.ExitError`. Code comment: exit codes beyond 0 are UNVERIFIED.

- [ ] **Step 3: Update the interface and every fake**

`fakePodClient` (commands tests) and `fakeEnabledPodClient` (gui tests) gain both methods. Run `go build ./...` and the package tests.

- [ ] **Step 4: Commit** `feat(healthcheck): run probes and probe the subcommand lazily`.

---

### Task 3: Health log rendering and truncation

**Files:**
- Create: `pkg/gui/presentation/health.go`, `pkg/gui/presentation/health_test.go`

**Interfaces:**
- Produces:
  - `const MaxHealthRows = 20`, `const MaxHealthOutputRunes = 200`
  - `type HealthRow struct { Time string; ExitCode int; Output string }`
  - `func HealthRows(log []*container.HealthcheckResult) []HealthRow` — newest first, capped, `Time` = `End` (or `Start`) as `15:04:05`, zero time `--:--:--`.
  - `func TruncateHealthOutput(output string) string` — newlines/tabs to `⏎`/`→`, rune-safe truncation with `…`, empty → `(no output)`.
  - `func HealthExitColor(code int) color.Attribute` — 0 green, 1/2 red, anything else yellow.
- Consumes: nothing from earlier tasks.

- [ ] **Step 1: Write the failing test** (data marked `// SYNTHETIC`):
- ordering newest-first, cap at 20 from a 25-entry log;
- zero `End` and zero `Start` → `--:--:--`;
- multiline output becomes `a⏎b`;
- 250-rune ASCII truncated to 200 + `…`; 250-rune multibyte string truncates without splitting a rune (assert valid UTF-8);
- empty output → `(no output)`; nil log → empty slice;
- colour map for 0, 1, 2, 125.

Run: `GOFLAGS=-mod=vendor go test ./pkg/gui/presentation/ -run TestHealth -v`
Expected: FAIL — undefined.

- [ ] **Step 2: Implement**, then run the same command → PASS.
- [ ] **Step 3: Commit** `feat(healthcheck): render and truncate probe results`.

---

### Task 4: Health tab, `t` key, flash channel, capability-aware off-line

**Files:**
- Create: `pkg/gui/health_tab.go`
- Modify: `pkg/gui/containers_panel.go`, `pkg/gui/keybindings.go`, `pkg/gui/app_status_manager.go`, `pkg/gui/keys.go`, `pkg/gui/views.go`, `pkg/gui/pods_panel.go`, `pkg/i18n/english.go`

**Interfaces:**
- Produces:
  - `func (gui *Gui) renderContainerHealth(container *commands.Container) tasks.TaskFunc`
  - `func (gui *Gui) handleContainerHealthcheckRun(g *gocui.Gui, v *gocui.View) error`
  - `func (gui *Gui) flashStatus(message string)` — writes the AppStatus line now and clears it after 10 s; a waiting status wins while set.
  - `func (gui *Gui) capabilities() commands.Capabilities`
  - i18n: `HealthTitle`, `NoHealthcheck`, `ContainerNotRunning`, `RunningHealthcheckStatus`, `HealthcheckHealthyResult`, `HealthcheckUnhealthyResult`, `HealthcheckOff`, `CliOff`, `PodsOff`.
- Consumes: `HealthcheckRun`, `ProbeHealthcheck`, `Capabilities`, `HealthRows`, `TruncateHealthOutput`, `HealthExitColor`.

- [ ] **Step 1: Failing tests** (`pkg/gui/health_tab_test.go`, `pkg/gui/app_status_manager_test.go`, `pkg/gui/keybindings_test.go`):
  - classification: `(out,0)`, `(out,1)`, `(out,2)` → flash, no error panel; `(out,125)` and spawn error → error panel (expose `classifyHealthcheck(exitCode int, err error) healthcheckOutcome` pure function and test it).
  - short-circuits: details `Health == nil` → no client call; state not running (list state) → no client call (fake client counts calls).
  - "details lag" case (Review Focus 1): list says running, `Details.State.Running == false` → still no call, message shown.
  - single-flight (Review Focus 5): two concurrent `t` calls → one `ProbeHealthcheck` call (fake counts).
  - flash: waiting status outranks flash; flash expires after its duration (inject a clock).
  - key table: `t` bound exactly once, on `containers`, and not on `""`, `project`, `main`, `menu`, `confirmation`, `filter`, `limit`.

- [ ] **Step 2: Implement** the tab (ticker 1 s reading `Details`), handler, flash, capability-aware `PodsEnabled`, and the off-line segments in `getInformationContent`:
  - CLI unavailable → `cli off: <reason>` only;
  - else pods unavailable (`never`) → `pods off: disabled in config`;
  - else a failed probed feature → `<feature> off: <reason>`; unknown features add nothing.

- [ ] **Step 3: Bind `t`** with description `Tr.RunHealthcheck` (wording states the side effect: the run is recorded in the podman log and can change the health status).

- [ ] **Step 4: Run** `GOFLAGS=-mod=vendor go test ./pkg/gui/... -v` then the four global commands; commit `feat(healthcheck): add the health tab and the run key`.

---

### Task 5: Docs, help text, demo checklist

**Files:** `README.md`, `docs/Config.md`, `CHANGELOG.md`, `docs/keybindings/*` (generated), `pkg/cheatsheet/generate.go` (only if a new section is needed).

- [ ] **Step 1:** Document the Health tab, the `t` key with its recorded-run side effect, the capability off-line rules, and the UNVERIFIED exit-code table.
- [ ] **Step 2:** Regenerate help: `GOFLAGS=-mod=vendor go run scripts/cheatsheet/main.go generate` and `… check` → `Cheatsheets are up to date`.
- [ ] **Step 3:** Append the 7-item manual checklist to this plan (copy from the design), then run the four global commands and `GOOS=windows`/`GOOS=darwin` builds.
- [ ] **Step 4: Commit** `docs(healthcheck): document the tab, key and off-line rules`.

---

## Manual checklist (human, rootless Podman)

Record each line as `verified by human (podman x.y)` or `not verified`:

1. `podman run -d --name hc-ok --health-cmd 'exit 0' --health-interval 2s docker.io/library/busybox sleep 600` → Health tab lists probes with exit 0; `t` → status line says healthy and the tab refreshes.
2. `podman run -d --name hc-bad --health-cmd 'exit 1' --health-interval 2s …` → exit 1 rows red; `t` → **unhealthy result** (status line), not an error panel.
3. `podman run -d --name hc-none docker.io/library/busybox sleep 600` → tab says there is no healthcheck; `t` says the same and spawns nothing.
4. Stop `hc-ok`, reopen its Health tab → last probes still listed; `t` → "not running", no process.
5. `pgrep -a podman` before/after `t` → one process appears, none left afterwards.
6. `CONTAINER_HOST=ssh://…` run → `t` reports `cli off: <reason>`, spawns nothing; Health tab still renders from inspect.
7. Record the real exit codes of `podman healthcheck run` for healthy, unhealthy, no healthcheck and a stopped container; correct the design table and mark it verified.

## Reporting levels

Every claim in the PR/commit summary is reported at exactly one level:
`verified by human (podman x.y)`, `tested against synthetic fixtures only`, or
`not verified`. Nothing here is human-verified when the branch lands.
