package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/assert"
)

// writeInteractiveFakePodman replaces podman with a script that drops the user
// into a real shell, which is what `podman exec -it` does for the tests.
func writeInteractiveFakePodman(t *testing.T) (string, string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("fake podman script requires a POSIX shell")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "podman")
	argvFile := filepath.Join(dir, "argv.log")

	script := `#!/bin/sh
for arg in "$@"; do printf '[%s]\n' "$arg"; done >> "$ARGV_FILE"
printf -- '---\n' >> "$ARGV_FILE"
exec /bin/sh
`

	requireNoError(t, os.WriteFile(binary, []byte(script), 0o755))
	t.Setenv("ARGV_FILE", argvFile)

	return binary, argvFile
}

func waitForScreen(t *testing.T, terminal *Terminal, want string) []string {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		rows, _, _, _ := terminal.Snapshot()
		if strings.Contains(strings.Join(rows, "\n"), want) {
			return rows
		}

		time.Sleep(20 * time.Millisecond)
	}

	rows, _, _, _ := terminal.Snapshot()
	t.Fatalf("timed out waiting for %q in screen:\n%s", want, strings.Join(rows, "\n"))

	return nil
}

func TestOpenTerminalRunsAShellAndForwardsInput(t *testing.T) {
	binary, argvFile := writeInteractiveFakePodman(t)

	client := &cliPodmanClient{binary: binary}
	terminal, err := client.OpenTerminal(context.Background(), "abc123", 60, 10)
	requireNoError(t, err)
	defer func() { _ = terminal.Close() }()

	requireNoError(t, terminal.Write([]byte("echo terminal-ok\r")))
	rows := waitForScreen(t, terminal, "terminal-ok")

	assert.LessOrEqual(t, len(rows), 10, "screen stays inside the PTY size")

	// The shell must see the PTY as a terminal of the size we asked for.
	requireNoError(t, terminal.Write([]byte("stty size\r")))
	waitForScreen(t, terminal, "10 60")
	assert.Contains(t, readRecordedArgv(t, argvFile), "[exec]\n[-it]\n[--]\n[abc123]\n")
}

func TestOpenTerminalResizeKeepsWorking(t *testing.T) {
	binary, _ := writeInteractiveFakePodman(t)

	client := &cliPodmanClient{binary: binary}
	terminal, err := client.OpenTerminal(context.Background(), "abc123", 40, 8)
	requireNoError(t, err)
	defer func() { _ = terminal.Close() }()

	requireNoError(t, terminal.Resize(80, 20))

	rows, _, _, _ := terminal.Snapshot()
	assert.Len(t, rows, 20)

	requireNoError(t, terminal.Write([]byte("echo after-resize\r")))
	waitForScreen(t, terminal, "after-resize")
}

func TestOpenTerminalRejectsInvalidSizeAndReference(t *testing.T) {
	client := &cliPodmanClient{binary: "/usr/bin/podman"}

	_, err := client.OpenTerminal(context.Background(), "abc", 0, 10)
	assert.Error(t, err)

	_, err = client.OpenTerminal(context.Background(), "--help", 80, 24)
	assert.Error(t, err)
}

func TestTerminalCloseIsIdempotent(t *testing.T) {
	binary, _ := writeInteractiveFakePodman(t)

	client := &cliPodmanClient{binary: binary}
	terminal, err := client.OpenTerminal(context.Background(), "abc123", 40, 8)
	requireNoError(t, err)

	requireNoError(t, terminal.Close())
	assert.NoError(t, terminal.Close())
}

func TestTerminalWriteAndSnapshotWithoutTerminal(t *testing.T) {
	var terminal *Terminal
	assert.Error(t, terminal.Write([]byte("x")))
	assert.NoError(t, terminal.Close())

	rows, _, _, visible := terminal.Snapshot()
	assert.Nil(t, rows)
	assert.False(t, visible)
}

func TestScrollbackKeepsBurstOutput(t *testing.T) {
	terminal := &Terminal{historyScreen: vt10x.New(vt10x.WithSize(20, 200))}

	for i := 0; i < 50; i++ {
		_, _ = fmt.Fprintf(terminal.historyScreen, "line %d\r\n", i)
	}

	rows, _, cursorY, ok := terminal.Scrollback()
	assert.True(t, ok)
	// The blank row where the caret waits is kept so the prompt has a place.
	assert.Len(t, rows, 51)
	assert.Equal(t, "line 0", rows[0])
	assert.Equal(t, "line 49", rows[49])
	assert.Equal(t, 50, cursorY)
}

func TestScrollbackIgnoresAltScreen(t *testing.T) {
	terminal := &Terminal{historyScreen: vt10x.New(vt10x.WithSize(20, 200))}
	_, _ = terminal.historyScreen.Write([]byte("\x1b[?1049h"))

	_, _, _, ok := terminal.Scrollback()
	assert.False(t, ok)
}
