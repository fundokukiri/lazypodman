package commands

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// writeEchoShellPodman replaces podman with a script that behaves like
// `podman exec -i`: every line read is echoed back with an "out:" prefix.
func writeEchoShellPodman(t *testing.T) (string, string) {
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
while IFS= read -r line; do printf 'out:%s\n' "$line"; done
`

	requireNoError(t, os.WriteFile(binary, []byte(script), 0o755))
	t.Setenv("ARGV_FILE", argvFile)

	return binary, argvFile
}

func TestOpenShellSendsLinesAndReadsOutput(t *testing.T) {
	binary, argvFile := writeEchoShellPodman(t)

	client := &cliPodmanClient{binary: binary}
	session, err := client.OpenShell(context.Background(), "abc123")
	requireNoError(t, err)
	defer func() { _ = session.Close() }()

	requireNoError(t, session.Send("echo hi"))
	requireNoError(t, session.Send("pwd"))

	scanner := bufio.NewScanner(session.Output())
	var lines []string

	deadline := time.After(5 * time.Second)
	for len(lines) < 2 {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for shell output, got %v", lines)
		default:
		}

		if !scanner.Scan() {
			t.Fatalf("shell output ended early: %v", scanner.Err())
		}

		lines = append(lines, scanner.Text())
	}

	assert.Equal(t, []string{"out:echo hi", "out:pwd"}, lines)
	assert.Contains(t, readRecordedArgv(t, argvFile), "[exec]\n[-i]\n[--]\n[abc123]\n[/bin/sh]\n")
}

func TestOpenShellClosesCleanly(t *testing.T) {
	binary, _ := writeEchoShellPodman(t)

	client := &cliPodmanClient{binary: binary}
	session, err := client.OpenShell(context.Background(), "abc123")
	requireNoError(t, err)

	requireNoError(t, session.Close())
	assert.Equal(t, session.Close(), session.Close(), "Close must be idempotent")
}

func TestOpenShellRejectsUnsafeReference(t *testing.T) {
	client := &cliPodmanClient{binary: "/usr/bin/podman"}
	_, err := client.OpenShell(context.Background(), "--help")
	assert.Error(t, err)
}

func TestShellSessionSendWithoutSession(t *testing.T) {
	var session *ShellSession
	assert.Error(t, session.Send("ls"))
	assert.NoError(t, session.Close())
}

func TestOpenShellReportsSpawnFailure(t *testing.T) {
	client := &cliPodmanClient{binary: filepath.Join(t.TempDir(), "missing-podman")}
	_, err := client.OpenShell(context.Background(), "abc123")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "cannot find"))
}
