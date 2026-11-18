//go:build !windows

package commands

import (
	"os/exec"
	"syscall"
)

// configureProcessGroup puts the child in its own process group and kills the
// whole group when the context is cancelled. Streaming podman commands often
// wrap helpers (podman, sh, sleep); killing only the direct child leaves the
// descendants holding the pipe open, which makes Close wait for them.
func configureProcessGroup(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}

		return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
}
