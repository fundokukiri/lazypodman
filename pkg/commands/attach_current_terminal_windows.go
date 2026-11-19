//go:build windows

package commands

import (
	"fmt"
	"strings"
)

func currentTerminalAttachArgv(podmanBinary, ref string) []string {
	quote := func(s string) string {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}

	command := fmt.Sprintf(`cls & %s logs %s & %s attach --sig-proxy=false %s`,
		quote(podmanBinary), quote(ref), quote(podmanBinary), quote(ref))

	return []string{"cmd.exe", "/c", command}
}
