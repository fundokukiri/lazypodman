# Batch 2 — Kube generate / play / down

**Status:** implemented and committed (`c3d6049`, `41fe5d5`). Same rules as batch 1: lazy
capabilities, no UI goroutine blocking, graceful degradation, synthetic tests until
a human records real output.

## Intent

Let a pod be exported as a Kubernetes YAML file and let the YAML files already on
disk be applied (or torn down) without leaving lazypodman. This is the Podman-native
workflow "pod → YAML → pod" that `podman kube` exists for.

## Capabilities and fallbacks

Two lazy capabilities, probed on first use of the kube menu:

| Feature | first probe | fallback |
| --- | --- | --- |
| generate | `podman kube generate --help` | `podman generate kube --help` |
| play/down | `podman kube play --help` | `podman play kube --help` |

The probe stores the working argv prefix (`["kube","generate"]` or
`["generate","kube"]`, `["kube","play"]` or `["play","kube"]`) so later calls use
the syntax the installed Podman actually accepts. No version numbers are hard-coded.
Down is the play prefix plus `--down`.

A probe failure is shown as `kube generate off: <reason>` / `kube play off: <reason>`
on the information line, but only after a probe really failed. A timeout is
transient and retried; "unknown subcommand" is cached for the session.

## Generate (2a)

Pods panel → `g` → menu item **Generate kube YAML**.

1. Probe the capability; if off, flash the off reason, spawn nothing.
2. Run `<prefix> <pod>` (no `--podman-only`, no extra flags) through
   `KubeGenerate(ctx, prefix, pod)`, capturing stdout and folding stderr into the
   error.
3. Show the YAML in the main panel as a preview (it is plain text, not a popup).
4. Ask for confirmation with the full destination path and a warning that the file
   contains environment variables in plaintext and no Secrets.
5. Write `<gui.kubeDir>/<pod-name>.yaml` with mode `0600`, creating the directory
   with `0755` if needed. An existing file gets an explicit "overwrite?" prompt;
   cancelling leaves it untouched.
6. Flash the saved path.

The pod name is sanitised for the filesystem (`[^A-Za-z0-9._-]` → `-`) so a pod
name can never escape the directory.

## Play / down (2b)

Pods panel → `g` → **Play kube YAML…** or **Down kube YAML…**.

- Files: scan `gui.kubeDir` non-recursively for `*.yaml` / `*.yml`, sort by name,
  and keep only files whose first 20 non-empty/non-comment lines contain a
  top-level `kind:` (`Pod`, `Deployment`, `DaemonSet`…). Anything else is not a
  kube manifest and is not offered.
- Never auto-run. Selecting a file always asks a confirmation that contains the
  **full path**; for down the prompt says the pods will be stopped and removed.
- The run is a normal task: output is streamed verbatim into the main panel
  (`stdout` and `stderr` merged), the command is bound to the task context so a
  newer task cancels it, and there is no short enablement timeout. No `--replace`,
  no `--build` in this iteration.
- A missing `gui.kubeDir` is not an error popup: the menu/flash says which path is
  missing, and generate still works (it creates the directory).

## Config

```yaml
gui:
  kubeDir: ""   # default: <user config dir>/lazypodman/kube
```

`~` is expanded. The directory is both the save target for generate and the source
for play/down.

## Security notes

- Files are written `0600`; the directory `0755`.
- The YAML contains env vars in plaintext (Kubernetes has no env encryption here).
  The confirmation says so; no Secrets are created by default.
- Nothing is written to, or run from, a path the user did not see in full.

## Files

| File | Change |
| --- | --- |
| `pkg/commands/kube.go` | probes, `KubeGenerate`, `StreamKube`, capability methods |
| `pkg/commands/kube_files.go` | directory scan, `kind:` sniff, sanitise name, safe write |
| `pkg/commands/podman_client.go` | three new interface methods |
| `pkg/commands/podman_cli.go` | CLI implementations |
| `pkg/gui/kube_menu.go` | `g` menu, generate/play/down handlers, streaming task |
| `pkg/gui/pods_panel.go`, `keybindings.go` | `g` binding |
| `pkg/gui/views.go` | kube off lines |
| `pkg/config/app_config.go` | `gui.kubeDir` |
| `pkg/i18n/english.go` | new strings |
| docs | README, `docs/Config.md`, `CHANGELOG.md`, keybindings regen |

## Tests

- `ProbeSubcommand`: new syntax works; new fails + old works; both fail (permanent);
  timeout (transient).
- `KubeGenerate`: stdout is returned; stderr is surfaced on failure.
- `StreamKube`: lines arrive in order; cancelling the context kills the child and
  returns promptly.
- `ScanKubeDir`: yaml/yml only, non-recursive, `kind:` required, malformed skipped,
  sorted; missing directory returns an error the UI can show as a message.
- `SafeKubeFilename`: path separators and `..` cannot escape the directory.
- `WriteKubeFile`: mode `0600`, new directory created, overwrite refused without
  permission.
- GUI: `g` is bound exactly once (pods), kube off line appears only after a failed
  probe, generate/play/down choices short-circuit when the capability is off.
- Key table test extended for `g`.

## Manual checklist (human)

1. Create a pod with a container, press `g` → generate → preview shows `kind: Pod`
   → save → file exists with mode `600`.
2. Pod with env vars → the confirmation shows the plaintext warning.
3. Generate again to the same name → overwrite prompt; Cancel leaves the file
   untouched (compare mtime).
4. Play the saved file → output streams, the new pod appears in the pods panel.
5. Down it → confirmation shows the full path; Cancel spawns nothing.
6. A `.yaml` without `kind:` and a file in a subdirectory are not listed.
7. On a Podman without `kube` subcommands, the flash says `kube play off: …` and
   nothing is spawned; if the old syntax is used, playback still works.
8. `pgrep -a podman` before/after generate: one process, none left afterwards.
9. Socket-only setup: both kube features report off and nothing is spawned.

## Verification levels

Everything here is **tested against synthetic fixtures only** until a human runs the
checklist. Exit statuses of `podman kube play`, real YAML shapes and old-syntax
availability are **not verified**.

## Implementation status

Implemented as designed:

- `ProbeSubcommand` stores the working syntax (`["kube","generate"]` or
  `["generate","kube"]`; `["kube","play"]` or `["play","kube"]`) and caches it; a
  timeout is transient, a missing subcommand is permanent.
- `KubeGenerate` and `StreamKube` live on `PodmanClient`; streaming merges stdout and
  stderr line by line and kills the child when the task context is cancelled.
- `ScanKubeDir` / `sniffKubeKind` / `SafeKubeFilename` / `WriteKubeFile` are in
  `pkg/commands/kube_files.go`; `WriteKubeFile` is the only writer and always ends at
  mode `0600`.
- The `g` menu and both flows are in `pkg/gui/kube_menu.go`; `gui.kubeDir` resolves
  the default and `~`.
- Everything here is **tested against synthetic fixtures only** until the manual
  checklist is run; the 9 checklist items are in
  `docs/superpowers/plans/2026-11-18-kube.md`.
