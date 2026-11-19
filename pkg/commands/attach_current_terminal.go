package commands

// CurrentTerminalAttachArgv is the fallback for `a` when no external terminal
// emulator exists, and the command behind `A`. It clears the screen (including
// scrollback where supported), replays the container logs and then attaches in
// the terminal lazypodman is already running in.
func CurrentTerminalAttachArgv(podmanBinary, ref string) []string {
	return currentTerminalAttachArgv(podmanBinary, ref)
}
