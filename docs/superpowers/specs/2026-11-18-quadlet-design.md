# Phase 4 — Quadlet export and `podman auto-update`

**Status:** implemented and committed (`3552f13`). Design record for what shipped;
behaviour stays **unverified on a real Podman** until `TEST.md` sections G–H are run.

## Intent

Two Podman-native workflows that end in Podman itself rather than in lazypodman:

1. A container that should survive a reboot can be exported as a **Quadlet**
   `.container` unit (the modern replacement for `podman generate systemd`).
2. Containers that carry the auto-update label can be updated from the bulk menu.

Both are opt-in and confirmed; nothing runs or is enabled in the background.

## 4a. Quadlet export (`Q` on Containers)

**Why generate the file ourselves:** Podman has no `podman quadlet generate`
subcommand. The alternative — shelling out to `podman generate systemd` — emits a
unit in the old style, needs a container, and is being deprecated. Building the
`.container` text from inspect data keeps the feature CLI-free, testable and
reviewable, at the cost of covering only the common fields. That trade-off is
documented here and in the README.

**Mapping (container inspect → Quadlet):**

| Inspect | Quadlet key | Notes |
| --- | --- | --- |
| `Config.Image` (fallback `Container.Image`) | `Image=` | |
| `Container.Name` | `ContainerName=` | |
| `Config.Env` | `Environment=` | sorted, space-separated; **plaintext** |
| `Config.Cmd` | `Exec=` | joined with spaces, no shell quoting (limitation) |
| `HostConfig.PortBindings` | `PublishPort=host:container` | one line per binding, sorted |
| `Mounts[].Source:Destination` | `Volume=` | bind mounts only; named volumes appear as their source name |
| `HostConfig.NetworkMode` | `Network=` | raw value, skipped when empty |

Always appended: `[Unit] Description=`, `[Service] Restart=always`,
`[Install] WantedBy=default.target`.

**Write path:** `gui.quadletDir` (default `<user config dir>/containers/systemd`, the
directory Podman reads user units from; `~` expanded). File name is
`SafeKubeFilename(containerName) + ".container"` — the same single-segment sanitiser
used by the kube export, so `../../etc/passwd` cannot escape the directory.

- Preview in the main panel first, then a confirmation containing the **full path**,
  the overwrite state, and a plaintext-env warning (always shown, not only when env
  exists).
- `WriteManifestFile(path, data, true)` → directory `0755`, file `0600`, chmod after
  the write. Overwrite only after the confirmation.
- `systemctl` is never called: enabling/starting the unit is the user's decision.
- Details are refreshed on demand if the container has not been inspected yet;
  inspect failure shows the error, no partial file is written.

**Non-goals:** translating every inspect field, `[Pod]`/`.pod` units, kube YAML →
Quadlet conversion, enabling units, timers, secrets, `Rootfs=`, healthcheck
translation, resource limits.

**Known limitations:**
- `Exec=`/`Environment=` are space-joined, so an argument containing a space would
  split. Fixing this properly needs quoting rules from Quadlet; not invented here.
- `Network=` emits the raw engine value (`bridge`, `host`, a network name); Quadlet
  accepts the simple cases, unusual values may need manual editing.
- The generated file is a readable starting point; the app does not validate it.

## 4b. `podman auto-update`

Entry: Containers bulk menu (`b`) → **Auto-update containers**. It lives in the bulk
menu because auto-update acts on **every container carrying the auto-update label**,
not on the selection or the marks — putting it on a single-container key would
misrepresent its scope.

Flow:

1. If the CLI capability is `Unavailable`, flash `cli off: <reason>` and stop.
2. Confirmation `Run podman auto-update?`.
3. `StreamPodman(ctx, ["auto-update"], onLine)` inside a task; header
   `# podman auto-update`; output streamed verbatim; errors appended as text.
4. Refresh containers/services when the stream ends (success or not, unless the
   context was cancelled).
5. Cancelling the task kills the child via the task context.

**Non-goals:** a scheduled auto-update, per-container update, `--dry-run` toggle,
parsing the result into a summary, restarting dependent pods.

**Known limitation:** the output format of `podman auto-update` differs across
versions; showing it verbatim is deliberate, and no success/failure is claimed from
the text (only the process exit code would be authoritative, and the stream shows
whatever Podman prints).

## Edge cases and degradation

| Situation | Behaviour |
| --- | --- |
| No CLI / socket only | Quadlet export still works (inspect data); auto-update flashes `cli off:` |
| Container without inspect details | Refreshed on demand; error shown if that fails |
| Existing unit file | Overwrite prompt; Cancel keeps the old file |
| No container has the auto-update label | Stream shows Podman's own output; nothing is updated |
| Task cancelled mid-stream | Child killed, no success message |

## Verification status

| Item | Level |
| --- | --- |
| Unit text mapping, name sanitiser, prompt contents | tested against synthetic fixtures only (unit) |
| Directory default/expansion | tested against synthetic fixtures only (unit) |
| Quadlet file accepted by `systemctl --user` | not verified (TEST.md G4) |
| Auto-update menu + stream on a real Podman | not verified (TEST.md H1–H4) |
