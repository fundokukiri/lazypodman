package commands

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PodsMode is how hard lazypodman should try to enable the Podman-native
// features, configured as gui.pods.enabled (auto|always|never).
type PodsMode int

const (
	// PodsAuto runs every check: binary, version and a proven shared store.
	PodsAuto PodsMode = iota
	// PodsAlways skips the store comparison, keeping only the binary and
	// version checks. Escape hatch for cold rootless starts where `podman info`
	// is slow or reports an unexpected path.
	PodsAlways
	// PodsNever keeps the meaningful Podman features off.
	PodsNever
)

// minPodmanVersion is the oldest Podman whose CLI JSON output lazypodman
// supports.
const minPodmanVersion = "3.0.0"

// cliCheckBudget bounds the whole enablement step (binary, version, store
// comparison and capability probe), not each individual command. A var so tests
// can shrink it.
var cliCheckBudget = 5 * time.Second

// CLICheck is the outcome of the startup enablement check. Reason is a short,
// UI-safe code (rendered as `pods off: <reason>`); Detail carries the raw output
// for the debug log.
type CLICheck struct {
	Enabled  bool
	Reason   string
	Detail   string
	Version  string
	Features map[string]bool
}

// detectCLI decides whether the podman CLI can back the Podman-native features.
//
// It never returns an error: every failure becomes a short Reason, because a
// machine without podman (or with a podman talking to another store) is an
// expected state, not a crash. `run` is injected so the checks are testable.
func detectCLI(
	ctx context.Context,
	socketPath string,
	mode PodsMode,
	run func(context.Context, ...string) ([]byte, error),
) CLICheck {
	check := CLICheck{Features: map[string]bool{}}

	if mode == PodsNever {
		check.Reason = "disabled in config"

		return check
	}

	ctx, cancel := context.WithTimeout(ctx, cliCheckBudget)
	defer cancel()

	versionOut, err := run(ctx, "--version")
	if err != nil {
		check.Reason = cliFailureReason(ctx, err)
		check.Detail = err.Error()

		return check
	}

	check.Detail = strings.TrimSpace(string(versionOut))

	version, err := parsePodmanVersion(string(versionOut))
	if err != nil {
		check.Reason = "podman version unknown"

		return check
	}

	check.Version = version

	if !versionAtLeast(version) {
		check.Reason = "podman too old"

		return check
	}

	if mode != PodsAlways {
		if reason := checkSharedStore(ctx, socketPath, run); reason != "" {
			check.Reason = reason

			return check
		}
	}

	// Capability probe, not fatal: older Podman versions simply lose the
	// single-process merged log stream.
	if helpOut, err := run(ctx, "pod", "logs", "--help"); err == nil && strings.Contains(string(helpOut), "--names") {
		check.Features["pod-logs-names"] = true
	}

	check.Enabled = true

	return check
}

// checkSharedStore proves that the CLI and the socket lazypodman talks to
// operate on the same store. It returns a reason code when they do not, and an
// empty string when the store is shared.
func checkSharedStore(ctx context.Context, socketPath string, run func(context.Context, ...string) ([]byte, error)) string {
	normalizedSocket, comparable := normalizeSocketPath(socketPath)
	if !comparable {
		return "socket not comparable"
	}

	infoOut, err := run(ctx, "info", "--format", "{{.Host.RemoteSocket.Path}}")
	if err != nil {
		return cliFailureReason(ctx, err)
	}

	normalizedInfo, comparable := normalizeSocketPath(string(infoOut))
	if !comparable {
		return "remote podman CLI"
	}

	if normalizedInfo != normalizedSocket {
		return "CLI/socket store mismatch"
	}

	return ""
}

func cliFailureReason(ctx context.Context, err error) string {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "checks timed out"
	}

	if errors.Is(err, exec.ErrNotFound) {
		return "podman CLI not found"
	}

	return "podman CLI failed"
}

var podmanVersionRegexp = regexp.MustCompile(`podman version (\d+)\.(\d+)\.(\d+)`)

// parsePodmanVersion extracts a numeric X.Y.Z from `podman --version`, dropping
// vendor suffixes such as `-rhel` in `4.9.3-rhel`.
func parsePodmanVersion(out string) (string, error) {
	match := podmanVersionRegexp.FindStringSubmatch(out)
	if match == nil {
		return "", fmt.Errorf("cannot parse podman version from %q", strings.TrimSpace(out))
	}

	return match[1] + "." + match[2] + "." + match[3], nil
}

// versionAtLeast reports whether the numeric X.Y.Z version is at least the
// minimum Podman version lazypodman supports.
func versionAtLeast(version string) bool {
	return compareVersions(version, minPodmanVersion) >= 0
}

// compareVersions compares two numeric X.Y.Z versions. Unparsable versions
// compare as -1 against anything.
func compareVersions(a, b string) int {
	left, ok := parseVersionParts(a)
	if !ok {
		return -1
	}

	right, ok := parseVersionParts(b)
	if !ok {
		return -1
	}

	for i := range left {
		if left[i] != right[i] {
			return left[i] - right[i]
		}
	}

	return 0
}

func parseVersionParts(version string) ([3]int, bool) {
	var parts [3]int

	split := strings.Split(version, ".")
	if len(split) != 3 {
		return parts, false
	}

	for i, part := range split {
		value, err := strconv.Atoi(part)
		if err != nil {
			return parts, false
		}

		parts[i] = value
	}

	return parts, true
}
