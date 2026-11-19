//go:build !windows

package commands

func currentTerminalAttachArgv(podmanBinary, ref string) []string {
	// Clear the screen and scrollback, replay the logs, then replace the shell
	// with the live attach. Positional parameters keep the binary and ref out of
	// the shell source.
	const script = `printf '\033[H\033[2J\033[3J'; "$bin" logs "$ref"; exec "$bin" attach --sig-proxy=false "$ref"`

	return []string{"sh", "-c", script, "lazypodman-attach", podmanBinary, ref}
}
