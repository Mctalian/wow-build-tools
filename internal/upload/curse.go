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

type gameVersionTypeId int

const (
	classic gameVersionTypeId = 67408
	bcc     gameVersionTypeId = 73246
	wrath   gameVersionTypeId = 73713
	cata    gameVersionTypeId = 77522
	retail  gameVersionTypeId = 517
)

type ReleaseType string

const (
	AlphaRelease ReleaseType = "alpha"
	BetaRelease  ReleaseType = "beta"
	Release      ReleaseType = "release"
)

type RelationshipType string

const (
	Incompatible       RelationshipType = "incompatible"
	EmbeddedLibrary    RelationshipType = "embeddedLibrary"
	OptionalDependency RelationshipType = "optionalDependency"
	RequiredDependency RelationshipType = "requiredDependency"
	Tool               RelationshipType = "tool"
)

type ProjectRelationship struct {
	Slug string           `json:"slug"`
	Type RelationshipType `json:"type"`
}

type Relations struct {
	Projects []ProjectRelationship `json:"projects"`
}

type cursePayload struct {
	Changelog     string               `json:"changelog"`
	ChangelogType changelog.MarkupType `json:"changelogType"`
	DisplayName   string               `json:"displayName"`
	ReleaseType   ReleaseType          `json:"releaseType"`
	GameVersions  []int                `json:"gameVersions"`
	Relations     Relations            `json:"relations"`
}

type curseUpload struct {
	projectId    string
	token        string
	metadataPart string
	uploadUrl    string
	zipFile      string
	displayName  string
	gameVersions []int
	changelog    *changelog.Changelog
}

type gameVersion struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	GameVersionTypeID int    `json:"gameVersionTypeId"`
}

type gameVersionResponse []gameVersion

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
	projects := make([]ProjectRelationship, 0)
	for _, embed := range pkgMeta.EmbeddedLibraries {
		projects = append(projects, ProjectRelationship{
			Slug: embed,
			Type: EmbeddedLibrary,
		})
	}

	for _, tool := range pkgMeta.ToolsUsed {
		projects = append(projects, ProjectRelationship{
			Slug: tool,
			Type: Tool,
		})
	}

	for _, reqDep := range pkgMeta.RequiredDependencies {
		projects = append(projects, ProjectRelationship{
			Slug: reqDep,
			Type: RequiredDependency,
		})
	}

	for _, optDep := range pkgMeta.OptionalDependencies {
		projects = append(projects, ProjectRelationship{
			Slug: optDep,
			Type: OptionalDependency,
		})
	}

	// TODO: Currently not possible to specify incompatible relationships

	changelogPath, err := c.changelog.GetChangelog()
	if err != nil {
		return
	}

	changelogContents, err := os.ReadFile(changelogPath)
	if err != nil {
		return
	}

	payload := cursePayload{
		Changelog:     string(changelogContents),
		ChangelogType: c.changelog.MarkupType,
		DisplayName:   c.displayName,
		ReleaseType:   AlphaRelease,
		GameVersions:  c.gameVersions,
	}

	if len(projects) > 0 {
		payload.Relations = Relations{
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
	gameVersionsUrl := fmt.Sprintf("%sgame/wow/versions", curseApiUrl)

	req, err := http.NewRequest("GET", gameVersionsUrl, nil)
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

	var versions gameVersionResponse
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

	// // Create the POST request
	// req, err := http.NewRequest("POST", c.uploadUrl, &body)
	// if err != nil {
	// 	return fmt.Errorf("failed to create request: %w", err)
	// }
	// // Set the proper headers
	// req.Header.Set("Content-Type", writer.FormDataContentType())
	// req.Header.Set("Accept", "application/json")
	// req.Header.Set("x-api-token", c.token) // Adjust this header key if needed

	// // Prepare the HTTP client and exponential backoff parameters
	// client := &http.Client{}
	// maxAttempts := 5
	// delay := 2 * time.Second

	// var resp *http.Response
	// for attempt := 1; attempt <= maxAttempts; attempt++ {
	// 	resp, err = client.Do(req)
	// 	if err == nil && (resp.StatusCode >= 200 && resp.StatusCode < 300) {
	// 		logger.Info("Upload successful!")
	// 		return
	// 	}

	// 	// Log the error details for debugging
	// 	if err != nil {
	// 		logger.Error("upload error: %s", err)
	// 	} else {
	// 		jsonBody := map[string]interface{}{}
	// 		err = json.NewDecoder(resp.Body).Decode(&jsonBody)
	// 		if err != nil {
	// 			logger.Error("failed to decode response body: %s", err)
	// 		}

	// 		logger.Warn("%v", jsonBody)
	// 		logger.Error("unexpected status code: %d", resp.StatusCode)
	// 	}

	// 	// If not the last attempt, wait for the delay before retrying
	// 	if attempt < maxAttempts {
	// 		logger.Info("Retrying in %s...", delay)
	// 		time.Sleep(delay)
	// 		delay *= 2 // Exponential backoff: double the delay each time
	// 	}
	// }

	// // If we exhausted our attempts, report the failure.
	// if err != nil {
	// 	return fmt.Errorf("upload failed after %d attempts: %w", maxAttempts, err)
	// }
	// if resp != nil && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
	// 	return fmt.Errorf("upload failed with status code %d", resp.StatusCode)
	// }

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
	TocFiles  []*toc.Toc
	ZipPath   string
	FileLabel string
	PkgMeta   *pkg.PkgMeta
	Changelog *changelog.Changelog
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

	curseUpload := curseUpload{
		projectId:   curseId,
		uploadUrl:   fmt.Sprintf("%sprojects/%s/upload-file", curseApiUrl, curseId),
		zipFile:     args.ZipPath,
		displayName: args.FileLabel,
		changelog:   args.Changelog,
	}

	if err := curseUpload.lookupCurseToken(); err != nil {
		logger.Info("Skipping CurseForge upload: %s", err)
		return nil
	}

	var gameVersionsSet = make(map[string]bool)
	for _, tocFile := range tocFiles {
		gameVersions := tocFile.GetGameVersions()
		for _, version := range gameVersions {
			gameVersionsSet[version] = true
		}
	}

	var gameVersions []string
	for version := range gameVersionsSet {
		gameVersions = append(gameVersions, version)
	}

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
