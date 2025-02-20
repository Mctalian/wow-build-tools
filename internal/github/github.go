package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/McTalian/wow-build-tools/internal/logger"
)

var githubApiUrl = "https://api.github.com/"

type releaseResponse struct {
	Id string `json:"id"`
}

func GetReleaseId(slug, tag string) (releaseId string, err error) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		logger.Error("GITHUB_TOKEN not set")
		err = fmt.Errorf("GITHUB_TOKEN not set")
		return
	}

	url := fmt.Sprintf("%srepos/%s/releases/tags/%s", githubApiUrl, slug, tag)

	req, err := http.NewRequest("GET", url, nil)

	req.Header.Set("Authorization", fmt.Sprintf("token %s", os.Getenv("GITHUB_TOKEN")))

	resp, err := http.Get(url)
	if err != nil {
		return
	}

	if resp.StatusCode == 200 {
		var release releaseResponse
		err = json.NewDecoder(resp.Body).Decode(&release)
		if err != nil {
			return
		}
		releaseId = release.Id
		return
	}

	return
}

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
