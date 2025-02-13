package changelog

import (
	"fmt"
	"os"
	"path/filepath"

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
	projectName         string
	pkgDir              string
	PreExistingFilePath string
	MarkupType          MarkupType
	generateChangelog   bool
}

var ErrManualChangelogNotFound = fmt.Errorf("Manual changelog file not found")
var ErrInvalidMarkupType = fmt.Errorf("Invalid markup type")

func (c *Changelog) verifyManualChangelog() error {
	if c.PreExistingFilePath == "" {
		return ErrManualChangelogNotFound
	}
	pkgDir := c.pkgDir

	relativeFilePath := c.PreExistingFilePath
	topDirChangelogPath := filepath.Join(c.repo.GetTopDir(), relativeFilePath)
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

func (c *Changelog) GetChangelog() (string, error) {
	if !c.generateChangelog {
		_, err := os.Stat(c.PreExistingFilePath)
		if err == nil {
			contents, err := os.ReadFile(c.PreExistingFilePath)
			if err != nil {
				logger.Error("Could not read the manual changelog file (even though it exists): %v", err)
				return "", err
			}
			return string(contents), nil
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
		return "", err
	}
	defer f.Close()

	contents, err := c.repo.GetChangelog(c.projectName)
	if err != nil {
		logger.Error("Could not get the changelog from the repository: %v", err)
		return "", err
	}
	_, err = f.WriteString(contents)
	if err != nil {
		logger.Error("Could not write the changelog to the file: %v", err)
		return "", err
	}

	return c.PreExistingFilePath, nil
}

func NewChangelog(repo repo.VcsRepo, pkgMeta *pkg.PkgMeta, projectName string, pkgDir string) (*Changelog, error) {
	var changelog *Changelog
	if pkgMeta.ManualChangelog.Filename != "" {
		changelog = &Changelog{
			projectName:         projectName,
			pkgDir:              pkgDir,
			repo:                repo,
			PreExistingFilePath: pkgMeta.ManualChangelog.Filename,
			MarkupType:          MarkupType(pkgMeta.ManualChangelog.MarkupType),
			generateChangelog:   false,
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
		projectName:         projectName,
		pkgDir:              pkgDir,
		repo:                repo,
		MarkupType:          MarkdownMT,
		PreExistingFilePath: filepath.Join(pkgDir, "CHANGELOG.md"),
		generateChangelog:   true,
	}

	return changelog, nil

}
