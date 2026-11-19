//go:build !windows && !darwin

package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExternalTerminalArgvUsesTerminalEnvironment(t *testing.T) {
	t.Setenv("TERMINAL", "myterm --class podman")

	got, err := externalTerminalArgv([]string{"podman", "attach", "abc"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"myterm", "--class", "podman", "-e", "podman", "attach", "abc"}, got)
}

func TestOpenExternalTerminalStartsDetachedLauncher(t *testing.T) {
	dir := t.TempDir()
	argvFile := filepath.Join(dir, "argv.log")
	terminal := filepath.Join(dir, "myterm")

	script := "#!/bin/sh\nfor arg in \"$@\"; do printf '%s\\n' \"$arg\"; done > \"$ARGV_FILE\"\n"
	if err := os.WriteFile(terminal, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TERMINAL", terminal)
	t.Setenv("ARGV_FILE", argvFile)

	osCommand := NewOSCommand(NewDummyLog(), NewDummyAppConfig())
	if err := osCommand.OpenExternalTerminal([]string{"podman", "attach", "--sig-proxy=false", "abc123"}); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(argvFile); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the terminal launcher")
		}
		time.Sleep(10 * time.Millisecond)
	}

	got, err := os.ReadFile(argvFile)
	assert.NoError(t, err)
	assert.Equal(t, strings.Join([]string{
		"-e",
		"podman",
		"attach",
		"--sig-proxy=false",
		"abc123",
		"",
	}, "\n"), string(got))
}

func TestAttachTerminalScriptPrintsLogsThenAttaches(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "podman")
	script := `#!/bin/sh
case "$1" in
  logs) echo LOGS-HISTORY ;;
  attach) echo ATTACH-LIVE ;;
esac
`
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	argv := AttachTerminalArgv(fake, "abc123")
	out, err := exec.Command(argv[0], argv[1:]...).CombinedOutput()
	assert.NoError(t, err)
	assert.Equal(t, "LOGS-HISTORY\nATTACH-LIVE\n", string(out))
}

func TestAttachTerminalArgvReplaysLogsBeforeAttach(t *testing.T) {
	got := AttachTerminalArgv("/usr/bin/podman", "abc123")

	assert.Equal(t, "sh", got[0])
	assert.Equal(t, "-c", got[1])
	assert.Equal(t, "lazypodman-attach", got[3])
	assert.Equal(t, "/usr/bin/podman", got[4])
	assert.Equal(t, "abc123", got[5])

	script := got[2]
	assert.Contains(t, script, `"$bin" logs "$ref"`)
	assert.Contains(t, script, `exec "$bin" attach --sig-proxy=false "$ref"`)
	assert.Less(t, strings.Index(script, " logs "), strings.Index(script, " attach "))
}

func TestCurrentTerminalAttachArgvClearsLogsThenAttaches(t *testing.T) {
	got := CurrentTerminalAttachArgv("/usr/bin/podman", "abc123")

	assert.Equal(t, "sh", got[0])
	assert.Equal(t, "-c", got[1])
	assert.Equal(t, "lazypodman-attach", got[3])
	assert.Equal(t, "/usr/bin/podman", got[4])
	assert.Equal(t, "abc123", got[5])

	script := got[2]
	assert.Contains(t, script, `printf '\033[H\033[2J\033[3J'`)
	assert.Contains(t, script, `"$bin" logs "$ref"`)
	assert.Contains(t, script, `exec "$bin" attach --sig-proxy=false "$ref"`)
	assert.Less(t, strings.Index(script, "printf"), strings.Index(script, " logs "))
	assert.Less(t, strings.Index(script, " logs "), strings.Index(script, " attach "))
}
