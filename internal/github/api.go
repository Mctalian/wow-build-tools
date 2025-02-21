package github

import (
	"fmt"
	"net/http"
	"os"

	"github.com/McTalian/wow-build-tools/internal/logger"
)

var githubApiUrl = "https://api.github.com/"
var authHeaderValue string

func IsTokenSet() bool {
	if os.Getenv("GITHUB_OAUTH") == "" {
		return false
	}

	return true
}

func getAuthHeaderValue() (string, error) {
	if authHeaderValue == "" {
		if !IsTokenSet() {
			logger.Error("GITHUB_OAUTH not set")
			err := fmt.Errorf("GITHUB_OAUTH not set")
			return "", err
		}

		authHeaderValue = fmt.Sprintf("token %s", os.Getenv("GITHUB_OAUTH"))
	}

	return authHeaderValue, nil
}

func addAcceptHeader(req *http.Request) {
	req.Header.Add("Accept", "application/vnd.github.v3+json")
}

func addAuthHeader(req *http.Request) error {
	tokenValue, err := getAuthHeaderValue()
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", tokenValue)

	return nil
}
