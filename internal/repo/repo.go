package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/McTalian/wow-build-tools/internal/external"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/tokens"
)

type VcsRepo interface {
	IsIgnored(path string, isDir bool) bool
	IsGitHubHosted() bool
	GetGitHubSlug() string
	GetInjectionValues(stm *tokens.SimpleTokenMap) error
	GetFileInjectionValues(filePath string) (*tokens.SimpleTokenMap, error)
	GetRepoRoot() string
	GetChangelog(title string) (string, error)
	GetCurrentTag() string
	GetPreviousVersion() string
	GetProjectVersion() string
}

type BaseVcsRepo struct {
	VcsRepo
	repo            *Repo
	CurrentTag      string
	PreviousVersion string
	ProjectVersion  string
}

func (bV *BaseVcsRepo) GetInjectionValues(stm *tokens.SimpleTokenMap) error {
	return nil
}

func (bV *BaseVcsRepo) IsGitHubHosted() bool {
	return false
}

func (bV *BaseVcsRepo) GetGitHubSlug() string {
	return ""
}

func (bV *BaseVcsRepo) IsIgnored(path string, isDir bool) bool {
	return false
}

func (bV *BaseVcsRepo) GetRepoRoot() string {
	return bV.repo.GetRepoRoot()
}

func (bV *BaseVcsRepo) GetChangelog(title string) (string, error) {
	return "", nil
}

func (bV *BaseVcsRepo) GetCurrentTag() string {
	return bV.CurrentTag
}

func (bV *BaseVcsRepo) GetPreviousVersion() string {
	return bV.PreviousVersion
}

func (bV *BaseVcsRepo) GetProjectVersion() string {
	return bV.ProjectVersion
}

type Repo struct {
	repoRoot    string
	repoVcsType external.VcsType
}

func (r *Repo) GetRepoRoot() string {
	return r.repoRoot
}

func (r *Repo) GetVcsType() external.VcsType {
	return r.repoVcsType
}

func (r *Repo) GetVcsTypeString() string {
	return r.repoVcsType.ToString()
}

func (r *Repo) String() string {
	return fmt.Sprintf("TopDir: %s\nVcsType: %s", r.repoRoot, r.repoVcsType.ToString())
}

func NewRepo(topDir string) (*Repo, error) {
	repoVcsType := external.Unknown

	dirToCheck, err := filepath.Abs(topDir)
	if err != nil {
		return nil, err
	}

	logger.Verbose("Checking directory: %s", dirToCheck)

	iterations := 0
	for {
		if dirToCheck == "" {
			break
		}

		if iterations > 3 {
			break
		}

		// Check if the directory is a git repository
		if _, err := os.Stat(filepath.Join(dirToCheck, ".git")); err == nil {
			repoVcsType = external.Git
			break
		}

		// Check if the directory is a svn repository
		if _, err := os.Stat(filepath.Join(dirToCheck, ".svn")); err == nil {
			repoVcsType = external.Svn
			break
		}

		// Check if the directory is a hg repository
		if _, err := os.Stat(filepath.Join(dirToCheck, ".hg")); err == nil {
			repoVcsType = external.Hg
			break
		}

		dirToCheck = filepath.Join(dirToCheck, "..")
		iterations++
		if iterations <= 3 {
			logger.Verbose("Checking for a vcs directory in: %s", dirToCheck)
		}
	}

	if repoVcsType == external.Unknown {
		return nil, fmt.Errorf("could not determine the VCS type of the repository")
	}

	newTopDir := topDir
	if iterations > 0 {
		newTopDir = dirToCheck
	}

	return &Repo{
		repoRoot:    newTopDir,
		repoVcsType: repoVcsType,
	}, nil
}
