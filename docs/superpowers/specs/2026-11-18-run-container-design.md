# Run container from an image — design

## Intent

Let a user start a new container from the Images panel without remembering
`podman run` flags. The selected image is the default; every value can be
changed from a menu, and doing nothing runs that image with Podman's defaults
plus detached mode.

## Entry points

- `r` on the Images panel opens the run menu immediately.
- The same action is the first item in the `x` options menu (bindings are
  listed in declaration order, so the images binding is declared before the
  existing ones).

## Interaction

A menu, not a full-screen form:

- Root rows: image, name, command, entrypoint, workdir, user, hostname, ports,
  volumes, env, labels, network, restart, detach, auto-remove, interactive,
  tty, advanced groups, run.
- `enter` edits a text field or opens a choice/list submenu.
- `space` toggles booleans.
- Lists open a submenu with one row per value and an add row. Selecting a value
  opens edit/delete. This needs no extra keybindings and cannot collide with
  panel keys.
- After every edit the previous menu is reopened, so the user keeps context.
- `esc` closes the menu and returns to the Images panel.

## Field coverage

Fields are schema-driven. The schema is the single source of truth for menu
labels, flags and argv order, which keeps the menu and the command in sync.

Core fields cover image, name, command, entrypoint, workdir, user, hostname,
ports, volumes, env, labels, network and restart, plus detach/rm/interactive/tty.

Advanced groups cover network/DNS, storage, resources, security, healthcheck,
runtime/image flags and extra args. Every field maps to a real `podman run`
flag; anything not modelled is reachable through **Extra args**.

## Command construction and execution

- Build `[]string` argv directly; never build a shell string for execution.
- Flags come from the schema, repeated for list values. The image ref comes
  from the image's first repo tag, falling back to its ID. Command words are
  parsed with the existing argv splitter and placed after the image. Extra args
  are parsed the same way and placed before the image.
- Defaults: detach on, auto-remove off, interactive off, tty off, pull missing
  (Podman default), no name, port, volume, env or label changes.
- Before running, show the exact command in a confirmation panel. Confirming
  runs `podman run` without blocking the UI. Detached mode collects combined
  output; attached mode suspends the TUI. A failure shows the output verbatim;
  success refreshes containers and services.
- The command is previewed with readable quoting. Secrets are not logged.

## Non-goals

- Saving or reusing run presets.
- Parsing an arbitrary existing command line back into fields.
- Changing the Images panel's other actions.

## Constraints

- Podman-only, rootless first.
- New strings go in `pkg/i18n/english.go` only.
- No new dependencies; `mgutz/str` and `gocui` already exist.
- Tests pin argv construction, defaults, list edits, menu ordering and schema
  integrity. Docs and the generated cheatsheet are updated.
