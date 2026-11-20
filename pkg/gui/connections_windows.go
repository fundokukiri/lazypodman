//go:build windows

package gui

import "errors"

func restartWithPodmanHost(uri string) error {
	return errors.New("automatic connection switching is not supported on Windows; set CONTAINER_HOST and restart lazypodman")
}
