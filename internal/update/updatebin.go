package update

import (
	"bufio"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"

	"github.com/McTalian/wow-build-tools/internal/logger"
)

const version = "LOCAL" // This will be replaced by the tag version in the binary
const repo = "McTalian/wow-build-tools"

func checkVersion() (semver.Version, error) {
	trimmedVersion := strings.TrimPrefix(version, "v")
	v, err := semver.Parse(trimmedVersion)
	if err != nil {
		logger.Debug("Running in local, alpha, or beta mode (%s). Skipping self-update.", trimmedVersion)
		return semver.Version{}, err
	}
	return v, nil
}

func ConfirmAndSelfUpdate() {
	v, err := checkVersion()
	if err != nil {
		return
	}

	latest, found, err := selfupdate.DetectLatest(repo)
	if err != nil {
		logger.Error("Error occurred while detecting version: %v", err)
		return
	}

	if !found || latest.Version.LTE(v) {
		logger.Debug("Current version is the latest")
		return
	}

	logger.Prompt("Do you want to update to %s? (y/N): ", latest.Version)
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil || (input != "y\n" && input != "n\n" && input != "\n") {
		logger.Error("Invalid input (%s), needed 'y' or 'n'", strings.TrimSuffix(input, "\n"))
		return
	}
	if input == "n\n" || input == "\n" {
		logger.Info("Skipping update, if you change your mind, run `wow-build-tools update` at any time")
		return
	}

	exe, err := os.Executable()
	if err != nil {
		logger.Error("Could not locate executable path: %v", err)
		return
	}
	if err := selfupdate.UpdateTo(latest.AssetURL, exe); err != nil {
		logger.Error("Error occurred while updating binary: %v", err)
		return
	}

	logger.Success("Successfully updated to version %s", latest.Version)
	logger.Info("Re-run the command to use the new version")
	os.Exit(0)
}

func DoSelfUpdate() {
	v, err := checkVersion()
	if err != nil {
		return
	}

	logger.Info("Checking for newer versions that %s...", v.String())
	latest, err := selfupdate.UpdateSelf(v, repo)
	if err != nil {
		logger.Error("Binary update failed: %v", err)
		return
	}
	if latest.Version.Equals(v) {
		// latest version is the same as current version. It means current binary is up to date.
		logger.Info("Current binary is the latest version %s", v.String())
	} else {
		logger.Info("Successfully updated to version %s", latest.Version)
		logger.Info("Release note:\n%s", latest.ReleaseNotes)
	}
}
