package commands

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	Enabled bool
	Reason  string
	Detail  string
	Version string
}

// detectCLI decides whether the podman CLI can back the Podman-native features.
//
// It never returns an error: every failure becomes a short Reason, because a
// machine without podman (or with a podman talking to another store) is an
// expected state, not a crash. `run` is injected so the checks are testable.
// podmanInfoFormat asks `podman info` for the two facts the check needs in one
// process: the client version and the socket the CLI talks to. Rootless podman
// pays a storage/database initialisation cost per process, so the count matters.
const podmanInfoFormat = "{{.Version.Version}}|{{.Host.RemoteSocket.Path}}"

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
	check := CLICheck{}

	if mode == PodsNever {
		check.Reason = "disabled in config"

		return check
	}

	ctx, cancel := context.WithTimeout(ctx, cliCheckBudget)
	defer cancel()

	infoOut, err := run(ctx, "info", "--format", podmanInfoFormat)
	if err != nil {
		// Older podman cannot render that template. Fall back to `--version` so a
		// too-old client still gets the accurate reason, then report the info
		// failure itself.
		versionOut, versionErr := run(ctx, "--version")
		if versionErr != nil {
			check.Reason = cliFailureReason(ctx, versionErr)
			check.Detail = versionErr.Error()

			return check
		}

		check.Detail = strings.TrimSpace(string(versionOut))

		version, parseErr := parsePodmanVersion(string(versionOut))
		if parseErr != nil {
			check.Reason = "podman version unknown"

			return check
		}

		check.Version = version

		if !versionAtLeast(version) {
			check.Reason = "podman too old"

			return check
		}

		check.Reason = cliFailureReason(ctx, err)

		return check
	}

	version, remoteSocket, ok := parsePodmanInfo(string(infoOut))
	if !ok {
		check.Reason = "podman version unknown"
		check.Detail = strings.TrimSpace(string(infoOut))

		return check
	}

	check.Version = version
	check.Detail = version

	if !versionAtLeast(version) {
		check.Reason = "podman too old"

		return check
	}

	if mode != PodsAlways {
		if reason := compareStores(socketPath, remoteSocket); reason != "" {
			check.Reason = reason

			return check
		}
	}

	check.Enabled = true

	return check
}

// parsePodmanInfo splits the `{{.Version.Version}}|{{.Host.RemoteSocket.Path}}`
// output. The socket may be empty (remote client), which is not a parse failure.
func parsePodmanInfo(out string) (version string, remoteSocket string, ok bool) {
	line := strings.TrimSpace(out)

	split := strings.SplitN(line, "|", 2)
	if len(split) != 2 {
		return "", "", false
	}

	version = strings.TrimSpace(split[0])
	if _, ok := parseVersionParts(version); !ok {
		return "", "", false
	}

	return version, strings.TrimSpace(split[1]), true
}

// compareStores proves that the CLI and the socket lazypodman talks to operate on
// the same store. It returns a reason code when they do not, and an empty string
// when the store is shared.
func compareStores(socketPath, remoteSocket string) string {
	normalizedSocket, comparable := normalizeSocketPath(socketPath)
	if !comparable {
		return "socket not comparable"
	}

	normalizedInfo, comparable := normalizeSocketPath(remoteSocket)
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

// cliPodmanClient is the only PodmanClient implementation: it shells out to the
// podman binary. The socket backend stays untouched; this exists for the data
// the Docker-compatible API cannot serve (pods).
type cliPodmanClient struct {
	binary string
	check  CLICheck

	// podLogsNames is probed on first use rather than at startup: a rootless
	// podman pays a full storage init per process and most sessions never open a
	// pod log tab. Guarded by podLogsNamesMutex, cached for the session.
	podLogsNamesMutex  sync.Mutex
	podLogsNamesProbed bool
	podLogsNames       bool
}

// newCLIPodmanClient builds the client and runs the one-time enablement check.
func newCLIPodmanClient(ctx context.Context, binary, socketPath string, mode PodsMode) *cliPodmanClient {
	client := &cliPodmanClient{binary: binary}
	client.check = detectCLI(ctx, socketPath, mode, client.run)

	return client
}

// Enabled reports whether the CLI can back the Podman-native features.
func (c *cliPodmanClient) Enabled() bool {
	return c.check.Enabled
}

// Reason is the short code shown as `pods off: <reason>`, empty when enabled.
func (c *cliPodmanClient) Reason() string {
	return c.check.Reason
}

// Version is the detected podman client version, empty when detection failed.
func (c *cliPodmanClient) Version() string {
	return c.check.Version
}

// run executes podman and returns its stdout. stderr is folded into the error so
// the UI can show podman's own message instead of a bare exit status.
func (c *cliPodmanClient) run(ctx context.Context, argv ...string) ([]byte, error) {
	return runPodmanCommand(ctx, c.binary, argv...)
}

// runPodmanCommand runs a one-shot podman command. It is not used for streaming:
// those need a reader plus kill-on-close (see PodLogs).
func runPodmanCommand(ctx context.Context, binary string, argv ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, argv...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Cancel = func() error { return cmd.Process.Kill() }
	cmd.WaitDelay = 2 * time.Second

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}

		return stdout.Bytes(), fmt.Errorf("%s %s: %w: %s", binary, strings.Join(argv, " "), err, message)
	}

	return stdout.Bytes(), nil
}

// validatePodRef rejects references empty or starting with a dash, so a name can
// never be interpreted as a flag by podman.
func validatePodRef(ref string) error {
	if ref == "" || strings.HasPrefix(ref, "-") {
		return fmt.Errorf("invalid pod reference %q", ref)
	}

	return nil
}

// podArgv builds the argv for a pod verb. The reference goes after `--` so names
// with spaces, quotes or other shell-ish characters stay one argument.
func podArgv(verb, ref string, extra ...string) ([]string, error) {
	if err := validatePodRef(ref); err != nil {
		return nil, err
	}

	argv := append([]string{"pod", verb}, extra...)

	return append(argv, "--", ref), nil
}

// PodList returns the pods of the podman store.
func (c *cliPodmanClient) PodList(ctx context.Context) ([]*Pod, error) {
	out, err := c.run(ctx, "pod", "ls", "--format", "json")
	if err != nil {
		return nil, err
	}

	pods, err := parsePods(out)
	if err != nil {
		return nil, err
	}

	for _, pod := range pods {
		pod.PodClient = c
		pod.PodmanBinary = c.binary
	}

	return pods, nil
}

// ContainerPodMap returns full container ID -> pod name. Uses `ps -a` because
// the Docker-compatible API reports PodName empty; `--external` is deliberately
// not passed: containers created outside Podman are out of scope.
func (c *cliPodmanClient) ContainerPodMap(ctx context.Context) (map[string]string, error) {
	out, err := c.run(ctx, "ps", "-a", "--format", "json")
	if err != nil {
		return nil, err
	}

	return parseContainerPodMap(out)
}

// PodInspect returns the JSON document of `pod inspect`.
func (c *cliPodmanClient) PodInspect(ctx context.Context, ref string) ([]byte, error) {
	if err := validatePodRef(ref); err != nil {
		return nil, err
	}

	return c.run(ctx, "pod", "inspect", "--", ref)
}

// PodAction runs a lifecycle verb (start, stop, restart, pause, unpause, rm).
func (c *cliPodmanClient) PodAction(ctx context.Context, verb, ref string, extra ...string) error {
	argv, err := podArgv(verb, ref, extra...)
	if err != nil {
		return err
	}

	_, err = c.run(ctx, argv...)

	return err
}

// PodPrune removes all stopped pods.
func (c *cliPodmanClient) PodPrune(ctx context.Context) error {
	_, err := c.run(ctx, "pod", "prune", "--force")

	return err
}

// maxPodLogProcesses caps the per-container log fallback: one process per
// container does not scale to a forty-container pod, and the UI cannot show that
// much anyway.
const maxPodLogProcesses = 8

// parsePodLogLine splits a merged log line produced with `--names` into the
// container name and the message. Names are matched at the start, longest first,
// followed by a single space, so a message like "webbish: hi" is not attributed
// to a container called "web".
func ParsePodLogLine(line string, names []string) (string, string) {
	sorted := make([]string, 0, len(names))
	for _, name := range names {
		if name != "" {
			sorted = append(sorted, name)
		}
	}

	sort.Slice(sorted, func(i, j int) bool { return len(sorted[i]) > len(sorted[j]) })

	for _, name := range sorted {
		if strings.HasPrefix(line, name+" ") {
			return name, strings.TrimPrefix(line, name+" ")
		}
	}

	return "", line
}

// streamCloser closes the reading end and kills the children once.
type streamCloser struct {
	reader io.ReadCloser
	close  func() error
	once   sync.Once
	err    error
}

func (s *streamCloser) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *streamCloser) Close() error {
	s.once.Do(func() { s.err = s.close() })

	return s.err
}

// PodLogs streams the merged logs of a pod. When podman advertises `--names` the
// whole pod is one process; otherwise each container gets its own process (and
// containers started after the follow begins are missed until the tab reloads).
//
// Closing the returned reader cancels the context, kills the children and waits
// for them, so switching tabs leaves no orphaned podman processes.
func (c *cliPodmanClient) PodLogs(ctx context.Context, pod *Pod, containers []*Container, tail int) (io.ReadCloser, error) {
	if pod == nil {
		return nil, errors.New("no pod selected")
	}

	if err := validatePodRef(pod.Name); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	reader, writer := io.Pipe()

	type command struct {
		argv []string
	}

	argvs, omitted := podLogArgvs(pod, containers, tail, c.podLogsNamesAvailable(ctx))
	if omitted > 0 {
		_, _ = writer.Write([]byte(fmt.Sprintf("# showing %d containers, %d omitted\n", len(argvs), omitted)))
	}

	commands := make([]command, 0, len(argvs))
	for _, argv := range argvs {
		commands = append(commands, command{argv: argv})
	}

	var wg sync.WaitGroup
	for _, cmd := range commands {
		wg.Add(1)

		go func(argv []string) {
			defer wg.Done()

			command := exec.CommandContext(ctx, c.binary, argv...)
			command.Cancel = func() error { return command.Process.Kill() }
			command.WaitDelay = 2 * time.Second

			stdout, err := command.StdoutPipe()
			if err != nil {
				_ = writer.CloseWithError(err)

				return
			}

			if err := command.Start(); err != nil {
				_ = writer.CloseWithError(err)

				return
			}

			scanner := bufio.NewScanner(stdout)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

			for scanner.Scan() {
				line := append(scanner.Bytes(), '\n')
				if _, err := writer.Write(line); err != nil {
					break
				}
			}

			_ = command.Wait()
		}(cmd.argv)
	}

	go func() {
		wg.Wait()
		_ = writer.Close()
	}()

	return &streamCloser{
		reader: reader,
		close: func() error {
			cancel()
			wg.Wait()

			return reader.Close()
		},
	}, nil
}

// ParsePodsMode maps the gui.pods.enabled config value to a PodsMode. Anything
// unrecognised means "auto", so a typo never silently disables the feature.
func ParsePodsMode(value string) PodsMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "always":
		return PodsAlways
	case "never":
		return PodsNever
	default:
		return PodsAuto
	}
}

// podLogArgvs decides which podman processes back the logs tab. merged keeps it
// to a single process (`pod logs --names`, Podman >= 4.1); otherwise one process
// per container, running containers first, capped at maxPodLogProcesses. omitted
// counts the containers the fallback cannot follow.
func podLogArgvs(pod *Pod, containers []*Container, tail int, merged bool) (argvs [][]string, omitted int) {
	tailArg := strconv.Itoa(tail)

	if merged {
		return [][]string{{"pod", "logs", "--follow", "--names", "--tail", tailArg, "--", pod.Name}}, 0
	}

	selected := make([]*Container, 0, len(containers))
	for _, ctr := range containers {
		if validatePodRef(ctr.Name) == nil {
			selected = append(selected, ctr)
		}
	}

	if len(selected) > maxPodLogProcesses {
		running := make([]*Container, 0, len(selected))
		stopped := make([]*Container, 0, len(selected))

		for _, ctr := range selected {
			if ctr.Container.State == "running" {
				running = append(running, ctr)
			} else {
				stopped = append(stopped, ctr)
			}
		}

		ordered := append(running, stopped...)
		omitted = len(ordered) - maxPodLogProcesses
		selected = ordered[:maxPodLogProcesses]
	}

	for _, ctr := range selected {
		argvs = append(argvs, []string{"logs", "--follow", "--names", "--tail", tailArg, "--", ctr.Name})
	}

	return argvs, omitted
}

// healthcheckBudget bounds one manual probe run. A hanging probe must not pin the
// waiting status forever.
const healthcheckBudget = 10 * time.Second

// healthcheckProbeBudget bounds the capability probe. It is a variable so tests
// can shrink it.
var healthcheckProbeBudget = 5 * time.Second

// ProbeHealthcheck asks the CLI whether it has a healthcheck subcommand. It never
// returns an error: the answer is the capability. A timeout is transient (retry
// later), a missing subcommand is not.
func (c *cliPodmanClient) ProbeHealthcheck(ctx context.Context) Capability {
	ctx, cancel := context.WithTimeout(ctx, healthcheckProbeBudget)
	defer cancel()

	_, err := c.run(ctx, "healthcheck", "run", "--help")
	if err == nil {
		return Capability{State: CapabilityAvailable}
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return Capability{
			State:     CapabilityUnavailable,
			Reason:    "probe timed out",
			Detail:    err.Error(),
			Transient: true,
		}
	}

	return Capability{
		State:  CapabilityUnavailable,
		Reason: "no healthcheck subcommand",
		Detail: err.Error(),
	}
}

// HealthcheckRun runs exactly one probe of the container's healthcheck.
//
// A real exit status is returned as a code with a nil error: exit 1 means the
// container is unhealthy, which is a result, not a failure. Only a process that
// could not run (or was killed) returns an error. Exit codes beyond the documented
// 0/1/2 are UNVERIFIED; callers treat anything else as an error.
func (c *cliPodmanClient) HealthcheckRun(ctx context.Context, ref string) (string, int, error) {
	if err := validatePodRef(ref); err != nil {
		return "", 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, healthcheckBudget)
	defer cancel()

	command := exec.CommandContext(ctx, c.binary, "healthcheck", "run", "--", ref)

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	command.Cancel = func() error { return command.Process.Kill() }
	command.WaitDelay = 2 * time.Second

	err := command.Run()
	output := stdout.String() + stderr.String()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		if code < 0 {
			// Killed by a signal (including our own timeout): no usable result.
			return output, 0, err
		}

		return output, code, nil
	}

	if err != nil {
		return output, 0, err
	}

	return output, 0, nil
}

// podLogsProbeBudget bounds the lazy `pod logs --help` probe.
var podLogsProbeBudget = 5 * time.Second

// podLogsNamesAvailable reports whether `podman pod logs` accepts `--names`.
// Probed once per session on the first pod log request, so startup does not pay
// for a feature most sessions never use.
func (c *cliPodmanClient) podLogsNamesAvailable(ctx context.Context) bool {
	c.podLogsNamesMutex.Lock()
	defer c.podLogsNamesMutex.Unlock()

	if c.podLogsNamesProbed {
		return c.podLogsNames
	}

	c.podLogsNamesProbed = true

	if !c.check.Enabled {
		return false
	}

	probeCtx, cancel := context.WithTimeout(ctx, podLogsProbeBudget)
	defer cancel()

	helpOut, err := c.run(probeCtx, "pod", "logs", "--help")
	if err != nil {
		// A cancelled probe (tab switched away) proves nothing; retry on the next
		// request instead of caching a false negative for the session.
		if ctx.Err() != nil || probeCtx.Err() != nil {
			c.podLogsNamesProbed = false
		}

		return false
	}

	c.podLogsNames = strings.Contains(string(helpOut), "--names")

	return c.podLogsNames
}
