//go:build !windows && !darwin

package commands

import (
	"os"
	"os/exec"

	"github.com/mgutz/str"
)

type terminalCandidate struct {
	binary string
	flag   []string
}

var terminalCandidates = []terminalCandidate{
	{binary: "xdg-terminal-exec"},
	{binary: "x-terminal-emulator", flag: []string{"-e"}},
	{binary: "gnome-terminal", flag: []string{"--"}},
	{binary: "konsole", flag: []string{"-e"}},
	{binary: "xfce4-terminal", flag: []string{"-e"}},
	{binary: "alacritty", flag: []string{"-e"}},
	{binary: "kitty", flag: []string{"-e"}},
	{binary: "wezterm", flag: []string{"start", "--"}},
	{binary: "foot", flag: []string{"-e"}},
	{binary: "tilix", flag: []string{"-e"}},
	{binary: "urxvt", flag: []string{"-e"}},
	{binary: "st", flag: []string{"-e"}},
	{binary: "xterm", flag: []string{"-e"}},
}

func externalTerminalArgv(argv []string, title string) ([]string, error) {
	if len(argv) == 0 {
		return nil, ErrNoTerminalEmulator
	}

	argv = withTerminalTitle(argv, title)

	if terminal := os.Getenv("TERMINAL"); terminal != "" {
		parts := str.ToArgv(terminal)
		if len(parts) > 0 {
			launcher := append(parts, "-e")
			return append(launcher, argv...), nil
		}
	}

	for _, candidate := range terminalCandidates {
		path, err := exec.LookPath(candidate.binary)
		if err != nil {
			continue
		}

		launcher := append([]string{path}, candidate.flag...)
		return append(launcher, argv...), nil
	}

	return nil, ErrNoTerminalEmulator
}
