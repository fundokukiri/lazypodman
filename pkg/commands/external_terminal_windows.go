//go:build windows

package commands

import "strings"

func externalTerminalArgv(argv []string, title string) ([]string, error) {
	if len(argv) == 0 {
		return nil, ErrNoTerminalEmulator
	}

	quoted := make([]string, len(argv))
	for i, arg := range argv {
		if strings.ContainsAny(arg, " \t") {
			arg = `"` + strings.ReplaceAll(arg, `"`, `\"`) + `"`
		}
		quoted[i] = arg
	}

	// The argument after start is the window title; the rest is the command line
	// of the new console window.
	return append([]string{"cmd.exe", "/c", "start", strings.ReplaceAll(title, `"`, "")}, quoted...), nil
}
