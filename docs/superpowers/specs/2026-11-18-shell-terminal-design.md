# Shell tab with a real terminal

Status: implemented. Verification: **not verified** in a real container (this
sandbox cannot start containers). Backend covered by a fake `podman` script that
execs `/bin/sh` on a PTY.

## Intent

Run commands inside a container without leaving lazypodman, including
full-screen tools (`vim`, `htop`, `less`). The previous line-oriented design was
rolled back because it could not do that.

## How it works

- `exec shell` is the existing `E` binding, also listed in the `x` options menu.
  The tab does not exist before it is used, so nothing is spawned at startup.
- Choosing it starts `podman exec -it -- <id> /bin/sh -c '<login shell>'` on a
  PTY (`creack/pty`), sized to the main panel, and switches the main panel to the
  new **shell** tab, inserted right after **Logs**.
- Output is parsed by `hinshun/vt10x` and rendered cell-by-cell as plain text,
  with the terminal cursor. Keys are forwarded raw: printable ASCII, arrows,
  home/end, page up/down, delete, tab, enter, backspace, escape and ctrl+a..z.
- `ctrl+x` closes the shell, hides the tab and returns to the container list.
  The tab also disappears when the shell exits by itself.
- The PTY is resized whenever the layout gives the terminal view a new size.
- One shell at a time; starting another container's shell closes the old one.
- Without a usable CLI or a running container, an error panel explains why and
  no tab is created.

## Trade-offs

- Two dependencies (`creack/pty`, `hinshun/vt10x`), both small and focused.
- Colours are parsed but not rendered because gocui's view API only draws plain
  text. Mouse reporting is not forwarded.
- `ctrl+x` is reserved for closing the shell, so it cannot be used inside the
  container.
- `E` in the Containers panel now opens the in-app terminal instead of
  suspending the TUI; the native attach stays on `a`.
