package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunContainerDefaults(t *testing.T) {
	state := newRunContainerState("nginx:alpine")
	args := state.argv("")

	assert.Equal(t, "run", args[0])
	assert.NotContains(t, args, "--name")
	assert.Contains(t, args, "-d")
	assert.Equal(t, "nginx:alpine", args[len(args)-1])
}

func TestRunContainerBuildsEveryKindOfField(t *testing.T) {
	state := newRunContainerState("nginx:alpine")
	state.text["name"] = "web"
	state.text["command"] = "nginx -g 'daemon off;'"
	state.text["restart"] = "unless-stopped"
	state.lists["ports"] = []string{"8080:80", "127.0.0.1:8443:443"}
	state.lists["env"] = []string{"A=1"}
	state.toggles["rm"] = true
	state.toggles["interactive"] = true
	state.toggles["tty"] = true
	state.text["pull"] = "always"

	args := state.argv("--annotation note=hello")

	assert.Equal(t, []string{
		"run",
		"--name", "web",
		"-p", "8080:80",
		"-p", "127.0.0.1:8443:443",
		"-e", "A=1",
		"--restart", "unless-stopped",
		"-d",
		"--rm",
		"-i",
		"-t",
		"--pull", "always",
		"--annotation", "note=hello",
		"nginx:alpine",
		"nginx", "-g", "daemon off;",
	}, args)
}

func TestRunContainerExtraArgsStayBeforeTheImage(t *testing.T) {
	state := newRunContainerState("alpine")
	delete(state.text, "name")

	args := state.argv("--pull=always --annotation a=b")

	assert.Equal(t, []string{"run", "-d", "--pull=always", "--annotation", "a=b", "alpine"}, args)
}

func TestRunContainerEqualsFlags(t *testing.T) {
	state := newRunContainerState("alpine")
	state.text["http_proxy"] = "false"
	state.text["sig_proxy"] = "false"

	args := state.argv("")

	assert.Contains(t, args, "--http-proxy=false")
	assert.Contains(t, args, "--sig-proxy=false")
}

func TestRunContainerNoNewPrivilegesUsesSecurityOpt(t *testing.T) {
	state := newRunContainerState("alpine")
	state.toggles["no_new_privileges"] = true

	assert.Contains(t, state.argv(""), "no-new-privileges")
}

func TestRunContainerListEditHelpers(t *testing.T) {
	state := newRunContainerState("alpine")

	state.addListValue("ports", "8080:80")
	state.addListValue("ports", "   ")
	state.addListValue("ports", "9090:90")
	state.setListValue("ports", 0, "8081:81")
	state.removeListValue("ports", 1)
	state.removeListValue("ports", 99)

	assert.Equal(t, []string{"8081:81"}, state.lists["ports"])
}

func TestRunContainerEmptyValuesDoNotReachArgv(t *testing.T) {
	state := newRunContainerState("alpine")
	state.text["name"] = "   "
	state.lists["ports"] = []string{"", "  "}

	assert.Equal(t, []string{"run", "-d", "alpine"}, state.argv("  "))
}

func TestRunContainerSchemaKeysAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, field := range runContainerFields {
		assert.False(t, seen[field.Key], "duplicate key %q", field.Key)
		seen[field.Key] = true
	}
	for _, group := range runContainerGroups {
		for _, field := range group.Fields {
			assert.False(t, seen[field.Key], "duplicate key %q", field.Key)
			seen[field.Key] = true
		}
	}
}

func TestRunFieldByKeyFindsAdvancedFields(t *testing.T) {
	field := runFieldByKey("health_cmd")

	assert.Equal(t, "--health-cmd", field.Flag)
}
