package osutil

import (
	"os"
	"os/exec"
	"strings"
)

func IsWSL() bool {
	if _, err := os.Stat("/proc/version"); err != nil {
		return false
	}

	contents, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}

	return strings.Contains(strings.ToLower(string(contents)), "microsoft")
}

func GetWindowsPath(path string) (string, error) {
	if !IsWSL() {
		return path, nil
	}

	wslPathCmd := exec.Command("wslpath", "-w", path)
	wslPathCmd.Stderr = nil
	output, err := wslPathCmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
