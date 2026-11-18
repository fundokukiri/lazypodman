package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// kubeKindScanLines is how far into a file we look for the top-level `kind:`.
const kubeKindScanLines = 20

// KubeFile is one YAML manifest offered by the kube menu.
type KubeFile struct {
	// Name includes the kind, so the menu shows pod/deployment at a glance.
	Name string
	Path string
}

// ScanKubeDir lists the kube manifests in dir, non-recursively, sorted by name.
// A file without a top-level kind: is not a manifest and is skipped silently.
func ScanKubeDir(dir string) ([]KubeFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read kube dir %s: %w", dir, err)
	}

	files := make([]KubeFile, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".yaml", ".yml":
		default:
			continue
		}

		path := filepath.Join(dir, entry.Name())

		kind, err := sniffKubeKind(path)
		if err != nil || kind == "" {
			continue
		}

		files = append(files, KubeFile{Name: entry.Name() + " (" + kind + ")", Path: path})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	return files, nil
}

// sniffKubeKind returns the value of a top-level `kind:` line, or "" when the file
// does not look like a kube manifest.
func sniffKubeKind(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for line := 0; line < kubeKindScanLines && scanner.Scan(); line++ {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// A nested key (indented) is not the document kind.
		if len(trimmed) != len(raw) || !strings.HasPrefix(trimmed, "kind:") {
			continue
		}

		kind := strings.TrimSpace(strings.TrimPrefix(trimmed, "kind:"))
		kind = strings.Trim(kind, `"'`)

		return kind, nil
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}

// SafeKubeFilename reduces a pod name to one safe path segment: no separators, no
// leading dots, never empty.
func SafeKubeFilename(name string) string {
	var builder strings.Builder

	for _, char := range name {
		switch {
		case char >= 'a' && char <= 'z',
			char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9',
			char == '.', char == '_', char == '-':
			builder.WriteRune(char)
		default:
			builder.WriteRune('-')
		}
	}

	safe := strings.Trim(builder.String(), ".")
	if safe == "" {
		return "pod"
	}

	return safe
}

// WriteManifestFile writes a generated manifest (kube YAML or Quadlet unit) with mode 0600, creating the directory
// if needed. An existing file is only replaced when overwrite is true; the caller
// must have asked the user first.
func WriteManifestFile(path string, data []byte, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file already exists: %s", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	if _, err := file.Write(data); err != nil {
		_ = file.Close()

		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	// An existing file keeps its old mode through O_CREATE; tighten it.
	return os.Chmod(path, 0o600)
}
