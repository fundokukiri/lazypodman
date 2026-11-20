package commands

import (
	"context"
	"io"
	"path/filepath"
	"runtime"
	"strings"
)

// PodmanClient is the podman-native backend. The Docker-compatible service API
// has no pod endpoints, so pods (and anything else Podman-only) go through the
// podman CLI. Implementations must be safe for concurrent use.
type PodmanClient interface {
	Enabled() bool
	Reason() string
	Version() string

	PodList(ctx context.Context) ([]*Pod, error)
	ContainerPodMap(ctx context.Context) (map[string]string, error)
	PodInspect(ctx context.Context, ref string) ([]byte, error)
	PodAction(ctx context.Context, verb, ref string, extra ...string) error
	PodPrune(ctx context.Context) error
	PodLogs(ctx context.Context, pod *Pod, containers []*Container, tail int) (io.ReadCloser, error)

	// HealthcheckRun runs one probe of a container healthcheck. The returned exit
	// code is a result, not an error.
	HealthcheckRun(ctx context.Context, ref string) (string, int, error)

	// ProbeHealthcheck reports whether the CLI has a healthcheck subcommand.
	ProbeHealthcheck(ctx context.Context) Capability

	// ProbeSubcommand returns the first argv variant this Podman accepts, plus
	// the capability that goes with it.
	ProbeSubcommand(ctx context.Context, variants [][]string) ([]string, Capability)

	// KubeGenerate runs the kube generate command for one pod.
	KubeGenerate(ctx context.Context, prefix []string, ref string) (string, error)

	// StreamKube runs play/down and streams merged output line by line.
	StreamKube(ctx context.Context, prefix, args []string, onLine func(string)) error

	// StreamPodman runs any podman command and streams merged output line by line.
	StreamPodman(ctx context.Context, argv []string, onLine func(string)) error

	// OpenTerminal starts a real PTY shell (`podman exec -it ...`) for the Shell tab.
	OpenTerminal(ctx context.Context, ref string, cols, rows int) (*Terminal, error)

	// OpenAttachTerminal attaches to the container's main process in a PTY for
	// the Shell tab, instead of opening an external terminal.
	OpenAttachTerminal(ctx context.Context, ref string, cols, rows int) (*Terminal, error)
}

// PodStatsRow is the pod-level CPU/memory figure shown in the pod stats tab.
// The numbers are a sum over the pod's containers (infra container included),
// not a value reported by `podman pod stats`.
type PodStatsRow struct {
	PodName  string
	CPU      float64
	MemUsage int64
	MemLimit int64
}

// PodState is one snapshot of the podman-native data: the pods themselves plus
// the container-ID -> pod-name mapping used to annotate the containers panel.
type PodState struct {
	Pods           []*Pod
	PodByContainer map[string]string
}

// nonComparableSocketPrefixes are transports that do not denote a local path, so
// comparing them with the path `podman info` reports is meaningless: fd:// is an
// inherited descriptor, the rest are remote transports.
var nonComparableSocketPrefixes = []string{"fd://", "ssh://", "tcp://", "http://", "https://"}

// normalizeSocketPath turns a socket endpoint as reported by the environment,
// the Docker client config or `podman info` into a form that can be compared.
//
// The second return value is false when the endpoint is not a comparable local
// path (empty input, fd://, or a remote transport). Callers must treat that as
// "cannot prove the CLI and the socket share a store" rather than as a mismatch
// with a guessed value.
func normalizeSocketPath(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", false
	}

	for _, prefix := range nonComparableSocketPrefixes {
		if strings.HasPrefix(s, prefix) {
			return "", false
		}
	}

	if strings.HasPrefix(s, "npipe://") {
		s = strings.TrimPrefix(s, "npipe://")
		if runtime.GOOS == "windows" {
			// npipe paths are case-insensitive and use backslashes; do not run
			// filepath.Clean over them because it would mangle the \\?\ prefix.
			s = strings.ToLower(strings.ReplaceAll(s, "/", `\`))
		}

		return s, true
	}

	s = strings.TrimPrefix(s, "unix://")
	// Collapse redundant leading slashes so the result is the same on every OS;
	// filepath.Clean treats `//x` as a UNC root on Windows.
	for strings.HasPrefix(s, "//") {
		s = s[1:]
	}

	// Resolve symlinks when possible (e.g. /var/run -> /run), otherwise fall back
	// to the cleaned path: a socket that does not exist yet is still comparable.
	if resolved, err := filepath.EvalSymlinks(s); err == nil {
		s = resolved
	}

	s = filepath.Clean(s)
	if runtime.GOOS == "windows" {
		s = strings.ToLower(s)
	}

	return s, true
}
