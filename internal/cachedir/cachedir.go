package cachedir

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/McTalian/wow-build-tools/internal/logger"
)

var cacheDir string

// Get returns the global cache directory for external repositories.
func Get() (string, error) {
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to determine user home directory: %w", err)
		}
		cacheDir = filepath.Join(homeDir, ".wow-build-tools", ".cache", "externals")
	}
	return cacheDir, nil
}

// Create creates the cache directory if it does not exist.
func Create() (string, error) {
	cacheDir, err := Get()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		logger.Verbose("Creating cache directory: %s", cacheDir)
		if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
			return "", fmt.Errorf("failed to create cache directory: %w", err)
		}
	}

	return cacheDir, nil
}

// Delete removes the entire cache directory.
func Delete() error {
	cacheDir, err := Get()
	if err != nil {
		return err
	}

	logger.Verbose("Removing cache directory: %s", cacheDir)
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(cacheDir)
}
