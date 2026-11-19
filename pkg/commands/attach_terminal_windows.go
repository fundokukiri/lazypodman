//go:build windows

package commands

// Windows has no POSIX shell to sequence logs and attach without bringing in
// cmd quoting rules, so keep the direct attach command there.
func attachTerminalArgv(podmanBinary, ref string) []string {
	return []string{podmanBinary, "attach", "--sig-proxy=false", ref}
}
