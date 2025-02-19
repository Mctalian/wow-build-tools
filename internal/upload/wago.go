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
	"slices"
	"time"

	"github.com/McTalian/wow-build-tools/internal/changelog"
	f "github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/toc"
)

var ErrNoWagoId = fmt.Errorf("no Wago ID provided")
var ErrMultipleWagoIds = fmt.Errorf("multiple Wago IDs found")
var ErrNoWagoUpload = fmt.Errorf("Wago upload is disabled")
var ErrNoWagoApiKey = fmt.Errorf("WAGO_API_TOKEN not set")

var wagoApiUrl = "https://addons.wago.io/api/"
var wagoGameVersionsUrl = fmt.Sprintf("%sdata/game", wagoApiUrl)

type wagoPayload struct {
	Label            string              `json:"label"`
	Stability        string              `json:"stability"`
	Changelog        string              `json:"changelog"`
	SupportedPatches map[string][]string `json:"-"`
}

func (wp *wagoPayload) MarshalJSON() ([]byte, error) {
	type Alias wagoPayload
	alias := &struct {
		*Alias
	}{
		Alias: (*Alias)(wp),
	}

	data, err := json.Marshal(alias)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	for k, v := range wp.SupportedPatches {
		result[fmt.Sprintf("supported_%s_patches", k)] = v
	}

	return json.Marshal(result)
}

type wagoGameVersionResponse struct {
	StabilityValues []string            `json:"stability_values"`
	Patches         map[string][]string `json:"patches"`
	TocSuffixes     map[string][]string `json:"toc_suffixes"`
}

var wagoStabilityValues = []string{"alpha", "beta", "stable"}

type wagoUpload struct {
	token          string
	supportMap     map[string][]string
	projectId      string
	uploadUrl      string
	zipFile        string
	displayName    string
	changelog      *changelog.Changelog
	stabilityValue string
	metadataPart   string
	logGroup       *logger.LogGroup
}

func locateWagoId(tocFiles []*toc.Toc) (wagoId string, err error) {
	var foundWagoId string
	for _, tocFile := range tocFiles {
		if tocFile.WagoId != "" {
			if foundWagoId != "" && foundWagoId != tocFile.WagoId {
				err = ErrMultipleWagoIds
				return
			}
			foundWagoId = tocFile.WagoId
		}
	}

	if foundWagoId == "" {
		err = ErrNoWagoId
		return
	}

	wagoId = foundWagoId
	return
}

func getWagoId(tocFiles []*toc.Toc) (wagoId string, err error) {
	if f.SkipUpload {
		err = ErrNoWagoUpload
		return
	}

	if f.WagoId != "" {
		if f.WagoId == "0" {
			wagoId = ""
		} else {
			wagoId = f.WagoId
		}
	} else {
		wagoId, err = locateWagoId(tocFiles)
		if err == ErrMultipleWagoIds {
			return
		}
	}
	if wagoId == "" {
		err = ErrNoWagoId
		return
	}

	return
}

func (w *wagoUpload) lookupWagoToken() (err error) {
	token, found := os.LookupEnv("WAGO_API_TOKEN")
	if !found {
		err = ErrNoWagoApiKey
		return
	}

	w.token = token

	return
}

func (w *wagoUpload) validateGameVersions(gameVersions []string) (err error) {
	req, err := http.NewRequest("GET", wagoGameVersionsUrl, nil)
	if err != nil {
		w.logGroup.Error("Could not fetch game versions: %v", err)
		return
	}

	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.logGroup.Error("Could not fetch game versions: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("could not fetch game versions: %v", resp.Status)
		return
	}

	var versionResp wagoGameVersionResponse
	err = json.NewDecoder(resp.Body).Decode(&versionResp)
	if err != nil {
		w.logGroup.Error("Could not decode game versions: %v", err)
		return
	}

	for _, stability := range versionResp.StabilityValues {
		if !slices.Contains(wagoStabilityValues, stability) {
			w.logGroup.Warn("Unknown stability value: %s", stability)
		}
	}

	// w.logGroup.Info("Game versions: %v", versions)

	var missingVersions = make(map[string]bool)
	for _, version := range gameVersions {
		missingVersions[version] = true
	}

	flavorVersionMap := toc.GetGameFlavorVersionsMap()

	for flavor, versions := range flavorVersionMap {
		var wago_type string
		switch flavor {
		case toc.TbcClassic:
			wago_type = "bc"
		case toc.WotlkClassic:
			wago_type = "wotlk"
		default:
			wago_type = flavor.ToString()
		}
		for _, version := range versions {
			if slices.Contains(versionResp.Patches[wago_type], version) {
				w.supportMap[wago_type] = append(w.supportMap[wago_type], version)
				missingVersions[version] = false
			}
		}
	}

	if len(w.supportMap) == 0 {
		w.logGroup.Error("Could not find any game versions from interface version(s) in toc file(s)")
		return fmt.Errorf("no game versions found")
	}

	var missingVersionsList []string
	for version, missing := range missingVersions {
		if missing {
			missingVersionsList = append(missingVersionsList, version)
		}
	}

	if len(missingVersionsList) > 0 {
		w.logGroup.Warn("Could not find all game versions from interface versions. Missing: %v", missingVersionsList)
	}

	return
}

func (w *wagoUpload) preparePayload() error {
	changelogPath := w.changelog.PreExistingFilePath

	changelogContents, err := os.ReadFile(changelogPath)
	if err != nil {
		return err
	}

	payload := wagoPayload{
		Label:            w.displayName,
		Stability:        w.stabilityValue,
		Changelog:        string(changelogContents),
		SupportedPatches: w.supportMap,
	}

	jsonPayload, err := json.Marshal(&payload)
	if err != nil {
		return err
	}

	w.metadataPart = string(jsonPayload)

	return nil
}

func (w *wagoUpload) upload() error {
	w.logGroup.Info("Uploading to Wago.io")

	file, err := os.Open(w.zipFile)
	if err != nil {
		return fmt.Errorf("could not open zip file: %v", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	w.logGroup.Verbose("metadata: %s", w.metadataPart)

	if err = writer.WriteField("metadata", w.metadataPart); err != nil {
		return fmt.Errorf("could not write metadata: %v", err)
	}

	part, err := writer.CreateFormFile("file", filepath.Base(w.zipFile))
	if err != nil {
		return fmt.Errorf("could not create form file: %v", err)
	}
	if _, err = io.Copy(part, file); err != nil {
		return fmt.Errorf("could not copy file: %v", err)
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("could not close writer: %v", err)
	}

	req, err := http.NewRequest("POST", w.uploadUrl, &body)
	if err != nil {
		return fmt.Errorf("could not create request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", w.token))

	client := &http.Client{}
	maxAttempts := 5
	delay := 2 * time.Second

	var resp *http.Response
	for attempt := 0; attempt <= maxAttempts; attempt++ {
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			w.logGroup.Info("Successfully uploaded to Wago.io!")
			return nil
		}
		if err != nil {
			w.logGroup.Warn("Failed to upload to Wago.io: %v", err)
		} else {
			w.logGroup.Warn("Failed to upload to Wago.io: %s", resp.Status)
			jsonBody := make(map[string]interface{})
			err = json.NewDecoder(resp.Body).Decode(&jsonBody)
			if err != nil {
				w.logGroup.Warn("failed to decode response body: %v", err)
			} else {
				w.logGroup.Warn("Response: %v", jsonBody)
			}
			if resp.StatusCode == http.StatusUnprocessableEntity {
				return fmt.Errorf("upload failed: %s", resp.Status)
			}
		}

		if attempt < maxAttempts {
			w.logGroup.Warn("Retrying: Attempt %d/%d in %s...", attempt+1, maxAttempts, delay)
			time.Sleep(delay)
			delay *= 2
		}
	}

	if err != nil {
		return fmt.Errorf("upload failed after %d attempts: %v", maxAttempts, err)
	}
	if resp != nil && (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices) {
		return fmt.Errorf("upload failed with status %s", resp.Status)
	}

	return nil
}

type UploadWagoArgs struct {
	TocFiles    []*toc.Toc
	ZipPath     string
	FileLabel   string
	Changelog   *changelog.Changelog
	ReleaseType string
}

func UploadToWago(args UploadWagoArgs) error {
	logGroup := logger.NewLogGroup("🪢  Uploading to Wago")
	defer logGroup.Flush(true)

	tocFiles := args.TocFiles

	wagoId, err := getWagoId(tocFiles)
	if err != nil {
		if err == ErrNoWagoId || err == ErrNoWagoUpload {
			logGroup.Verbose("Skipping Wago upload")
			return nil
		}
		return err
	}

	var stabilityValue string
	switch args.ReleaseType {
	case "alpha":
		stabilityValue = "alpha"
	case "beta":
		stabilityValue = "beta"
	case "release":
		stabilityValue = "stable"
	default:
		logGroup.Warn("Unknown release type: %s", args.ReleaseType)
		stabilityValue = "alpha"
	}

	wagoUpload := wagoUpload{
		projectId:      wagoId,
		uploadUrl:      fmt.Sprintf("%sprojects/%s/version", wagoApiUrl, wagoId),
		zipFile:        args.ZipPath,
		displayName:    args.FileLabel,
		changelog:      args.Changelog,
		stabilityValue: stabilityValue,
		supportMap:     make(map[string][]string),
		logGroup:       logGroup,
	}

	if err := wagoUpload.lookupWagoToken(); err != nil {
		logGroup.Info("Skipping Wago upload: %s", err)
		return nil
	}

	gameVersions := toc.GetGameVersions()

	if err := wagoUpload.validateGameVersions(gameVersions); err != nil {
		logGroup.Error("Could not validate game versions: %v", err)
		return err
	}

	if err := wagoUpload.preparePayload(); err != nil {
		return err
	}

	if err := wagoUpload.upload(); err != nil {
		return err
	}

	return nil
}
