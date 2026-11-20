//go:build !windows

package commands

import (
	"os/exec"
	"syscall"
)

// detachLauncher gives the terminal window its own session. Shared with
// lazypodman's process group, it would be signalled whenever lazypodman is
// (ctrl+c, closing the terminal hosting the UI) and the attached container
// would die with the UI.
func detachLauncher(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
