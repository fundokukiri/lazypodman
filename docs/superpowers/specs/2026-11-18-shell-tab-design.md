# Shell tab (approval A: line-oriented shell)

Status: implemented. Verification: **not verified** in a real container (sandbox
cannot start containers); the backend is covered by a fake `podman` script.

## Intent

Let the user run commands inside a container without leaving the TUI. Today `E`
suspends lazypodman and attaches a real terminal (`podman exec -it … /bin/sh`),
which loses the UI state and the other panels.

## Approach picked

Line-oriented shell inside the existing main-panel tab strip, directly after
**Logs**. No PTY and no terminal emulator, so no new dependency. The trade-off is
explicit: no full-screen programs (`vim`, `htop`, `less`), no cursor control and
no shell line editing; `E` stays available for the real terminal.

Considered and dropped: PTY + VT parser (real terminal, interactive apps) for
this round because it would add two dependencies and complex key/resize handling.

## How it behaves

- Tab **Shell** shows `# shell <name>` and a hint until a shell exists.
- `enter` starts `podman exec -i -- <id> /bin/sh` (once per container) and opens
  the command prompt. Later `enter` opens the prompt again.
- Submitting a command echoes `$ <command>` and streams merged stdout/stderr into
  the tab; the view autoscrolls and keeps the last 1000 rows.
- The session survives switching tabs and panels. On app exit every session is
  closed and the child killed.
- A dead shell (container stopped/removed) leaves `# shell exited`; the next
  `enter` starts a new one.
- Without a usable CLI or a running container, the tab explains why instead of
  spawning anything.
- `esc` returns to the container list, as on every other tab.

## Known limitations

- No PTY: prompts may be missing, `ctrl+c` kills the `podman exec` client rather
  than sending SIGINT to the running command, no tab completion or history.
- `enter` on the Shell tab is the only new binding; every other tab ignores it.
