package config

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/jesseduffield/yaml"
)

// RemovalLock lists the containers and images the user marked as "do not delete
// by accident". Deleting one asks for part of its ID first. It guards against a
// mis-pressed key, not against a person: the file is plain text and the user can
// edit it.
type RemovalLock struct {
	Containers []string `yaml:"containers,omitempty"`
	Images     []string `yaml:"images,omitempty"`
}

// RemovalLockPath is the list's file, kept next to config.yml.
func (c *AppConfig) RemovalLockPath() string {
	return filepath.Join(c.ConfigDir, "removal-lock.yml")
}

// LoadRemovalLock returns the saved list, or an empty one when there is none.
func (c *AppConfig) LoadRemovalLock() (*RemovalLock, error) {
	lock := &RemovalLock{}

	content, err := os.ReadFile(c.RemovalLockPath())
	if err != nil {
		if os.IsNotExist(err) {
			return lock, nil
		}

		return lock, err
	}

	if err := yaml.Unmarshal(content, lock); err != nil {
		return &RemovalLock{}, err
	}

	return lock, nil
}

// SaveRemovalLock writes the list back, creating the config directory if needed.
func (c *AppConfig) SaveRemovalLock(lock *RemovalLock) error {
	if err := os.MkdirAll(c.ConfigDir, 0o755); err != nil {
		return err
	}

	content, err := yaml.Marshal(lock)
	if err != nil {
		return err
	}

	return os.WriteFile(c.RemovalLockPath(), content, 0o600)
}

// ContainerLocked reports whether the container ID is on the list.
func (l *RemovalLock) ContainerLocked(id string) bool {
	return slices.Contains(l.Containers, id)
}

// ImageLocked reports whether the image ID is on the list.
func (l *RemovalLock) ImageLocked(id string) bool {
	return slices.Contains(l.Images, id)
}

// ToggleContainer adds or removes the container ID and returns true when it is
// now locked.
func (l *RemovalLock) ToggleContainer(id string) bool {
	l.Containers = toggleID(l.Containers, id)

	return slices.Contains(l.Containers, id)
}

// ToggleImage adds or removes the image ID and returns true when it is now
// locked.
func (l *RemovalLock) ToggleImage(id string) bool {
	l.Images = toggleID(l.Images, id)

	return slices.Contains(l.Images, id)
}

func toggleID(list []string, id string) []string {
	if slices.Contains(list, id) {
		return slices.DeleteFunc(list, func(s string) bool { return s == id })
	}

	return append(list, id)
}
