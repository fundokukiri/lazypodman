//go:build !windows

package commands

func attachTerminalArgv(podmanBinary, ref string) []string {
	// Positional parameters keep the binary and container reference out of the
	// shell source, so spaces or shell metacharacters cannot change the command.
	const script = `bin=$1; ref=$2; "$bin" logs "$ref"; exec "$bin" attach --sig-proxy=false "$ref"`

	return []string{"sh", "-c", script, "lazypodman-attach", podmanBinary, ref}
}
