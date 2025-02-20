package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/McTalian/wow-build-tools/internal/logger"
)

var githubApiUrl = "https://api.github.com/"
var authHeaderValue string

type GitHubRelease struct {
	Id         int    `json:"id"`
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

func getAuthHeaderValue() (string, error) {
	if authHeaderValue == "" {
		if os.Getenv("GITHUB_OAUTH") == "" {
			logger.Error("GITHUB_OAUTH not set")
			err := fmt.Errorf("GITHUB_OAUTH not set")
			return "", err
		}

		authHeaderValue = fmt.Sprintf("token %s", os.Getenv("GITHUB_OAUTH"))
	}

	return authHeaderValue, nil
}

func GetRelease(slug, tag string) (release GitHubRelease, err error) {
	url := fmt.Sprintf("%srepos/%s/releases/tags/%s", githubApiUrl, slug, tag)

	req, err := http.NewRequest("GET", url, nil)

	tokenValue, err := getAuthHeaderValue()
	if err != nil {
		return
	}
	req.Header.Add("Authorization", tokenValue)

	resp, err := http.Get(url)
	if err != nil {
		return
	}

	if resp.StatusCode == 200 {
		err = json.NewDecoder(resp.Body).Decode(&release)
		if err != nil {
			return
		}
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
