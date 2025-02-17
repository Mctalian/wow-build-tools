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
	"github.com/McTalian/wow-build-tools/internal/pkg"
	"github.com/McTalian/wow-build-tools/internal/toc"
)

var ErrNoCurseId = fmt.Errorf("no Curse ID provided")
var ErrMultipleCurseIds = fmt.Errorf("multiple Curse IDs found")
var ErrNoCurseUpload = fmt.Errorf("CurseForge upload is disabled")
var ErrNoCurseApiKey = fmt.Errorf("CF_API_KEY not set")

var curseApiUrl = "https://wow.curseforge.com/api/"
var curseGameVersionsUrl = fmt.Sprintf("%sgame/wow/versions", curseApiUrl)

type curseGameVersionTypeId int

const (
	classic curseGameVersionTypeId = 67408
	bcc     curseGameVersionTypeId = 73246
	wrath   curseGameVersionTypeId = 73713
	cata    curseGameVersionTypeId = 77522
	retail  curseGameVersionTypeId = 517
)

type curseReleaseType string

const (
	AlphaRelease curseReleaseType = "alpha"
	BetaRelease  curseReleaseType = "beta"
	Release      curseReleaseType = "release"
)

type curseRelationshipType string

const (
	Incompatible       curseRelationshipType = "incompatible"
	EmbeddedLibrary    curseRelationshipType = "embeddedLibrary"
	OptionalDependency curseRelationshipType = "optionalDependency"
	RequiredDependency curseRelationshipType = "requiredDependency"
	Tool               curseRelationshipType = "tool"
)

type curseProjectRelationship struct {
	Slug string                `json:"slug"`
	Type curseRelationshipType `json:"type"`
}

type curseRelations struct {
	Projects []curseProjectRelationship `json:"projects"`
}

type cursePayload struct {
	Changelog     string               `json:"changelog"`
	ChangelogType changelog.MarkupType `json:"changelogType"`
	DisplayName   string               `json:"displayName"`
	ReleaseType   curseReleaseType     `json:"releaseType"`
	GameVersions  []int                `json:"gameVersions"`
	Relations     curseRelations       `json:"relations"`
}

type curseUpload struct {
	projectId    string
	token        string
	metadataPart string
	uploadUrl    string
	zipFile      string
	displayName  string
	gameVersions []int
	releaseType  curseReleaseType
	changelog    *changelog.Changelog
}

type curseGameVersion struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	GameVersionTypeID int    `json:"gameVersionTypeId"`
}

type curseGameVersionResponse []curseGameVersion

func locateCurseId(tocFiles []*toc.Toc) (curseId string, err error) {
	var foundCurseId string
	for _, tocFile := range tocFiles {
		if tocFile.CurseId != "" {
			if foundCurseId != "" && foundCurseId != tocFile.CurseId {
				err = ErrMultipleCurseIds
				return
			}
			foundCurseId = tocFile.CurseId
		}
	}

	if foundCurseId == "" {
		err = ErrNoCurseId
		return
	}

	curseId = foundCurseId
	return
}

func (c *curseUpload) lookupCurseToken() (err error) {
	token, found := os.LookupEnv("CF_API_KEY")
	if !found {
		err = ErrNoCurseApiKey
		return
	}

	c.token = token

	return
}

func (c *curseUpload) preparePayload(pkgMeta *pkg.PkgMeta) (err error) {
	projects := make([]curseProjectRelationship, 0)
	for _, embed := range pkgMeta.EmbeddedLibraries {
		projects = append(projects, curseProjectRelationship{
			Slug: embed,
			Type: EmbeddedLibrary,
		})
	}

	for _, tool := range pkgMeta.ToolsUsed {
		projects = append(projects, curseProjectRelationship{
			Slug: tool,
			Type: Tool,
		})
	}

	for _, reqDep := range pkgMeta.RequiredDependencies {
		projects = append(projects, curseProjectRelationship{
			Slug: reqDep,
			Type: RequiredDependency,
		})
	}

	for _, optDep := range pkgMeta.OptionalDependencies {
		projects = append(projects, curseProjectRelationship{
			Slug: optDep,
			Type: OptionalDependency,
		})
	}

	// TODO: Currently not possible to specify incompatible relationships

	changelogPath := c.changelog.PreExistingFilePath

	changelogContents, err := os.ReadFile(changelogPath)
	if err != nil {
		return
	}

	payload := cursePayload{
		Changelog:     string(changelogContents),
		ChangelogType: c.changelog.MarkupType,
		DisplayName:   c.displayName,
		ReleaseType:   c.releaseType,
		GameVersions:  c.gameVersions,
	}

	if len(projects) > 0 {
		payload.Relations = curseRelations{
			Projects: projects,
		}
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return
	}

	c.metadataPart = string(jsonPayload)

	return
}

func (c *curseUpload) validateGameVersions(gameVersions []string) (err error) {
	req, err := http.NewRequest("GET", curseGameVersionsUrl, nil)
	if err != nil {
		logger.Error("Could not fetch game versions: %v", err)
		return
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-token", c.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error("Could not fetch game versions: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("Could not fetch game versions: %v", resp.Status)
	}

	var versions curseGameVersionResponse
	err = json.NewDecoder(resp.Body).Decode(&versions)
	if err != nil {
		logger.Error("Could not decode game versions: %v", err)
		return
	}

	// logger.Info("Game versions: %v", versions)

	var missingVersions = make(map[string]bool)
	for _, version := range gameVersions {
		missingVersions[version] = true
	}

	for _, version := range versions {
		if slices.Contains(gameVersions, version.Name) {
			c.gameVersions = append(c.gameVersions, version.ID)
			missingVersions[version.Name] = false
		}
	}

	if len(c.gameVersions) == 0 {
		logger.Error("Could not find any game versions from interface version(s) in toc file(s)")
		return fmt.Errorf("no game versions found")
	}

	if len(c.gameVersions) != len(gameVersions) {
		var missingVersionsList []string
		for version, missing := range missingVersions {
			if missing {
				missingVersionsList = append(missingVersionsList, version)
			}
		}
		logger.Warn("Could not find all game versions from interface versions. Missing: %v", missingVersionsList)
	}

	return
}

func (c *curseUpload) upload() (err error) {
	logger.Info("Uploading to CurseForge")

	// Open the zip file to upload
	file, err := os.Open(c.zipFile)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer file.Close()

	// Create a buffer and multipart writer for the request body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	logger.Info("metadata: %s", c.metadataPart)

	// Add metadata part as a form field
	if err = writer.WriteField("metadata", c.metadataPart); err != nil {
		return fmt.Errorf("failed to write metadata field: %w", err)
	}

	// Add the file part
	part, err := writer.CreateFormFile("file", filepath.Base(c.zipFile))
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
	req, err := http.NewRequest("POST", c.uploadUrl, &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	// Set the proper headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-token", c.token) // Adjust this header key if needed

	// Prepare the HTTP client and exponential backoff parameters
	client := &http.Client{}
	maxAttempts := 5
	delay := 2 * time.Second

	var resp *http.Response
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err = client.Do(req)
		if err == nil && (resp.StatusCode >= 200 && resp.StatusCode < 300) {
			logger.Info("Successfully uploaded to CurseForge!")
			return
		}

		// Log the error details for debugging
		if err != nil {
			logger.Warn("upload error: %v", err)
		} else {
			logger.Warn("unexpected status code: %d", resp.StatusCode)
			jsonBody := map[string]interface{}{}
			err = json.NewDecoder(resp.Body).Decode(&jsonBody)
			if err != nil {
				logger.Warn("failed to decode response body: %v", err)
			} else {
				logger.Warn("response body: %v", jsonBody)
			}
		}

		// If not the last attempt, wait for the delay before retrying
		if attempt < maxAttempts {
			logger.Warn("Retrying: Attempt %d/%d in %s...", attempt+1, maxAttempts, delay)
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

func getCurseId(tocFiles []*toc.Toc) (curseId string, err error) {
	if f.SkipUpload || f.OnlyLocalization {
		err = ErrNoCurseUpload
		return
	}

	if f.CurseId != "" {
		if f.CurseId == "0" {
			curseId = ""
		} else {
			curseId = f.CurseId
		}
	} else {
		curseId, err = locateCurseId(tocFiles)
		if err == ErrMultipleCurseIds {
			return
		}
	}
	if curseId == "" {
		err = ErrNoCurseId
		return
	}

	return
}

type UploadCurseArgs struct {
	TocFiles    []*toc.Toc
	ZipPath     string
	FileLabel   string
	PkgMeta     *pkg.PkgMeta
	Changelog   *changelog.Changelog
	ReleaseType string
}

func UploadToCurse(args UploadCurseArgs) error {
	tocFiles := args.TocFiles
	pkgMeta := args.PkgMeta

	curseId, err := getCurseId(tocFiles)
	if err != nil {
		if err == ErrNoCurseId || err == ErrNoCurseUpload {
			logger.Verbose("Skipping CurseForge upload")
			return nil
		}
		return err
	}

	var releaseType curseReleaseType
	switch args.ReleaseType {
	case "alpha":
		releaseType = AlphaRelease
	case "beta":
		releaseType = BetaRelease
	case "release":
		releaseType = Release
	default:
		logger.Warn("Invalid release type: %s, defaulting to alpha", args.ReleaseType)
		releaseType = AlphaRelease
	}

	curseUpload := curseUpload{
		projectId:   curseId,
		uploadUrl:   fmt.Sprintf("%sprojects/%s/upload-file", curseApiUrl, curseId),
		zipFile:     args.ZipPath,
		displayName: args.FileLabel,
		changelog:   args.Changelog,
		releaseType: releaseType,
	}

	if err := curseUpload.lookupCurseToken(); err != nil {
		logger.Info("Skipping CurseForge upload: %s", err)
		return nil
	}

	gameVersions := toc.GetGameVersions()

	if err := curseUpload.validateGameVersions(gameVersions); err != nil {
		logger.Error("Could not validate game versions: %v", err)
		return err
	}

	if err := curseUpload.preparePayload(pkgMeta); err != nil {
		return err
	}

	if err := curseUpload.upload(); err != nil {
		return err
	}

	return nil
}
