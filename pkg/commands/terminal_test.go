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
	assert.Contains(t, readRecordedArgv(t, argvFile), "command -v zsh")
	assert.Contains(t, readRecordedArgv(t, argvFile), "exec bash")
}

func TestShellTerminalArgvRunsThePreferredShell(t *testing.T) {
	got := ShellTerminalArgv("/usr/bin/podman", "abc123")

	assert.Equal(t, []string{
		"/usr/bin/podman", "exec", "-it", "--", "abc123", "/bin/sh", "-c",
		interactiveShellCommand,
	}, got)
	assert.Contains(t, interactiveShellCommand, "command -v zsh")
	assert.Contains(t, interactiveShellCommand, "exec bash")
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

func TestTerminalSnapshotKeepsTextColours(t *testing.T) {
	terminal := &Terminal{screen: vt10x.New(vt10x.WithSize(20, 2))}
	_, _ = terminal.screen.Write([]byte("\x1b[31mred\x1b[0m plain"))

	rows, _, _, _ := terminal.Snapshot()
	assert.Equal(t, "\x1b[0;31mred\x1b[0m plain", strings.TrimRight(rows[0], " "))
}

func TestTerminalSnapshotKeepsReverseDefaultCursor(t *testing.T) {
	terminal := &Terminal{screen: vt10x.New(vt10x.WithSize(20, 2))}
	_, _ = terminal.screen.Write([]byte("abc\x1b[7m \x1b[27mdef"))

	rows, _, _, _ := terminal.Snapshot()
	assert.Equal(t, "abc\x1b[7m \x1b[0mdef", strings.TrimRight(rows[0], " "))
}

func TestTerminalSnapshotKeepsReverseDefaultCursorAtLineStart(t *testing.T) {
	terminal := &Terminal{screen: vt10x.New(vt10x.WithSize(20, 2))}
	_, _ = terminal.screen.Write([]byte("\x1b[7m \x1b[27mdef"))

	rows, _, _, _ := terminal.Snapshot()
	assert.Equal(t, "\x1b[7m \x1b[0mdef", strings.TrimRight(rows[0], " "))
}

func TestTerminalSnapshotLeavesPlainRowsPlain(t *testing.T) {
	terminal := &Terminal{screen: vt10x.New(vt10x.WithSize(20, 2))}

	rows, _, _, _ := terminal.Snapshot()
	assert.Empty(t, strings.TrimRight(rows[0], " "))
}

func TestTerminalScrollbackKeepsTextColours(t *testing.T) {
	terminal := &Terminal{historyScreen: vt10x.New(vt10x.WithSize(20, 200))}
	_, _ = terminal.historyScreen.Write([]byte("\x1b[32mgreen\x1b[0m\r\n"))

	rows, _, _, ok := terminal.Scrollback()
	assert.True(t, ok)
	assert.Equal(t, "\x1b[0;32mgreen\x1b[0m", rows[0])
}

func TestSGRColorCodes(t *testing.T) {
	tests := []struct {
		name       string
		color      vt10x.Color
		foreground bool
		want       string
	}{
		{name: "ansi red", color: vt10x.Red, foreground: true, want: "31"},
		{name: "bright red", color: vt10x.LightRed, foreground: true, want: "91"},
		{name: "background blue", color: vt10x.Blue, foreground: false, want: "44"},
		{name: "256 colour", color: vt10x.Color(123), foreground: true, want: "38;5;123"},
		{name: "true colour", color: vt10x.Color(0x112233), foreground: true, want: "38;2;17;34;51"},
		{name: "default", color: vt10x.DefaultFG, foreground: true, want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, sgrColor(test.color, test.foreground))
		})
	}
}

func TestOpenAttachTerminalAttachesToTheContainer(t *testing.T) {
	// F2 attaches to the container's own process; only ctrl+x opens a new shell.
	binary, argvFile := writeInteractiveFakePodman(t)

	client := &cliPodmanClient{binary: binary}
	terminal, err := client.OpenAttachTerminal(context.Background(), "abc123", 60, 10)
	requireNoError(t, err)
	defer func() { _ = terminal.Close() }()

	assert.Contains(t, waitForRecordedArgv(t, argvFile, "[attach]"), "[attach]\n[--sig-proxy=false]\n[abc123]\n")
}

// writeLoggingFakePodman answers `logs` with a marker line and otherwise runs a
// shell, so a test can tell whether the replay reached the screen.
func writeLoggingFakePodman(t *testing.T) (string, string) {
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
if [ "$1" = "logs" ]; then printf 'before-attach\n'; exit 0; fi
exec /bin/sh
`

	requireNoError(t, os.WriteFile(binary, []byte(script), 0o755))
	t.Setenv("ARGV_FILE", argvFile)

	return binary, argvFile
}

func TestOpenAttachTerminalReplaysTheConsoleLog(t *testing.T) {
	binary, argvFile := writeLoggingFakePodman(t)

	client := &cliPodmanClient{binary: binary}
	terminal, err := client.OpenAttachTerminal(context.Background(), "abc123", 60, 10)
	requireNoError(t, err)
	defer func() { _ = terminal.Close() }()

	rows, _, _, _ := terminal.Snapshot()
	assert.Contains(t, strings.Join(rows, "\n"), "before-attach", "the output from before the attach must be on screen")
	assert.Contains(t, readRecordedArgv(t, argvFile), "[logs]\n[--tail]\n[500]\n[abc123]\n")
}

func TestOpenTerminalDoesNotReplayLogs(t *testing.T) {
	binary, argvFile := writeLoggingFakePodman(t)

	client := &cliPodmanClient{binary: binary}
	terminal, err := client.OpenTerminal(context.Background(), "abc123", 60, 10)
	requireNoError(t, err)
	defer func() { _ = terminal.Close() }()

	rows, _, _, _ := terminal.Snapshot()
	assert.NotContains(t, strings.Join(rows, "\n"), "before-attach", "a fresh shell has no history to replay")
	assert.NotContains(t, waitForRecordedArgv(t, argvFile, "---"), "[logs]")
}
