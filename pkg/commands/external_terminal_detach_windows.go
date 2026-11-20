//go:build windows

package commands

import "os/exec"

// detachLauncher is a no-op: `start` already opens the window outside
// lazypodman's console process group.
func detachLauncher(cmd *exec.Cmd) {}
