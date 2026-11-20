package commands

import "errors"

// ErrNoTerminalEmulator is returned when no supported terminal emulator can be
// found. The caller can turn it into a translated error message.
var ErrNoTerminalEmulator = errors.New("no terminal emulator found")

// OpenExternalTerminal starts argv in a new terminal window and returns as soon
// as the launcher has started. Neither stdin nor the output of the new terminal
// is connected to lazypodman, so the TUI keeps running.
func (c *OSCommand) OpenExternalTerminal(argv []string) error {
	launcher, err := externalTerminalArgv(argv)
	if err != nil {
		return err
	}

	c.Log.Infof("opening external terminal: %v", launcher)

	cmd := c.NewCmd(launcher[0], launcher[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	detachLauncher(cmd)

	if err := cmd.Start(); err != nil {
		return err
	}

	// Reap the launcher when it exits without blocking the UI.
	go func() {
		_ = cmd.Wait()
	}()

	return nil
}
