package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanKubeDir(t *testing.T) {
	dir := t.TempDir()

	write := func(name, content string) {
		requireNoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}

	write("pod.yaml", "apiVersion: v1\nkind: Pod\nmetadata:\n  name: web\n")
	write("deploy.yml", "# a comment\n\napiVersion: apps/v1\nkind: Deployment\n")
	write("no-kind.yaml", "apiVersion: v1\nmetadata:\n  name: nope\n")
	write("nested-kind.yaml", "spec:\n  kind: NotTopLevel\n")
	write("readme.txt", "kind: Pod\n")
	requireNoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	write(filepath.Join("sub", "nested.yaml"), "kind: Pod\n")

	files, err := ScanKubeDir(dir)
	requireNoError(t, err)

	assert.Len(t, files, 2)
	assert.Equal(t, "deploy.yml (Deployment)", files[0].Name)
	assert.Equal(t, filepath.Join(dir, "deploy.yml"), files[0].Path)
	assert.Equal(t, "pod.yaml (Pod)", files[1].Name)
}

func TestScanKubeDirQuotedKind(t *testing.T) {
	dir := t.TempDir()
	requireNoError(t, os.WriteFile(filepath.Join(dir, "p.yaml"), []byte(`kind: "Pod"`+"\n"), 0o600))

	files, err := ScanKubeDir(dir)
	requireNoError(t, err)

	assert.Len(t, files, 1)
	assert.Equal(t, "p.yaml (Pod)", files[0].Name)
}

func TestScanKubeDirMissing(t *testing.T) {
	_, err := ScanKubeDir(filepath.Join(t.TempDir(), "nope"))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nope")
}

func TestSafeKubeFilename(t *testing.T) {
	cases := map[string]string{
		"web":     "web",
		"my pod":  "my-pod",
		"a/b":     "a-b",
		"..":      "pod",
		"":        "pod",
		".hidden": "hidden",
		"web_1.2": "web_1.2",
	}

	for input, want := range cases {
		assert.Equal(t, want, SafeKubeFilename(input), input)
	}

	for _, hostile := range []string{"../../etc/passwd", "a/../b", "/etc/passwd", `..\\..\\x`} {
		safe := SafeKubeFilename(hostile)

		assert.NotContains(t, safe, "/", hostile)
		assert.NotEqual(t, ".", safe, hostile)
		assert.NotEqual(t, "..", safe, hostile)
		assert.False(t, strings.HasPrefix(safe, "."), hostile)
	}
}

func TestWriteKubeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "pod.yaml")

	requireNoError(t, WriteKubeFile(path, []byte("kind: Pod\n"), false))

	content, err := os.ReadFile(path)
	requireNoError(t, err)
	assert.Equal(t, "kind: Pod\n", string(content))

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		requireNoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestWriteKubeFileRefusesOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pod.yaml")
	requireNoError(t, WriteKubeFile(path, []byte("old\n"), false))

	err := WriteKubeFile(path, []byte("new\n"), false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	content, readErr := os.ReadFile(path)
	requireNoError(t, readErr)
	assert.Equal(t, "old\n", string(content))

	requireNoError(t, WriteKubeFile(path, []byte("new\n"), true))

	content, readErr = os.ReadFile(path)
	requireNoError(t, readErr)
	assert.Equal(t, "new\n", string(content))

	if runtime.GOOS != "windows" {
		info, statErr := os.Stat(path)
		requireNoError(t, statErr)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "an overwrite must tighten a loose mode")
	}
}
