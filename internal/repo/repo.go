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
	GetInjectionValues(stm *tokens.SimpleTokenMap) error
	GetFileInjectionValues(filePath string) (*tokens.SimpleTokenMap, error)
}

type BaseVcsRepo struct {
	VcsRepo
}

func (bV *BaseVcsRepo) GetInjectionValues(stm *tokens.SimpleTokenMap) error {
	return nil
}

func (uR *BaseVcsRepo) IsIgnored(path string, isDir bool) bool {
	return false
}

type Repo struct {
	topDir      string
	repoVcsType external.VcsType
}

func (r *Repo) GetTopDir() string {
	return r.topDir
}

func (r *Repo) GetVcsType() external.VcsType {
	return r.repoVcsType
}

func (r *Repo) GetVcsTypeString() string {
	return r.repoVcsType.ToString()
}

func (r *Repo) String() string {
	return fmt.Sprintf("TopDir: %s\nVcsType: %s", r.topDir, r.repoVcsType.ToString())
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

		if iterations > 2 {
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
		if iterations <= 2 {
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
		topDir:      newTopDir,
		repoVcsType: repoVcsType,
	}, nil
}
