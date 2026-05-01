//go:build linux

package config

import (
	"os"
	"path/filepath"
)

// userConfigDir returns the per-user config dir for Tomoe on Linux.
// Honors XDG_CONFIG_HOME, falling back to ~/.config/tomoe.
func userConfigDir() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.Getenv("HOME")
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "tomoe")
}

// userDataDir returns the per-user data dir for Tomoe on Linux.
// Honors XDG_DATA_HOME, falling back to ~/.local/share/tomoe.
func userDataDir() string {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.Getenv("HOME")
		}
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "tomoe")
}
