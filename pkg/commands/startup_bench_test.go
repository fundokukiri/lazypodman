package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// fakeTimedPodman writes a podman stand-in that answers the startup checks after
// a fixed delay. Benchmarking against it makes the cost of one rootless podman
// start visible without needing podman (or a socket) on the machine.
func fakeTimedPodman(tb testing.TB) (binary string, logFile string) {
	tb.Helper()

	if runtime.GOOS == "windows" {
		tb.Skip("fake podman script requires a POSIX shell")
	}

	delayMs, _ := strconv.Atoi(os.Getenv("BENCH_PODMAN_DELAY_MS"))

	dir := tb.TempDir()
	binary = filepath.Join(dir, "podman")
	logFile = filepath.Join(dir, "calls.log")

	script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %q
sleep 0.%03d
case "$1 $2" in
  "info --format") printf '5.2.3|/run/user/1000/podman/podman.sock\n' ;;
  "pod logs") printf -- '--names\n' ;;
  "pod ls") printf '[]\n' ;;
  "ps -a") printf '[]\n' ;;
  *) printf 'ok\n' ;;
esac
`, logFile, delayMs)

	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		tb.Fatal(err)
	}

	return binary, logFile
}

func TestStartupCLICheckSpawnsOneProcess(t *testing.T) {
	binary, logFile := fakeTimedPodman(t)

	client := newCLIPodmanClient(context.Background(), binary, "/run/user/1000/podman/podman.sock", PodsAuto)
	if !client.Enabled() {
		t.Fatalf("expected the fake CLI to enable pods, reason: %s", client.Reason())
	}

	calls := readFakeCalls(t, logFile)
	if len(calls) != 1 {
		t.Fatalf("startup must spawn exactly one podman process, got %d: %v", len(calls), calls)
	}
}

func TestRefreshPodStateRunsItsTwoCallsConcurrently(t *testing.T) {
	binary, _ := fakeTimedPodman(t)

	client := &cliPodmanClient{binary: binary}
	client.check = CLICheck{Enabled: true, Version: "5.2.3"}

	command := &PodmanCommand{}
	command.SetPodClient(client)

	const delay = 150 * time.Millisecond
	start := time.Now()
	if _, err := command.RefreshPodState(context.Background()); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)

	// Two sequential 150 ms calls would take at least 300 ms. The bound is loose on
	// purpose: CI machines are slow, but the calls must overlap.
	if elapsed >= 2*delay {
		t.Fatalf("pod list and container map did not overlap: %s", elapsed)
	}
}

func BenchmarkStartupCLICheck(b *testing.B) {
	binary, _ := fakeTimedPodman(b)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		newCLIPodmanClient(context.Background(), binary, "/run/user/1000/podman/podman.sock", PodsAuto)
	}
}

func BenchmarkRefreshPodState(b *testing.B) {
	binary, _ := fakeTimedPodman(b)

	client := &cliPodmanClient{binary: binary}
	client.check = CLICheck{Enabled: true, Version: "5.2.3"}

	command := &PodmanCommand{}
	command.SetPodClient(client)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := command.RefreshPodState(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}

func readFakeCalls(tb testing.TB, logFile string) []string {
	tb.Helper()

	data, err := os.ReadFile(logFile)
	if err != nil {
		return nil
	}

	lines := []string{}
	for _, line := range splitLines(string(data)) {
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

func splitLines(s string) []string {
	out := []string{}
	start := 0

	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}

	return append(out, s[start:])
}
