package commands

import (
	"context"
	"errors"
	"os/exec"
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
