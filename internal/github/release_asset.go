package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/McTalian/wow-build-tools/internal/logger"
)

type GitHubReleaseAsset struct {
	Name string `json:"name"`
	Id   int    `json:"id"`
	Url  string `json:"url"`
}

func (ghRA *GitHubReleaseAsset) downloadAsset(logGroup *logger.LogGroup) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", ghRA.Url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	err = addAuthHeader(req)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		logGroup.Info("Successfully downloaded asset %s", ghRA.Name)
		return resp.Body, nil
	}

	return nil, fmt.Errorf("failed to download asset %s: %s", ghRA.Name, resp.Status)
}

func getAssetId(slug string, releaseId int, filename string) (int, error) {
	url := fmt.Sprintf("%srepos/%s/releases/%d/assets", githubApiUrl, slug, releaseId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return -1, fmt.Errorf("failed to create request: %w", err)
	}

	addAcceptHeader(req)

	err = addAuthHeader(req)
	if err != nil {
		return -1, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return -1, fmt.Errorf("failed to get request: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		var assets []GitHubReleaseAsset
		err = json.NewDecoder(resp.Body).Decode(&assets)
		if err != nil {
			return -1, fmt.Errorf("failed to decode response: %w", err)
		}

		for _, asset := range assets {
			if asset.Name == filename {
				return asset.Id, nil
			}
		}
	}

	return -1, nil
}

func getAsset(slug string, assetId int) (*GitHubReleaseAsset, error) {
	url := fmt.Sprintf("%srepos/%s/releases/assets/%d", githubApiUrl, slug, assetId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	addAcceptHeader(req)

	err = addAuthHeader(req)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		var asset GitHubReleaseAsset
		err = json.NewDecoder(resp.Body).Decode(&asset)
		if err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return &asset, nil
	}

	return nil, fmt.Errorf("failed to get asset %d: %s", assetId, resp.Status)
}

func deleteAsset(slug string, assetId int, logGroup *logger.LogGroup) error {
	url := fmt.Sprintf("%srepos/%s/releases/assets/%d", githubApiUrl, slug, assetId)

	req, err := http.NewRequest("DELETE", url, nil)
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
		return fmt.Errorf("failed to delete request: %w", err)
	}

	if resp.StatusCode == http.StatusNoContent {
		logGroup.Info("Successfully deleted asset %d", assetId)
		return nil
	}

	return fmt.Errorf("failed to delete asset %d: %s", assetId, resp.Status)
}

func UploadGitHubAsset(slug string, releaseId int, filename string, filePath string, logGroup *logger.LogGroup) error {
	assetId, err := getAssetId(slug, releaseId, filename)
	if err != nil {
		return err
	}

	if assetId != -1 {
		logGroup.Verbose("Asset %s already exists in release %d", filename, releaseId)
		err = deleteAsset(slug, assetId, logGroup)
		if err != nil {
			logGroup.Error("Failed to delete asset %d: %v", assetId, err)
			return err
		}
	}

	encodedFilename := url.QueryEscape(filename)
	url := fmt.Sprintf("%srepos/%s/releases/%d/assets?name=%s", githubUploadUrl, slug, releaseId, encodedFilename)

	file, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	req, err := http.NewRequest("POST", url, file)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Get file size and set Content-Length manually.
	fi, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	req.ContentLength = fi.Size()

	fileExtension := strings.TrimPrefix(filepath.Ext(filePath), ".")
	fileContentType := "application/" + fileExtension

	addAcceptHeader(req)
	req.Header.Add("Content-Type", fileContentType)

	err = addAuthHeader(req)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("failed to post request: %w", err)
	}

	if resp.StatusCode == http.StatusCreated {
		logGroup.Info("Successfully uploaded %s to release %d", filename, releaseId)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	logGroup.Verbose("%s", string(body))

	return fmt.Errorf("failed to upload %s to release %d: %s", filename, releaseId, resp.Status)
}
