package commands

// AttachTerminalArgv is the command run for `a` in the external terminal and
// for `A` in the current terminal. It runs plain podman attach (no log replay)
// and nudges the terminal size once, because some containers leave the screen
// blank until they receive SIGWINCH after attaching.
func AttachTerminalArgv(podmanBinary, ref string) []string {
	return attachTerminalArgv(podmanBinary, ref)
}

// AttachFullLogsTerminalArgv is the explicit `ctrl+a` variant: replay the full
// container history, then attach. The nudge fixes containers that otherwise
// wait for SIGWINCH before drawing.
func AttachFullLogsTerminalArgv(podmanBinary, ref string) []string {
	return attachFullLogsTerminalArgv(podmanBinary, ref)
}
