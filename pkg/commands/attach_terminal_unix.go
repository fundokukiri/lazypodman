//go:build !windows

package commands

func attachTerminalArgv(podmanBinary, ref string) []string {
	const script = `bin=$1; ref=$2; ( sleep 0.2; kill -WINCH $$ 2>/dev/null; sleep 0.3; kill -WINCH $$ 2>/dev/null ) >/dev/null 2>&1 & exec "$bin" attach --sig-proxy=false "$ref"`

	return []string{"sh", "-c", script, "lazypodman-attach", podmanBinary, ref}
}

func attachFullLogsTerminalArgv(podmanBinary, ref string) []string {
	const script = `bin=$1; ref=$2; "$bin" logs "$ref"; ( sleep 0.2; kill -WINCH $$ 2>/dev/null; sleep 0.3; kill -WINCH $$ 2>/dev/null ) >/dev/null 2>&1 & exec "$bin" attach --sig-proxy=false "$ref"`

	return []string{"sh", "-c", script, "lazypodman-attach", podmanBinary, ref}
}
