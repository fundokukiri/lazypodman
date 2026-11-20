//go:build !windows

package gui

import (
	"os"
	"syscall"
)

func restartWithPodmanHost(uri string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	return syscall.Exec(executable, os.Args, withPodmanHostEnv(os.Environ(), uri))
}
