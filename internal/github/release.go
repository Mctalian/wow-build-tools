package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type GitHubRelease struct {
	GitHubReleasePayload
	Id   int `json:"id"`
	Slug string
}

type GitHubReleasePayload struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

func (r *GitHubRelease) getPayload() (*bytes.Buffer, error) {
	payload, err := json.Marshal(&r.GitHubReleasePayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal release: %w", err)
	}

	return bytes.NewBuffer(payload), nil
}

func (r *GitHubRelease) CreateRelease() error {
	url := fmt.Sprintf("%srepos/%s/releases", githubApiUrl, r.Slug)

	body, err := r.getPayload()
	if err != nil {
		return fmt.Errorf("failed to marshal release: %w", err)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	addAcceptHeader(req)

	err = addAuthHeader(req)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("failed to get request: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create release: %d", resp.StatusCode)
	}

	return nil
}

func (r *GitHubRelease) UpdateRelease() error {
	url := fmt.Sprintf("%srepos/%s/releases/%d", githubApiUrl, r.Slug, r.Id)

	body, err := r.getPayload()
	if err != nil {
		return fmt.Errorf("failed to marshal release: %w", err)
	}

	req, err := http.NewRequest("PATCH", url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	addAcceptHeader(req)

	err = addAuthHeader(req)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("failed to get request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update release: %d", resp.StatusCode)
	}

	return nil
}

func GetRelease(slug, tag string) (release *GitHubRelease, err error) {
	url := fmt.Sprintf("%srepos/%s/releases/tags/%s", githubApiUrl, slug, tag)

	req, err := http.NewRequest("GET", url, nil)

	addAcceptHeader(req)

	err = addAuthHeader(req)
	if err != nil {
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		return
	}

	if resp.StatusCode == http.StatusOK {
		err = json.NewDecoder(resp.Body).Decode(&release)
		if err != nil {
			return
		}
		release.Slug = slug

		return
	}

	return
}
