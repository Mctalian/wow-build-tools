package github

import (
	"fmt"
	"os"

	"github.com/McTalian/wow-build-tools/internal/logger"
)

func IsGitHubAction() bool {
	return os.Getenv("CI") == "true" && os.Getenv("GITHUB_ACTIONS") == "true"
}

func GetRunnerTempDir() (string, error) {
	if IsGitHubAction() {
		return os.Getenv("RUNNER_TEMP"), nil
	}

	return "", fmt.Errorf("not running in GitHub Actions")
}

func Output(name, value string) error {
	if IsGitHubAction() {
		output_file, ok := os.LookupEnv("GITHUB_OUTPUT")
		if !ok {
			logger.Warn("GITHUB_OUTPUT not set, cannot write output")
			return nil
		}
		if _, err := os.Stat(output_file); err != nil {
			return err
		}
		f, err := os.OpenFile(output_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open output file: %w", err)
		}
		defer f.Close()

		_, err = f.WriteString(fmt.Sprintf("%s=%s\n", name, value))
		if err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	return nil
}
