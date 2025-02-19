package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/McTalian/wow-build-tools/internal/changelog"
	f "github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/toc"
)

var ErrNoWowiId = fmt.Errorf("no WoW Interface ID provided")
var ErrMultipleWowiIds = fmt.Errorf("multiple WoW Interface IDs found")
var ErrNoWowiUpload = fmt.Errorf("WoW Interface upload is disabled")
var ErrNoWowiApiKey = fmt.Errorf("WOWI_API_TOKEN not set")

var wowiApiUrl = "https://api.wowinterface.com/addons/"
var wowiGameVersionsUrl = fmt.Sprintf("%scompatible.json", wowiApiUrl)
var wowiUploadUrl = fmt.Sprintf("%supdate", wowiApiUrl)

type wowiGameVersionsEntry struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Game      string `json:"game"`
	Interface string `json:"interface"`
	Default   bool   `json:"default"`
}

func locateWowiId(tocFiles []*toc.Toc) (wowiId string, err error) {
	var foundWowiId string
	for _, tocFile := range tocFiles {
		if tocFile.WowiId != "" {
			if foundWowiId != "" && foundWowiId != tocFile.WowiId {
				err = ErrMultipleWowiIds
				return
			}
			foundWowiId = tocFile.WowiId
		}
	}

	if foundWowiId == "" {
		err = ErrNoWowiId
		return
	}

	wowiId = foundWowiId
	return
}

func getWowiId(tocFiles []*toc.Toc) (wowiId string, err error) {
	if f.SkipUpload {
		err = ErrNoWowiUpload
		return
	}

	if f.WowiId != "" {
		if f.WowiId == "0" {
			wowiId = ""
		} else {
			wowiId = f.WowiId
		}
	} else {
		wowiId, err = locateWowiId(tocFiles)
		if err == ErrMultipleWowiIds {
			return
		}
	}
	if wowiId == "" {
		err = ErrNoWowiId
		return
	}

	return
}

type wowiUpload struct {
	token       string
	projectId   string
	zipFile     string
	displayName string
	changelog   *changelog.Changelog
	compatible  []string
	version     string
	archiveOld  bool
	logGroup    *logger.LogGroup
}

func (w *wowiUpload) lookupWowiToken() (err error) {
	token, found := os.LookupEnv("WOWI_API_TOKEN")
	if !found {
		err = ErrNoWowiApiKey
		return
	}

	w.token = token

	return
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func (w *wowiUpload) validateGameVersions(gameVersions []string) error {
	resp, err := http.Get(wowiGameVersionsUrl)
	if err != nil {
		w.logGroup.Error("Could not fetch game versions: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.logGroup.Error("Could not fetch game versions: %v", err)
		return fmt.Errorf("could not fetch game versions: %v", err)
	}

	var versionResp []wowiGameVersionsEntry
	err = json.NewDecoder(resp.Body).Decode(&versionResp)
	if err != nil {
		w.logGroup.Error("Could not fetch game versions: %v", err)
		return err
	}

	versionIdList := make([]string, len(versionResp))
	for i, version := range versionResp {
		versionIdList[i] = version.Id
	}

	for _, gameVersion := range gameVersions {
		if !stringInSlice(gameVersion, versionIdList) {
			w.logGroup.Warn("Game version %s is not supported by WoW Interface, skipping", gameVersion)
		} else {
			w.compatible = append(w.compatible, gameVersion)
		}
	}

	return nil
}

func (w *wowiUpload) upload() error {
	w.logGroup.Info("Uploading to WoW Interface")

	file, err := os.Open(w.zipFile)
	if err != nil {
		return fmt.Errorf("could not open zip file: %v", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err = writer.WriteField("id", w.projectId); err != nil {
		return fmt.Errorf("could not write id: %v", err)
	}

	if err = writer.WriteField("version", w.version); err != nil {
		return fmt.Errorf("could not write version: %v", err)
	}

	if err = writer.WriteField("compatible", strings.Join(w.compatible, ",")); err != nil {
		return fmt.Errorf("could not write compatible: %v", err)
	}

	if !w.archiveOld {
		if err = writer.WriteField("archive", "No"); err != nil {
			return fmt.Errorf("could not write archive: %v", err)
		}
	}

	if w.changelog != nil {
		if w.changelog.PreExistingFilePath != "" {
			changelogContents, err := os.ReadFile(w.changelog.PreExistingFilePath)
			if err != nil {
				return fmt.Errorf("could not read changelog: %v", err)
			}

			if err = writer.WriteField("changelog", string(changelogContents)); err != nil {
				return fmt.Errorf("could not write changelog: %v", err)
			}
		}
	}

	// Add the file part
	part, err := writer.CreateFormFile("updatefile", filepath.Base(w.zipFile))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err = io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close the writer to finalize the multipart form
	if err = writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create the POST request
	req, err := http.NewRequest("POST", wowiUploadUrl, &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	// Set the proper headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-token", w.token)

	// Prepare the HTTP client and exponential backoff parameters
	client := &http.Client{}
	maxAttempts := 5
	delay := 2 * time.Second

	var resp *http.Response
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err = client.Do(req)
		if err == nil && (resp.StatusCode >= 200 && resp.StatusCode < 300) {
			w.logGroup.Info("Successfully uploaded to WoW Interface!")
			return nil
		}

		// Log the error details for debugging
		if err != nil {
			w.logGroup.Warn("upload error: %v", err)
		} else {
			w.logGroup.Warn("unexpected status code: %d", resp.StatusCode)
			jsonBody := map[string]interface{}{}
			err = json.NewDecoder(resp.Body).Decode(&jsonBody)
			if err != nil {
				w.logGroup.Warn("failed to decode response body: %v", err)
			} else {
				w.logGroup.Warn("response body: %v", jsonBody)
			}
			if resp.StatusCode == http.StatusUnprocessableEntity || resp.StatusCode == http.StatusBadRequest {
				return fmt.Errorf("upload failed: %s", resp.Status)
			}
		}

		// If not the last attempt, wait for the delay before retrying
		if attempt < maxAttempts {
			w.logGroup.Warn("Retrying: Attempt %d/%d in %s...", attempt+1, maxAttempts, delay)
			time.Sleep(delay)
			delay *= 2 // Exponential backoff: double the delay each time
		}
	}

	// If we exhausted our attempts, report the failure.
	if err != nil {
		return fmt.Errorf("upload failed after %d attempts: %v", maxAttempts, err)
	}
	if resp != nil && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
		return fmt.Errorf("upload failed with status code %d", resp.StatusCode)
	}

	return nil
}

type UploadWowiArgs struct {
	TocFiles       []*toc.Toc
	ProjectVersion string
	ZipPath        string
	FileLabel      string
	Changelog      *changelog.Changelog
	ReleaseType    string
	WowiArchiveOld bool
}

func UploadToWowi(args UploadWowiArgs) error {
	logGroup := logger.NewLogGroup("🛜  Uploading to WoW Interface")
	defer logGroup.Flush(true)

	tocFiles := args.TocFiles

	wowiId, err := getWowiId(tocFiles)
	if err != nil {
		if err == ErrNoWowiId || err == ErrNoWowiUpload {
			logGroup.Verbose("Skipping WoW Interface upload")
			return nil
		}
		return err
	}

	wowiUpload := wowiUpload{
		projectId:   wowiId,
		zipFile:     args.ZipPath,
		displayName: args.FileLabel,
		changelog:   args.Changelog,
		version:     args.ProjectVersion,
		archiveOld:  args.WowiArchiveOld,
		logGroup:    logGroup,
	}

	if err := wowiUpload.lookupWowiToken(); err != nil {
		logGroup.Info("Skipping WoW Interface upload: %s", err)
		return nil
	}

	gameVersions := toc.GetGameVersions()

	if err := wowiUpload.validateGameVersions(gameVersions); err != nil {
		logGroup.Error("Could not validate game versions: %v", err)
		return err
	}

	if err := wowiUpload.upload(); err != nil {
		return err
	}

	return nil
}
