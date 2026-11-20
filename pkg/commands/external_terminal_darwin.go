//go:build darwin

package commands

import (
	"fmt"
	"strings"
)

func externalTerminalArgv(argv []string, title string) ([]string, error) {
	if len(argv) == 0 {
		return nil, ErrNoTerminalEmulator
	}

	argv = withTerminalTitle(argv, title)

	command := make([]string, len(argv))
	for i, arg := range argv {
		command[i] = shellQuote(arg)
	}

	script := fmt.Sprintf(`tell application "Terminal" to do script %s`, appleScriptQuote(strings.Join(command, " ")))

	return []string{"osascript", "-e", script}, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func appleScriptQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}
