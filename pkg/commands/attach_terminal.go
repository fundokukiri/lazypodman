package commands

// AttachTerminalArgv is the command run for `a` in a separate terminal and for
// F2 in the current terminal. It runs plain podman attach, so the screen shows
// what the container itself is running; e is the one that opens a new shell.
func AttachTerminalArgv(podmanBinary, ref string) []string {
	return attachTerminalArgv(podmanBinary, ref)
}
