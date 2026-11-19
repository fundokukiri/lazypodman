package commands

// AttachTerminalArgv is the command run inside the external terminal for `a`.
// podman attach only streams output produced after the attach starts, so a
// process that already printed and exited would appear truncated. Replaying
// the container's logs first keeps the earlier output in the terminal's
// scrollback, then the live attach continues from that point.
func AttachTerminalArgv(podmanBinary, ref string) []string {
	return attachTerminalArgv(podmanBinary, ref)
}
