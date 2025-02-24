package configdir

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/McTalian/wow-build-tools/internal/github"
	"github.com/McTalian/wow-build-tools/internal/logger"
)

var externalsCacheDir string

var configDir = ".wow-build-tools"
var cachePath = filepath.Join(configDir, ".cache")
var externalsPath = filepath.Join(cachePath, "externals")

func getHomeDir() (string, error) {
	if github.IsGitHubAction() {
		return github.GetRunnerTempDir()
	}
	return os.UserHomeDir()
}

// GetConfigDir returns the global configuration directory.
func GetConfigDir() (string, error) {
	dir, err := getHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine location for configuration directory: %w", err)
	}
	return filepath.Join(dir, configDir), nil
}

// CreateConfigDir creates the configuration directory if it does not exist.
func CreateConfigDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		logger.Verbose("Creating configuration directory: %s", configDir)
		if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
			return "", fmt.Errorf("failed to create configuration directory: %w", err)
		}
	}

	return configDir, nil
}

// DeleteConfigDir removes the entire configuration directory.
func DeleteConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	logger.Verbose("Removing configuration directory: %s", configDir)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(configDir)
}

// GetExternalsCache returns the global cache directory for external repositories.
func GetExternalsCache() (string, error) {
	if externalsCacheDir == "" {
		dir, err := getHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to determine location for cache directory: %w", err)
		}
		externalsCacheDir = filepath.Join(dir, externalsPath)
	}
	return externalsCacheDir, nil
}

// CreateExternalsCache creates the cache directory if it does not exist.
func CreateExternalsCache() (string, error) {
	cacheDir, err := GetExternalsCache()
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

// DeleteExternalsCache removes the entire cache directory.
func DeleteExternalsCache() error {
	cacheDir, err := GetExternalsCache()
	if err != nil {
		return err
	}

	logger.Verbose("Removing cache directory: %s", cacheDir)
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(cacheDir)
}
