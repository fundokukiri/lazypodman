package commands

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeResult struct {
	out string
	err error
}

// fakeCLI answers detection commands from a map keyed by the joined argv.
type fakeCLI struct {
	results map[string]fakeResult
	hang    bool
}

func (f fakeCLI) run(ctx context.Context, argv ...string) ([]byte, error) {
	if f.hang {
		<-ctx.Done()

		return nil, ctx.Err()
	}

	result, ok := f.results[strings.Join(argv, " ")]
	if !ok {
		return nil, errors.New("unexpected command: " + strings.Join(argv, " "))
	}

	return []byte(result.out), result.err
}

func healthyFakeCLI(socketPath string) fakeCLI {
	return fakeCLI{results: map[string]fakeResult{
		"--version": {"podman version 5.2.2", nil},
		"info --format {{.Host.RemoteSocket.Path}}": {socketPath + "\n", nil},
		"pod logs --help": {"--follow\n--names\n--tail int\n", nil},
	}}
}

func TestDetectCLIEnabled(t *testing.T) {
	socket := "/run/user/1000/podman/podman.sock"
	check := detectCLI(context.Background(), "unix://"+socket, PodsAuto, healthyFakeCLI(socket).run)

	assert.True(t, check.Enabled)
	assert.Empty(t, check.Reason)
	assert.Equal(t, "5.2.2", check.Version)
	assert.True(t, check.Features["pod-logs-names"])
	assert.Contains(t, check.Detail, "5.2.2")
}

func TestDetectCLIDisabledReasons(t *testing.T) {
	socket := "/run/user/1000/podman/podman.sock"

	cases := []struct {
		name string
		cli  fakeCLI
		mode PodsMode
		want string
	}{
		{
			name: "binary missing",
			cli:  fakeCLI{results: map[string]fakeResult{"--version": {"", exec.ErrNotFound}}},
			mode: PodsAuto,
			want: "podman CLI not found",
		},
		{
			name: "too old",
			cli: fakeCLI{results: map[string]fakeResult{
				"--version": {"podman version 2.2.1", nil},
			}},
			mode: PodsAuto,
			want: "podman too old",
		},
		{
			name: "unparsable version",
			cli: fakeCLI{results: map[string]fakeResult{
				"--version": {"something else entirely", nil},
			}},
			mode: PodsAuto,
			want: "podman version unknown",
		},
		{
			name: "remote cli",
			cli: fakeCLI{results: map[string]fakeResult{
				"--version": {"podman version 5.2.2", nil},
				"info --format {{.Host.RemoteSocket.Path}}": {"\n", nil},
			}},
			mode: PodsAuto,
			want: "remote podman CLI",
		},
		{
			name: "incomparable socket",
			cli: fakeCLI{results: map[string]fakeResult{
				"--version": {"podman version 5.2.2", nil},
				"info --format {{.Host.RemoteSocket.Path}}": {socket + "\n", nil},
			}},
			mode: PodsAuto,
			want: "socket not comparable",
		},
		{
			name: "store mismatch",
			cli: fakeCLI{results: map[string]fakeResult{
				"--version": {"podman version 5.2.2", nil},
				"info --format {{.Host.RemoteSocket.Path}}": {"/run/podman/podman.sock\n", nil},
				"pod logs --help": {"--names\n", nil},
			}},
			mode: PodsAuto,
			want: "CLI/socket store mismatch",
		},
		{
			name: "never",
			cli:  healthyFakeCLI(socket),
			mode: PodsNever,
			want: "disabled in config",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// fd:// is the incomparable case: the caller detected an inherited
			// descriptor instead of a socket path.
			detectedSocket := socket
			if c.name == "incomparable socket" {
				detectedSocket = "fd://3"
			}

			check := detectCLI(context.Background(), detectedSocket, c.mode, c.cli.run)

			assert.False(t, check.Enabled)
			assert.Equal(t, c.want, check.Reason)
		})
	}
}

func TestDetectCLIModes(t *testing.T) {
	socket := "/run/user/1000/podman/podman.sock"
	mismatching := healthyFakeCLI(socket)
	mismatching.results["info --format {{.Host.RemoteSocket.Path}}"] = fakeResult{"/run/podman/podman.sock\n", nil}

	assert.True(t, detectCLI(context.Background(), socket, PodsAlways, mismatching.run).Enabled)
	assert.False(t, detectCLI(context.Background(), socket, PodsAuto, mismatching.run).Enabled)

	missing := fakeCLI{results: map[string]fakeResult{"--version": {"", exec.ErrNotFound}}}
	assert.False(t, detectCLI(context.Background(), socket, PodsAlways, missing.run).Enabled)
}

func TestDetectCLIBudget(t *testing.T) {
	oldBudget := cliCheckBudget
	cliCheckBudget = 50 * time.Millisecond
	t.Cleanup(func() { cliCheckBudget = oldBudget })

	start := time.Now()
	check := detectCLI(context.Background(), "/run/podman/podman.sock", PodsAuto, fakeCLI{hang: true}.run)
	elapsed := time.Since(start)

	assert.False(t, check.Enabled)
	assert.Equal(t, "checks timed out", check.Reason)
	assert.Less(t, elapsed, 2*time.Second)
}

func TestDefaultCLICheckBudget(t *testing.T) {
	assert.Equal(t, 5*time.Second, cliCheckBudget)
}

func TestParsePodmanVersion(t *testing.T) {
	cases := map[string]string{
		"podman version 5.2.2":           "5.2.2",
		"podman version 4.9.3-rhel":      "4.9.3",
		"podman version 3.0.0-dev\n":     "3.0.0",
		"podman version 10.11.12\nfoo\n": "10.11.12",
	}

	for in, want := range cases {
		got, err := parsePodmanVersion(in)
		assert.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}

	for _, in := range []string{"", "podman version", "garbage"} {
		_, err := parsePodmanVersion(in)
		assert.Error(t, err, in)
	}
}

func TestVersionAtLeast(t *testing.T) {
	assert.True(t, versionAtLeast("3.0.0"))
	assert.True(t, versionAtLeast("4.1.0"))
	assert.False(t, versionAtLeast("2.2.1"))
	assert.False(t, versionAtLeast("garbage"))
}

// writeFakePodmanScript creates an executable `podman` that records each argv
// element on its own line and replays STDOUT_FILE/STDERR_FILE. It is skipped on
// Windows because it is a POSIX shell script.
func writeFakePodmanScript(t *testing.T) (binary string, argvFile string, stdoutFile string, stderrFile string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("fake podman script requires a POSIX shell")
	}

	dir := t.TempDir()
	binary = filepath.Join(dir, "podman")
	argvFile = filepath.Join(dir, "argv.log")
	stdoutFile = filepath.Join(dir, "stdout.log")
	stderrFile = filepath.Join(dir, "stderr.log")

	script := `#!/bin/sh
for arg in "$@"; do printf '[%s]\n' "$arg"; done >> "$ARGV_FILE"
printf -- '---\n' >> "$ARGV_FILE"
cat "$STDOUT_FILE" 2>/dev/null
cat "$STDERR_FILE" >&2 2>/dev/null
exit "${FAKE_EXIT:-0}"
`

	requireNoError(t, os.WriteFile(binary, []byte(script), 0o755))

	t.Setenv("ARGV_FILE", argvFile)
	t.Setenv("STDOUT_FILE", stdoutFile)
	t.Setenv("STDERR_FILE", stderrFile)

	return binary, argvFile, stdoutFile, stderrFile
}

func readRecordedArgv(t *testing.T, argvFile string) string {
	t.Helper()

	content, err := os.ReadFile(argvFile)
	requireNoError(t, err)

	return string(content)
}

func TestPodArgv(t *testing.T) {
	argv, err := podArgv("start", "web api")
	assert.NoError(t, err)
	assert.Equal(t, []string{"pod", "start", "--", "web api"}, argv)

	argv, err = podArgv("rm", "web", "--force")
	assert.NoError(t, err)
	assert.Equal(t, []string{"pod", "rm", "--force", "--", "web"}, argv)

	for _, ref := range []string{"", "-all", "--force"} {
		_, err := podArgv("start", ref)
		assert.Error(t, err, ref)
		assert.Contains(t, err.Error(), "invalid pod reference")
	}
}

func TestCLIActionsRecordArgv(t *testing.T) {
	binary, argvFile, _, _ := writeFakePodmanScript(t)
	client := &cliPodmanClient{binary: binary}

	assert.NoError(t, client.PodAction(context.Background(), "start", "web api"))
	assert.NoError(t, client.PodAction(context.Background(), "rm", "db", "--force"))

	want := "[pod]\n[start]\n[--]\n[web api]\n---\n[pod]\n[rm]\n[--force]\n[--]\n[db]\n---\n"
	assert.Equal(t, want, readRecordedArgv(t, argvFile))
}

func TestCLIActionSurfacesStderr(t *testing.T) {
	binary, _, _, stderrFile := writeFakePodmanScript(t)
	requireNoError(t, os.WriteFile(stderrFile, []byte("Error: no such pod\n"), 0o600))
	t.Setenv("FAKE_EXIT", "2")

	client := &cliPodmanClient{binary: binary}
	err := client.PodAction(context.Background(), "stop", "web")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such pod")
	assert.Contains(t, err.Error(), "pod stop")
}

func TestCLIPruneAndInspectRecordArgv(t *testing.T) {
	binary, argvFile, stdoutFile, _ := writeFakePodmanScript(t)
	requireNoError(t, os.WriteFile(stdoutFile, []byte("[]\n"), 0o600))
	client := &cliPodmanClient{binary: binary}

	assert.NoError(t, client.PodPrune(context.Background()))
	_, err := client.PodInspect(context.Background(), "web")
	assert.NoError(t, err)
	_, err = client.PodInspect(context.Background(), "-bad")
	assert.Error(t, err)

	want := "[pod]\n[prune]\n[--force]\n---\n[pod]\n[inspect]\n[--]\n[web]\n---\n"
	assert.Equal(t, want, readRecordedArgv(t, argvFile))
}

func TestCLIPodListBackfillsClient(t *testing.T) {
	binary, argvFile, stdoutFile, _ := writeFakePodmanScript(t)
	requireNoError(t, os.WriteFile(stdoutFile, readPodFixture(t, "v5.2.json"), 0o600))
	client := &cliPodmanClient{binary: binary}

	pods, err := client.PodList(context.Background())
	assert.NoError(t, err)
	assert.Len(t, pods, 3)
	assert.Same(t, client, pods[0].PodClient)
	assert.Equal(t, binary, pods[0].PodmanBinary)
	assert.Equal(t, "[pod]\n[ls]\n[--format]\n[json]\n---\n", readRecordedArgv(t, argvFile))
}

func TestCLIContainerPodMapUsesPsAll(t *testing.T) {
	binary, argvFile, stdoutFile, _ := writeFakePodmanScript(t)
	requireNoError(t, os.WriteFile(stdoutFile, readPodFixture(t, "ps_v5.json"), 0o600))
	client := &cliPodmanClient{binary: binary}

	podByContainer, err := client.ContainerPodMap(context.Background())
	assert.NoError(t, err)
	assert.Len(t, podByContainer, 3)
	assert.Equal(t, "[ps]\n[-a]\n[--format]\n[json]\n---\n", readRecordedArgv(t, argvFile))
}

func TestParsePodLogLine(t *testing.T) {
	names := []string{"web", "web-side", "side"}

	cases := []struct {
		line    string
		ctr     string
		message string
	}{
		{"web hello", "web", "hello"},
		{"web-side hello there", "web-side", "hello there"},
		{"side ", "side", ""},
		{"webbish: hi", "", "webbish: hi"},
		{"web", "", "web"},
		{"", "", ""},
	}

	for _, tc := range cases {
		ctr, message := parsePodLogLine(tc.line, names)
		assert.Equal(t, tc.ctr, ctr, tc.line)
		assert.Equal(t, tc.message, message, tc.line)
	}
}

func containerNamed(name, state string) *Container {
	ctr := &Container{Name: name}
	ctr.Container.State = state

	return ctr
}

func TestPodLogArgvs(t *testing.T) {
	pod := &Pod{Name: "web"}

	argvs, omitted := podLogArgvs(pod, nil, 100, true)
	assert.Equal(t, [][]string{{"pod", "logs", "--follow", "--names", "--tail", "100", "--", "web"}}, argvs)
	assert.Zero(t, omitted)

	// The fallback prefers running containers and caps the process count.
	containers := make([]*Container, 0, 10)
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		containers = append(containers, containerNamed(name, "exited"))
	}

	containers[8] = containerNamed("late", "running")

	argvs, omitted = podLogArgvs(pod, containers, 50, false)
	assert.Len(t, argvs, maxPodLogProcesses)
	assert.Equal(t, 2, omitted)
	assert.Equal(t, []string{"logs", "--follow", "--names", "--tail", "50", "--", "late"}, argvs[0])
	assert.Equal(t, []string{"logs", "--follow", "--names", "--tail", "50", "--", "a"}, argvs[1])

	// Containers whose name looks like a flag are skipped, never passed on.
	argvs, _ = podLogArgvs(pod, []*Container{containerNamed("-rf", "running"), containerNamed("ok", "running")}, 10, false)
	assert.Len(t, argvs, 1)
	assert.Equal(t, "ok", argvs[0][len(argvs[0])-1])

	argvs, omitted = podLogArgvs(pod, nil, 10, false)
	assert.Empty(t, argvs)
	assert.Zero(t, omitted)
}

func TestPodLogsStreamsAndCloses(t *testing.T) {
	binary, argvFile, stdoutFile, _ := writeFakePodmanScript(t)
	requireNoError(t, os.WriteFile(stdoutFile, []byte("web first\nweb second\n"), 0o600))

	client := &cliPodmanClient{binary: binary, check: CLICheck{Enabled: true, Features: map[string]bool{"pod-logs-names": true}}}

	stream, err := client.PodLogs(context.Background(), &Pod{Name: "web"}, nil, 100)
	requireNoError(t, err)

	content, err := io.ReadAll(stream)
	requireNoError(t, err)
	requireNoError(t, stream.Close())

	assert.Equal(t, "web first\nweb second\n", string(content))
	assert.Equal(t, "[pod]\n[logs]\n[--follow]\n[--names]\n[--tail]\n[100]\n[--]\n[web]\n---\n", readRecordedArgv(t, argvFile))
}

func TestPodLogsCloseKillsChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake podman script requires a POSIX shell")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "podman")
	requireNoError(t, os.WriteFile(binary, []byte("#!/bin/sh\nprintf 'web first\\n'\nsleep 30\n"), 0o755))

	client := &cliPodmanClient{binary: binary, check: CLICheck{Enabled: true, Features: map[string]bool{"pod-logs-names": true}}}

	stream, err := client.PodLogs(context.Background(), &Pod{Name: "web"}, nil, 10)
	requireNoError(t, err)

	buffer := make([]byte, 64)
	n, err := stream.Read(buffer)
	requireNoError(t, err)
	assert.Equal(t, "web first\n", string(buffer[:n]))

	start := time.Now()
	requireNoError(t, stream.Close())
	assert.Less(t, time.Since(start), 5*time.Second, "Close must not wait for podman to finish on its own")
}
