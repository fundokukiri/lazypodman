//go:build windows

package commands

import "os/exec"

// configureProcessGroup is a no-op on Windows; exec.CommandContext already
// kills the direct child.
func configureProcessGroup(_ *exec.Cmd) {}
