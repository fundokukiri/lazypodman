//go:build windows

package commands

func attachTerminalArgv(podmanBinary, ref string) []string {
	return []string{podmanBinary, "attach", "--sig-proxy=false", ref}
}
