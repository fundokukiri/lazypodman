package commands

// AttachFullLogsTerminalArgv is the explicit `ctrl+a` variant: replay the full
// container history, then attach. The nudge fixes containers that otherwise
// wait for SIGWINCH before drawing.
func AttachFullLogsTerminalArgv(podmanBinary, ref string) []string {
	return attachFullLogsTerminalArgv(podmanBinary, ref)
}

// AttachTerminalArgv is the command run for `a` in a separate terminal, for `A`
// in the current terminal and for F2 in the shell tab. It runs plain podman
// attach, so the screen shows what the container itself is running; ctrl+x is
// the one that opens a new shell in the container.
func AttachTerminalArgv(podmanBinary, ref string) []string {
	return attachTerminalArgv(podmanBinary, ref)
}
