package commands

// AttachFullLogsTerminalArgv is the explicit `ctrl+a` variant: replay the full
// container history, then attach. The nudge fixes containers that otherwise
// wait for SIGWINCH before drawing.
func AttachFullLogsTerminalArgv(podmanBinary, ref string) []string {
	return attachFullLogsTerminalArgv(podmanBinary, ref)
}
