//go:build !windows

package commands

// withTerminalTitle emits the OSC title sequence before starting command. It is
// the one mechanism every emulator honours: -T/--title differ per emulator and
// are unsupported by xdg-terminal-exec and wezterm.
func withTerminalTitle(argv []string, title string) []string {
	if title == "" {
		return argv
	}

	return append([]string{
		"sh", "-c", `printf '\033]0;%s\007' "$1"; shift; exec "$@"`, "lazypodman-title", title,
	}, argv...)
}
