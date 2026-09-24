// Package paths resolves the config file and state directory for the running OS.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Expand replaces a leading "~" with the user's home directory.
func Expand(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

// ConfigDir returns the directory that holds config.json, honouring XDG_CONFIG_HOME / APPDATA.
func ConfigDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if v := os.Getenv("APPDATA"); v != "" {
			return filepath.Join(v, "cla-notify"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AppData", "Roaming", "cla-notify"), nil
	default:
		if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
			return filepath.Join(v, "cla-notify"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "cla-notify"), nil
	}
}

// ConfigPath returns the config file path, honouring CLA_NOTIFY_CONFIG.
func ConfigPath() (string, error) {
	if v := os.Getenv("CLA_NOTIFY_CONFIG"); v != "" {
		return Expand(v), nil
	}
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// StateDir returns the directory cla-notify keeps session state, cards and logs in.
func StateDir() (string, error) {
	if v := os.Getenv("CLA_NOTIFY_STATE_DIR"); v != "" {
		return Expand(v), nil
	}
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "cla-notify"), nil
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "cla-notify"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AppData", "Local", "cla-notify"), nil
	default:
		if v := os.Getenv("XDG_STATE_HOME"); v != "" {
			return filepath.Join(v, "cla-notify"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "state", "cla-notify"), nil
	}
}
