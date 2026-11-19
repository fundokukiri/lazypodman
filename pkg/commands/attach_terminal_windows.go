//go:build windows

package commands

func attachTerminalArgv(podmanBinary, ref string) []string {
	return []string{podmanBinary, "attach", "--sig-proxy=false", ref}
}

func attachFullLogsTerminalArgv(podmanBinary, ref string) []string {
	// Windows has no POSIX shell to run the two commands with safe quoting.
	// Keep the explicit full-log key on the normal attach path there.
	return attachTerminalArgv(podmanBinary, ref)
}
