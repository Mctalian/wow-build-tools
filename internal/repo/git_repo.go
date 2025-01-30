package repo

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hashicorp/go-version"

	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/tokens"
)

type GitRepo struct {
	BaseVcsRepo
	r              *Repo
	gitRepo        *git.Repository
	worktree       *git.Worktree
	headRef        *plumbing.Reference
	commit         *object.Commit
	ignorePatterns []gitignore.Pattern
}

func (gR *GitRepo) openRepo() error {
	var err error
	gR.gitRepo, err = git.PlainOpen(gR.r.GetTopDir())
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	gR.worktree, err = gR.gitRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	return nil
}

func (gR *GitRepo) getUntrackedFiles() ([]gitignore.Pattern, error) {
	// Get the untracked files.
	status, err := gR.worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree status: %v", err)
	}

	var untrackedFiles []gitignore.Pattern
	for file, stat := range status {
		if stat.Worktree == git.Untracked {
			untrackedFiles = append(untrackedFiles, gitignore.ParsePattern(file, []string{}))
		}
	}

	return untrackedFiles, nil
}

func (gR *GitRepo) getIgnores() ([]gitignore.Pattern, error) {
	patterns, err := gitignore.ReadPatterns(gR.worktree.Filesystem, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to read patterns: %w", err)
	}

	logger.Verbose("Found %d gitignore patterns", len(patterns))

	untrackedFiles, err := gR.getUntrackedFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get untracked files: %w", err)
	}

	logger.Verbose("Found %d untracked files", len(untrackedFiles))

	patterns = append(patterns, untrackedFiles...)

	return patterns, nil
}

func (gR *GitRepo) IsIgnored(path string, isDir bool) bool {
	for _, pattern := range gR.ignorePatterns {
		result := pattern.Match(strings.Split(path, "/"), isDir)
		if result == gitignore.Exclude {
			return true
		}
	}

	return false
}

func (gR *GitRepo) populateCommitInfo() error {
	var err error
	gR.headRef, err = gR.gitRepo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	gR.commit, err = gR.gitRepo.CommitObject(gR.headRef.Hash())
	if err != nil {
		return fmt.Errorf("failed to get commit object: %w", err)
	}

	return nil
}

func (gR *GitRepo) getProjectHash() (string, error) {
	// The Hash method returns the commit hash; convert it to a string.
	projectHash := gR.headRef.Hash().String()
	if projectHash == "" {
		return "", fmt.Errorf("failed to get project hash")
	}
	return projectHash, nil
}

func (gR *GitRepo) getProjectAuthor() (string, error) {
	return gR.commit.Author.Name, nil
}

func (gR *GitRepo) getProjectTimestamp() (int64, error) {
	return gR.commit.Author.When.Unix(), nil
}

func (gR *GitRepo) getProjectRevision() (int, error) {
	commitIter, err := gR.gitRepo.Log(&git.LogOptions{
		From: gR.headRef.Hash(),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get commit log: %w", err)
	}
	defer commitIter.Close()

	// Iterate through the commits and count them.
	count := 0
	err = commitIter.ForEach(func(c *object.Commit) error {
		count++
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("error iterating commits: %w", err)
	}

	return count, nil
}

// GetProjectTag retrieves a tag associated with HEAD and provides both an “always” (fallback) and the most recent tag.
// It returns two values: siTag (for the exact tag) and siTagAbbrev (for the abbreviated form).
func (gR *GitRepo) getProjectTag() (tag7 string, tag0 string, err error) {
	refIter, err := gR.gitRepo.Tags()
	if err != nil {
		return "", "", fmt.Errorf("failed to get tag objects: %w", err)
	}
	defer refIter.Close()

	matchingTags := make([]*plumbing.Reference, 0)
	tagCount := 0
	err = refIter.ForEach(func(t *plumbing.Reference) error {
		tagCount++
		if t.Hash() == gR.headRef.Hash() {
			matchingTags = append(matchingTags, t)
		}
		return nil
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to iterate tags: %w", err)
	}

	if tagCount == 0 {
		// No tags found.
		tag0 = gR.headRef.Hash().String()[:7]
		tag7 = tag0
		return
	}

	// If there is at least one tag that points to HEAD, use it.
	if len(matchingTags) > 0 {
		if len(matchingTags) == 1 {
			tag7 = matchingTags[0].Name().Short()
			logger.Verbose("Found tag %s", tag7)
			tag0 = tag7
			return tag7, tag0, nil
		}

		slices.SortFunc(matchingTags, func(i, j *plumbing.Reference) int {
			iCommit, err := gR.gitRepo.CommitObject(i.Hash())
			if err != nil {
				return 1
			}
			jCommit, err := gR.gitRepo.CommitObject(j.Hash())
			if err != nil {
				return -1
			}

			if iCommit.Committer.When.Before(jCommit.Committer.When) {
				return -1
			}
			if iCommit.Committer.When.After(jCommit.Committer.When) {
				return 1
			}

			// If committer dates are equal, sort by tag values.
			// Secondary sort: Compare tag names using version semantics.
			// Remove the "v" prefix if present.
			nameI := strings.TrimPrefix(i.Name().Short(), "v")
			nameJ := strings.TrimPrefix(j.Name().Short(), "v")

			// Attempt to parse the tag names as versions.
			verI, errI := version.NewVersion(nameI)
			verJ, errJ := version.NewVersion(nameJ)
			if errI == nil && errJ == nil {
				if verI.LessThan(verJ) {
					return -1
				}
				if verJ.LessThan(verI) {
					return 1
				}
				return 0
			}

			// Fallback: lexicographical order if version parsing fails.
			if i.Name().Short() < j.Name().Short() {
				return -1
			}
			if i.Name().Short() > j.Name().Short() {
				return 1
			}

			iTag, err := object.GetTag(gR.gitRepo.Storer, i.Hash())
			if err != nil {
				return 1
			}
			jTag, err := object.GetTag(gR.gitRepo.Storer, j.Hash())
			if err != nil {
				return -1
			}
			// If committer dates are equal and the tag values are equal, sort by tagger date descending.
			if iTag.Tagger.When.After(jTag.Tagger.When) {
				return -1
			}
			if iTag.Tagger.When.Before(jTag.Tagger.When) {
				return 1
			}
			return 0
		})

		tag7 = matchingTags[0].Name().Short()
		tag0 = tag7
	}

	iter, err := gR.gitRepo.Tags()
	if err != nil {
		return "", "", fmt.Errorf("failed to get tags: %w", err)
	}
	defer iter.Close()

	type TagNameHash struct {
		Name string
		Hash plumbing.Hash
	}

	var githubRefName string
	if githubRef := os.Getenv("GITHUB_REF"); strings.HasPrefix(githubRef, "refs/tags/") {
		githubRefName = os.Getenv("GITHUB_REF_NAME")
	}

	ErrFoundGHTagRef := fmt.Errorf("found GitHub tag ref")

	tagNameHashes := make([]TagNameHash, 0)
	if err = iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() != plumbing.HashReference {
			// logger.Verbose("Skipping non-hash reference %s", ref.Name())
			return nil
		}

		tagName := ref.Name().Short()
		if tagName == "" {
			// logger.Verbose("Skipping empty tag name")
			return nil
		}

		if ref.Name() == plumbing.ReferenceName(githubRefName) {
			tag7 = tagName
			tag0 = tagName
			return ErrFoundGHTagRef
		}

		if strings.Contains(tagName, "alpha") || strings.Contains(tagName, "beta") {
			// logger.Verbose("Skipping alpha or beta tag %s", tagName)
			return nil
		}

		tag7 = tagName
		tagNameHashes = append(tagNameHashes, TagNameHash{Name: tagName, Hash: ref.Hash()})
		return nil
	}); err != nil {
		if err != ErrFoundGHTagRef {
			return "", "", fmt.Errorf("failed to iterate tags: %w", err)
		} else {
			return tag7, tag0, nil
		}
	}

	slices.SortFunc(tagNameHashes, func(e1, e2 TagNameHash) int {
		tag1 := e1.Name
		tag2 := e2.Name
		tag1 = strings.TrimPrefix(tag1, "v")
		tag2 = strings.TrimPrefix(tag2, "v")

		tag1Parts := strings.Split(tag1, ".")
		tag2Parts := strings.Split(tag2, ".")

		for i := 0; i < len(tag1Parts) && i < len(tag2Parts); i++ {
			tag1Part, err := strconv.Atoi(tag1Parts[i])
			if err != nil {
				logger.Verbose("Failed to parse tag part %s: %v", tag1Parts[i], err)
				panic(err)
			}

			tag2Part, err := strconv.Atoi(tag2Parts[i])
			if err != nil {
				logger.Verbose("Failed to parse tag part %s: %v", tag2Parts[i], err)
				panic(err)
			}

			if tag1Part < tag2Part {
				return 1
			} else if tag1Part > tag2Part {
				return -1
			}
		}

		return 0
	})

	var latestTag TagNameHash
	if len(tagNameHashes) > 0 {
		latestTag = tagNameHashes[0]
	}

	// Get the number of commits since the latest tag.
	opts := &git.LogOptions{
		From: gR.headRef.Hash(),
	}
	cIter, err := gR.gitRepo.Log(opts)
	if err != nil {
		return "", "", fmt.Errorf("failed to get commit log: %w", err)
	}
	defer cIter.Close()

	commitCount := 0
	ErrFoundLatestTag := fmt.Errorf("found latest tag")
	if err = cIter.ForEach(func(c *object.Commit) error {
		if c.Hash == latestTag.Hash {
			return ErrFoundLatestTag
		}
		commitCount++
		return nil
	}); err != nil && err != ErrFoundLatestTag {
		return "", "", fmt.Errorf("failed to count commits: %w", err)
	}

	tag7 = fmt.Sprintf("%s-%d-g%s", latestTag.Name, commitCount, gR.headRef.Hash().String()[:7])
	tag0 = latestTag.Name

	return tag7, tag0, nil
}

func (gR *GitRepo) GetInjectionValues(stm *tokens.SimpleTokenMap) error {
	projectHash, err := gR.getProjectHash()
	if err != nil {
		return err
	}
	stm.Add(tokens.ProjectHash, projectHash)
	stm.Add(tokens.ProjectAbbrevHash, projectHash[:7])

	projectAuthor, err := gR.getProjectAuthor()
	if err != nil {
		return err
	}
	stm.Add(tokens.ProjectAuthor, projectAuthor)

	projectTimestamp, err := gR.getProjectTimestamp()
	if err != nil {
		return err
	}
	stm.Add(tokens.ProjectTimestamp, strconv.FormatInt(projectTimestamp, 10))
	t := time.Unix(projectTimestamp, 0).UTC()
	stm.Add(tokens.ProjectDateIso, t.Format("2006-01-02T15:04:05Z"))
	stm.Add(tokens.ProjectDateInteger, t.Format("20060102150405"))

	projectRevision, err := gR.getProjectRevision()
	if err != nil {
		return err
	}
	stm.Add(tokens.ProjectRevision, strconv.Itoa(projectRevision))

	tag7, _, err := gR.getProjectTag()
	if err != nil {
		return err
	}
	stm.Add(tokens.ProjectVersion, tag7)

	return nil
}

func (gR *GitRepo) GetFileInjectionValues(filePath string) (*tokens.SimpleTokenMap, error) {
	logger.Verbose("Getting file injection values for %s", filePath)
	stm := &tokens.SimpleTokenMap{}

	commitIter, err := gR.gitRepo.Log(&git.LogOptions{
		FileName: &filePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}
	defer commitIter.Close()

	// Iterate through the commits and get the first one.
	latestCommit, _ := commitIter.Next()
	// if err != nil {
	// 	return nil, fmt.Errorf("error getting latest commit for %s (%s): %w", fileName, filePath, err)
	// }
	if latestCommit == nil {
		logger.Warn("No prior commits found for %s, but file tokens were present. Using empty strings", filePath)
		stm.Add(tokens.FileAuthor, "")
		stm.Add(tokens.FileTimestamp, "")
		stm.Add(tokens.FileDateIso, "")
		stm.Add(tokens.FileDateInteger, "")
		stm.Add(tokens.FileHash, "")
		stm.Add(tokens.FileAbbrevHash, "")
		stm.Add(tokens.FileRevision, "")
		return stm, nil
	}

	stm.Add(tokens.FileAuthor, latestCommit.Author.Name)
	fileTimestamp := latestCommit.Author.When.Unix()
	stm.Add(tokens.FileTimestamp, strconv.FormatInt(fileTimestamp, 10))
	t := time.Unix(fileTimestamp, 0).UTC()
	stm.Add(tokens.FileDateIso, t.Format("2006-01-02T15:04:05Z"))
	stm.Add(tokens.FileDateInteger, t.Format("20060102150405"))

	// Get the file's hash.
	fullHash := latestCommit.Hash.String()
	stm.Add(tokens.FileHash, fullHash)

	// Get the file's abbreviated hash.
	abbrevHash := fullHash
	if len(fullHash) >= 7 {
		abbrevHash = fullHash[:7]
	}

	stm.Add(tokens.FileAbbrevHash, abbrevHash)
	stm.Add(tokens.FileRevision, abbrevHash)

	return stm, nil
}

func NewGitRepo(r *Repo) (*GitRepo, error) {
	gR := GitRepo{r: r}
	if err := gR.openRepo(); err != nil {
		return nil, err
	}
	if err := gR.populateCommitInfo(); err != nil {
		return nil, err
	}

	ignorePatterns, err := gR.getIgnores()
	if err != nil {
		return nil, fmt.Errorf("failed to get gitignore patterns: %w", err)
	}

	gR.ignorePatterns = ignorePatterns

	return &gR, nil
}
