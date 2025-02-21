package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/McTalian/wow-build-tools/internal/cachedir"
	"github.com/McTalian/wow-build-tools/internal/github"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/pkg"
	"github.com/McTalian/wow-build-tools/internal/repo"
)

type MarkupType string

const (
	MarkdownMT MarkupType = "markdown"
	HTMLMT     MarkupType = "html"
	TextMT     MarkupType = "text"
)

type Changelog struct {
	repo                repo.VcsRepo
	title               string
	pkgDir              string
	topDir              string
	PreExistingFilePath string
	MarkupType          MarkupType
	generateChangelog   bool
	requiresCleanup     bool
}

func (c *Changelog) Cleanup() {
	if !c.requiresCleanup {
		return
	}

	if c.PreExistingFilePath == "" {
		return
	}

	if _, err := os.Stat(c.PreExistingFilePath); err == nil {
		if err := os.Remove(c.PreExistingFilePath); err != nil {
			logger.Error("Could not remove the temporary changelog file: %v", err)
		}
	} else if os.IsNotExist(err) {
		return
	}

	return
}

var ErrManualChangelogNotFound = fmt.Errorf("Manual changelog file not found")
var ErrInvalidMarkupType = fmt.Errorf("Invalid markup type")

func (c *Changelog) verifyManualChangelog() error {
	if c.PreExistingFilePath == "" {
		return ErrManualChangelogNotFound
	}
	pkgDir := c.pkgDir

	relativeFilePath := c.PreExistingFilePath
	topDirChangelogPath := filepath.Join(c.topDir, relativeFilePath)
	pkgDirChangelogPath := filepath.Join(pkgDir, relativeFilePath)

	if _, err := os.Stat(topDirChangelogPath); os.IsNotExist(err) {
		// If it wasn't found in the top directory, it won't be found in the package directory
		return ErrManualChangelogNotFound
	}

	// Check for the file in the package directory in case it had token replacements
	if _, err := os.Stat(pkgDirChangelogPath); err == nil {
		c.PreExistingFilePath = pkgDirChangelogPath
	} else if os.IsNotExist(err) {
		c.PreExistingFilePath = topDirChangelogPath
	} else {
		// Something unexpected happened
		return err
	}

	var err error
	switch c.MarkupType {
	case MarkdownMT, HTMLMT, TextMT:
		err = nil
	default:
		err = ErrInvalidMarkupType
	}

	return err
}

func (c *Changelog) GetChangelog() error {
	if !c.generateChangelog {
		_, err := os.Stat(c.PreExistingFilePath)
		if err == nil {
			_, err := os.ReadFile(c.PreExistingFilePath)
			if err != nil {
				logger.Error("Could not read the manual changelog file (even though it exists): %v", err)
				return err
			}
			if strings.Contains(c.PreExistingFilePath, c.pkgDir) {
				return nil
			} else {
				destPath := filepath.Join(c.pkgDir, strings.TrimPrefix(c.PreExistingFilePath, c.topDir))
				err = pkg.CopySingleFile(c.PreExistingFilePath, destPath, nil)
				if err != nil {
					logger.Error("Attempting generation - could not copy the manual changelog file to the package directory: %v", err)
				} else {
					c.PreExistingFilePath = destPath
					return nil
				}
			}
			return nil
		} else {
			logger.Warn("%v: will attempt to generate from commits instead", err)
		}
	}

	// TODO: Generate the changelog
	c.MarkupType = MarkdownMT
	c.PreExistingFilePath = filepath.Join(c.pkgDir, "CHANGELOG.md")

	// Write the changelog to the package directory
	f, err := os.OpenFile(c.PreExistingFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("Could not create the changelog file: %v", err)
		return err
	}
	defer f.Close()

	contents, err := c.repo.GetChangelog(c.title)
	if err != nil {
		logger.Error("Could not get the changelog from the repository: %v", err)
		return err
	}
	_, err = f.WriteString(contents)
	if err != nil {
		logger.Error("Could not write the changelog to the file: %v", err)
		return err
	}
	if err = f.Sync(); err != nil {
		logger.Error("Could not sync the changelog file: %v", err)
		return err
	}

	return nil
}

func NewChangelog(repo repo.VcsRepo, pkgMeta *pkg.PkgMeta, title string, pkgDir string, topDir string) (*Changelog, error) {
	var changelog *Changelog

	if pkgMeta.ChangelogFromGitHub {
		if !repo.IsGitHubHosted() {
			return nil, fmt.Errorf("ChangelogFromGitHub is true but the repository is not hosted on GitHub")
		}

		tag := repo.GetCurrentTag()
		if tag == "" {
			return nil, fmt.Errorf("Could not get the current tag, can't get the changelog from the release")
		}

		slug := repo.GetGitHubSlug()
		if slug == "" {
			return nil, fmt.Errorf("Could not get the GitHub slug, can't get the changelog from the release")
		}

		cache, err := cachedir.Get()
		if err != nil {
			return nil, fmt.Errorf("Could not get the cache directory: %w", err)
		}

		release, err := github.GetRelease(slug, tag)
		if err != nil {
			return nil, fmt.Errorf("Could not get the release: %w", err)
		}

		// Write the release body to the cache directory
		normalizedSlug := strings.ReplaceAll(slug, "/", "_")
		releaseBodyPath := filepath.Join(cache, fmt.Sprintf("%s_%s.md", normalizedSlug, tag))
		f, err := os.OpenFile(releaseBodyPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("Could not create the temporary github changelog file: %w", err)
		}
		defer f.Close()

		_, err = f.WriteString(release.Body)
		if err != nil {
			return nil, fmt.Errorf("Could not write the release body to the temporary github changelog file: %w", err)
		}

		if err = f.Sync(); err != nil {
			return nil, fmt.Errorf("Could not sync the temporary github changelog file: %w", err)
		}

		changelog = &Changelog{
			repo:                repo,
			title:               title,
			pkgDir:              pkgDir,
			topDir:              topDir,
			PreExistingFilePath: releaseBodyPath,
			MarkupType:          MarkdownMT,
			generateChangelog:   false,
			requiresCleanup:     true,
		}

		return changelog, nil
	}

	if pkgMeta.ManualChangelog.Filename != "" {
		changelog = &Changelog{
			topDir:              topDir,
			title:               title,
			pkgDir:              pkgDir,
			repo:                repo,
			PreExistingFilePath: pkgMeta.ManualChangelog.Filename,
			MarkupType:          MarkupType(pkgMeta.ManualChangelog.MarkupType),
			generateChangelog:   false,
			requiresCleanup:     false,
		}

		if err := changelog.verifyManualChangelog(); err == nil {
			return changelog, nil
		} else if err == ErrManualChangelogNotFound || err == ErrInvalidMarkupType {
			logger.Warn("%v: will attempt to generate from commits instead", err)
		} else {
			return nil, err
		}
	}

	// If the manual changelog wasn't found or was invalid, generate one
	changelog = &Changelog{
		title:               title,
		pkgDir:              pkgDir,
		repo:                repo,
		MarkupType:          MarkdownMT,
		PreExistingFilePath: filepath.Join(pkgDir, "CHANGELOG.md"),
		generateChangelog:   true,
		requiresCleanup:     false,
	}

	return changelog, nil

}
