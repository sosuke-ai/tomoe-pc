//go:build darwin

package config

import (
	"os"
	"path/filepath"
)

// userConfigDir returns the per-user config dir for Tomoe on macOS.
// macOS conventions put config under ~/Library/Application Support/<App>/.
// Apple explicitly recommends against ~/Library/Preferences for app-managed
// config files (that path is for NSUserDefaults plists).
func userConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, "Library", "Application Support", "Tomoe")
}

// userDataDir returns the per-user data dir for Tomoe on macOS.
// Same root as userConfigDir — macOS keeps config and data co-located.
func userDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, "Library", "Application Support", "Tomoe")
}
