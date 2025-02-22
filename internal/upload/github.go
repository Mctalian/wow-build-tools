package upload

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/McTalian/wow-build-tools/internal/changelog"
	"github.com/McTalian/wow-build-tools/internal/github"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/repo"
	"github.com/McTalian/wow-build-tools/internal/toc"
)

func shouldSkip(repo repo.VcsRepo, logGroup *logger.LogGroup) bool {
	if !repo.IsGitHubHosted() {
		logGroup.Verbose("Repository is not hosted on GitHub, skipping")
		return true
	}

	if repo.GetCurrentTag() == "" {
		logGroup.Verbose("No current tag found, skipping")
		return true
	}

	if repo.GetGitHubSlug() == "" {
		logGroup.Verbose("No GitHub slug found, skipping")
		return true
	}

	if !github.IsTokenSet() {
		logGroup.Verbose("GITHUB_OAUTH not set, skipping")
		return true
	}

	return false
}

func GetOrCreateRelease(repo repo.VcsRepo, prerelease bool, changelogContents string, logGroup *logger.LogGroup) (release *github.GitHubRelease, err error) {
	release, err = github.GetRelease(repo.GetGitHubSlug(), repo.GetCurrentTag())
	if err != nil && err != github.ErrReleaseNotFound {
		logGroup.Error("Could not get the release: %v", err)
		return
	}
	if err == github.ErrReleaseNotFound {
		payload := &github.GitHubReleasePayload{
			TagName:    repo.GetCurrentTag(),
			Name:       repo.GetCurrentTag(),
			Prerelease: prerelease,
			Body:       string(changelogContents),
			Draft:      false,
		}
		release, err = github.CreateRelease(repo.GetGitHubSlug(), *payload)
		if err != nil {
			logGroup.Error("Could not create the release: %v", err)
			return
		}
	} else {
		payload := github.GitHubReleasePayload{
			TagName:    repo.GetCurrentTag(),
			Name:       repo.GetCurrentTag(),
			Prerelease: prerelease,
			Body:       string(changelogContents),
			Draft:      false,
		}
		err = release.UpdateRelease(payload)
		if err != nil {
			logGroup.Error("Could not update the release: %v", err)
			return
		}
	}

	return release, nil
}

type UploadGitHubArgs struct {
	ProjectName    string
	ProjectVersion string
	Repo           repo.VcsRepo
	ZipPaths       []string
	Changelog      *changelog.Changelog
	ReleaseType    string
}

func UploadToGitHub(args UploadGitHubArgs) error {
	logGroup := logger.NewLogGroup("🐱 Uploading to GitHub")
	defer logGroup.Flush(true)
	var err error

	repo := args.Repo

	if shouldSkip(repo, logGroup) {
		return nil
	}

	prerelease := true
	switch args.ReleaseType {
	case "release":
		prerelease = false
	case "alpha", "beta":
		prerelease = true
	default:
		logGroup.Warn("Invalid release type: %s, defaulting to prerelease", args.ReleaseType)
		prerelease = true
	}

	changelogPath := args.Changelog.PreExistingFilePath

	changelogContents, err := os.ReadFile(changelogPath)
	if err != nil {
		return err
	}

	release, err := GetOrCreateRelease(repo, prerelease, string(changelogContents), logGroup)
	if err != nil {
		logGroup.Error("Could not get or create the release: %v", err)
		return err
	}

	gameVersions := toc.GetGameFlavorInterfacesMap()

	releaseFileContents, err := github.GetReleaseMetadataContents(
		args.ProjectName,
		args.ProjectVersion,
		gameVersions,
		args.ZipPaths...,
	)

	tmpDir := os.TempDir()
	releaseFile, err := os.CreateTemp(tmpDir, "release-metadata-*.json")
	if err != nil {
		logGroup.Error("Could not create the release metadata file: %v", err)
		return err
	}
	defer os.Remove(releaseFile.Name())
	defer releaseFile.Close()

	_, err = releaseFile.WriteString(releaseFileContents)
	if err != nil {
		logGroup.Error("Could not write the release metadata to the file: %v", err)
		return err
	}
	if err = releaseFile.Sync(); err != nil {
		logGroup.Error("Could not sync the release metadata file: %v", err)
		return err
	}

	zipNames := make([]string, len(args.ZipPaths))
	for i, zipPath := range args.ZipPaths {
		zipNames[i] = filepath.Base(zipPath)
	}

	type assetToUpload struct {
		FileName string
		FilePath string
	}

	assetsToUpload := []assetToUpload{
		{FileName: "release.json", FilePath: releaseFile.Name()},
	}

	for i, zipPath := range args.ZipPaths {
		assetsToUpload = append(assetsToUpload, assetToUpload{
			FileName: zipNames[i],
			FilePath: zipPath,
		})
	}

	var assetWg sync.WaitGroup
	assetErrChan := make(chan error, len(assetsToUpload))

	for _, asset := range assetsToUpload {
		assetWg.Add(1)
		go func(asset assetToUpload) {
			defer assetWg.Done()
			err := release.UploadAsset(asset.FileName, asset.FilePath)
			if err != nil {
				assetErrChan <- fmt.Errorf("could not upload asset %s: %w", asset.FileName, err)
			}
		}(asset)
	}

	assetWg.Wait()
	close(assetErrChan)

	for err := range assetErrChan {
		logGroup.Error("Uploading asset failed: %v", err)
		return err
	}

	return nil
}
