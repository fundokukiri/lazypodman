//go:build !windows

package commands

// attachWinchNudge makes the attached program redraw itself. Some terminals
// and TUIs render a blank screen until they receive a window-size change, and a
// key press is otherwise needed. Several attempts cover slow terminal startup
// and a slow podman attach; the first signals can arrive before attach listens.
const attachWinchNudge = `( sleep 0.5; kill -WINCH $$ 2>/dev/null; sleep 1; kill -WINCH $$ 2>/dev/null; sleep 2; kill -WINCH $$ 2>/dev/null; sleep 4; kill -WINCH $$ 2>/dev/null ) >/dev/null 2>&1 &`

func attachTerminalArgv(podmanBinary, ref string) []string {
	script := `bin=$1; ref=$2; ` + attachWinchNudge + ` exec "$bin" attach --sig-proxy=false "$ref"`

	return []string{"sh", "-c", script, "lazypodman-attach", podmanBinary, ref}
}

func attachFullLogsTerminalArgv(podmanBinary, ref string) []string {
	script := `bin=$1; ref=$2; "$bin" logs "$ref"; ` + attachWinchNudge + ` exec "$bin" attach --sig-proxy=false "$ref"`

	return []string{"sh", "-c", script, "lazypodman-attach", podmanBinary, ref}
}
