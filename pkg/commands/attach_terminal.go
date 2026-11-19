package commands

// AttachTerminalArgv is the command run for `a` in the external terminal and
// for `A` in the current terminal. It is the plain podman attach command: the
// live output starts immediately at the bottom instead of replaying history.
func AttachTerminalArgv(podmanBinary, ref string) []string {
	return []string{podmanBinary, "attach", "--sig-proxy=false", ref}
}
