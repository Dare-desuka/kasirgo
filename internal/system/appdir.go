package system

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// GetAppDataDir returns the per-user application data directory for pos-go.
// Database location MUST NOT depend on the working directory.
func GetAppDataDir() (string, error) {
	if runtime.GOOS == "linux" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		return filepath.Join(home, ".local", "share", "pos-go"), nil
	}
	// Windows: %APPDATA%\pos-go. Other OSes: ~/.config/pos-go.
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(cfg, "pos-go"), nil
}

// GetDatabasePath returns the absolute path to the SQLite database file.
func GetDatabasePath() (string, error) {
	dir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "data.db"), nil
}

// EnsureDir creates the directory if it does not exist.
func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}
	return nil
}