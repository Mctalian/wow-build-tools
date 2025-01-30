package external

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type GitExternal struct {
	BaseVcs
	metadata *ExternalEntry
}

func (gE *GitExternal) Checkout() error {
	// Determine the cached repository path.
	repoCachePath := gE.getRepoCachePath()
	e := gE.metadata

	// Instantiate the last-updated helper for the cache.
	helper := NewLastUpdatedHelper(repoCachePath, ".lastUpdated", e.LogGroup)
	lastUpdatedPath := helper.FilePath(e.Tag)

	// If forced, delete any existing marker; otherwise, if the marker exists and is fresh, skip heavy operations.
	if helper.Force {
		if err := helper.Delete(lastUpdatedPath); err != nil {
			return fmt.Errorf("GIT: failed to delete lastUpdated marker: %w", err)
		}
	} else {
		if stale, err := helper.IsStale(lastUpdatedPath, 24*time.Hour); err != nil {
			return err
		} else if !stale {
			e.LogGroup.Verbose("GIT: Cache is up-to-date for %s", e.DestPath)
			return nil
		}
	}

	// Clone or update the cached repository.
	var repo *git.Repository
	if _, err := os.Stat(repoCachePath); os.IsNotExist(err) {
		e.LogGroup.Verbose("GIT: Cloning %s into cache: %s", e.URL, repoCachePath)
		repo, err = git.PlainClone(repoCachePath, false, &git.CloneOptions{
			URL:      e.URL,
			Progress: nil,
		})
		if err != nil {
			return fmt.Errorf("failed to clone into cache %s: %w", e.URL, err)
		}
	} else {
		repo, err = git.PlainOpen(repoCachePath)
		if err != nil {
			return fmt.Errorf("failed to open cache %s: %w", repoCachePath, err)
		}
		e.LogGroup.Verbose("GIT: Fetching latest changes in cache for %s", e.URL)
		if err = repo.Fetch(&git.FetchOptions{
			Prune: true,
			Tags:  git.AllTags,
		}); err != nil && err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("failed to update cache: %w", err)
		}
	}

	// Retrieve the worktree.
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Perform the checkout based on the type.
	switch e.CheckoutType {
	case "branch":
		e.LogGroup.Verbose("GIT: Checking out branch %s", e.Tag)
		err = worktree.Checkout(&git.CheckoutOptions{
			Branch: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", e.Tag)),
			Create: false,
			Force:  true,
		})
		if err != nil {
			return fmt.Errorf("git checkout branch failed: %w", err)
		}
	case "tag":
		e.LogGroup.Verbose("GIT: Checking out tag %s", e.Tag)
		err = worktree.Checkout(&git.CheckoutOptions{
			Branch: plumbing.ReferenceName(fmt.Sprintf("refs/tags/%s", e.Tag)),
			Create: false,
			Force:  true,
		})
		if err != nil {
			return fmt.Errorf("git checkout tag failed: %w", err)
		}
	case "commit":
		e.LogGroup.Verbose("GIT: Checking out commit %s", e.Tag)
		if plumbing.IsHash(e.Tag) {
			commitHash := plumbing.NewHash(e.Tag)
			err = worktree.Checkout(&git.CheckoutOptions{
				Hash:  commitHash,
				Force: true,
			})
			if err != nil {
				return fmt.Errorf("git checkout of commit %s failed: %w", e.Tag, err)
			}
		} else if len(e.Tag) >= 7 {
			cIter, err := repo.CommitObjects()
			if err != nil {
				return fmt.Errorf("failed to get commit objects with abbreviated hash %s: %w", e.Tag, err)
			}
			defer cIter.Close()

			var ErrFoundCommit = fmt.Errorf("found commit")

			var commit *object.Commit
			if err = cIter.ForEach(func(c *object.Commit) error {
				if strings.HasPrefix(c.Hash.String(), e.Tag) {
					commit = c
					return ErrFoundCommit
				}
				return nil
			}); err != nil && err != ErrFoundCommit {
				return fmt.Errorf("failed to iterate commit objects with abbreviated hash %s: %w", e.Tag, err)
			}
			if commit == nil {
				return fmt.Errorf("commit not found with abbreviated hash %s", e.Tag)
			}
			err = worktree.Checkout(&git.CheckoutOptions{
				Hash:  commit.Hash,
				Force: true,
			})
			if err != nil {
				return fmt.Errorf("git checkout of commit %s failed: %w", commit.Hash.String(), err)
			}
		} else {
			return fmt.Errorf("invalid commit hash or abbreviated hash: %s", e.Tag)
		}
	default:
		e.LogGroup.Verbose("GIT: Checking out default branch")
		refs, err := repo.References()
		if err != nil {
			return fmt.Errorf("failed to get references: %w", err)
		}
		defaultBranch := ""
		if err = refs.ForEach(func(ref *plumbing.Reference) error {
			if ref.Type() == plumbing.SymbolicReference && ref.Name().String() == "HEAD" {
				if defaultBranch != "" {
					return fmt.Errorf("multiple default branches found")
				}
				defaultBranch = strings.TrimPrefix(ref.Target().String(), "refs/heads/")
			}
			return nil
		}); err != nil {
			return fmt.Errorf("failed to iterate references: %w", err)
		}
		if defaultBranch == "" {
			return fmt.Errorf("failed to determine default branch")
		}
		err = worktree.Checkout(&git.CheckoutOptions{
			Branch: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", defaultBranch)),
			Create: false,
			Force:  true,
		})
		if err != nil {
			return fmt.Errorf("git checkout %s failed: %w", defaultBranch, err)
		}
		e.Tag = defaultBranch
	}

	// Write the marker file now that the checkout is complete.
	if err := helper.Write(lastUpdatedPath); err != nil {
		return err
	}

	e.LogGroup.Debug("GIT: %s checkout successful: %s", e.DestPath, e.Tag)
	return nil
}

// getRepoCachePath returns the cache path for a specific repository.
func (e *GitExternal) getRepoCachePath() string {
	return e.metadata.RepoCacheDir
}

func NewGitExternal(e *ExternalEntry) (*GitExternal, error) {
	if e.EType != Git {
		return nil, fmt.Errorf("external entry is not a git type")
	}

	return &GitExternal{
		metadata: e,
	}, nil
}
