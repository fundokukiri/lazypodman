package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

type normalizeCase struct {
	in         string
	want       string
	comparable bool
}

func TestNormalizeSocketPath(t *testing.T) {
	cases := []normalizeCase{
		{"/run/user/1000/podman/podman.sock", "/run/user/1000/podman/podman.sock", true},
		{"unix:///run/user/1000/podman/podman.sock", "/run/user/1000/podman/podman.sock", true},
		{"unix:///run/user/1000/podman/podman.sock/", "/run/user/1000/podman/podman.sock", true},
		{"unix:////run/user/1000/podman/podman.sock", "/run/user/1000/podman/podman.sock", true},
		{"/run/user/1000/podman/./podman.sock", "/run/user/1000/podman/podman.sock", true},
		// nonexistent path: EvalSymlinks fails, the cleaned path is used
		{"/does/not/exist/../podman.sock", "/does/not/podman.sock", true},
		{"  /run/user/1000/podman/podman.sock  ", "/run/user/1000/podman/podman.sock", true},
		{"", "", false},
		{"   ", "", false},
		{"fd://3", "", false},
		{"ssh://core@host/run/user/1000/podman/podman.sock", "", false},
		{"tcp://10.0.0.1:2375", "", false},
	}

	for _, c := range cases {
		got, comparable := normalizeSocketPath(c.in)
		if got != c.want || comparable != c.comparable {
			t.Errorf("normalizeSocketPath(%q) = (%q, %v), want (%q, %v)", c.in, got, comparable, c.want, c.comparable)
		}
	}
}

func TestNormalizeSocketPathResolvesSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "podman.sock")
	link := filepath.Join(dir, "link.sock")
	requireNoError(t, os.WriteFile(target, nil, 0o600))
	requireNoError(t, os.Symlink(target, link))

	wantPath, wantComparable := normalizeSocketPath(target)
	gotPath, gotComparable := normalizeSocketPath("unix://" + link)

	assert.True(t, wantComparable)
	assert.True(t, gotComparable)
	assert.Equal(t, wantPath, gotPath)
}

func TestNormalizeSocketPathWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}

	got, comparable := normalizeSocketPath("npipe:////./pipe/podman")
	want, wantComparable := normalizeSocketPath(`\\.\pipe\podman`)

	assert.True(t, comparable)
	assert.True(t, wantComparable)
	assert.Equal(t, want, got)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}
