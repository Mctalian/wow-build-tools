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
	repo                *Repo
	gitRepo             *git.Repository
	worktree            *git.Worktree
	headRef             *plumbing.Reference
	commit              *object.Commit
	ignorePatterns      []gitignore.Pattern
	originURL           string
	isGitHubUrl         bool
	gitHubUrl           string
	gitHubSlug          string
	projectTimestamp    int64
	projectHash         string
	previousVersionHash string
}

func (gR *GitRepo) GetCurrentTag() string {
	return gR.CurrentTag
}

func (gR *GitRepo) GetPreviousVersion() string {
	return gR.PreviousVersion
}

func (gR *GitRepo) GetProjectVersion() string {
	return gR.ProjectVersion
}

func (gR *GitRepo) GetRepoRoot() string {
	return gR.repo.GetRepoRoot()
}

func (gR *GitRepo) parseGitHubURL() {
	gitHubUrl := strings.TrimSuffix(strings.TrimSuffix(gR.originURL, ".git"), "/")
	segments := strings.Split(gitHubUrl, "github.com")
	if len(segments) == 2 {
		httpify := strings.NewReplacer("git@", "https://", "github.com:", "github.com/")
		gitHubUrl = httpify.Replace(gitHubUrl)
		gR.gitHubUrl = gitHubUrl
		slug := strings.TrimPrefix(strings.TrimPrefix(segments[1], "/"), ":")
		slugSegments := strings.Split(slug, "/")
		if len(slugSegments) == 2 {
			gR.gitHubSlug = slug
		} else {
			logger.Warn("Invalid GitHub slug: %s", slug)
		}
	} else {
		logger.Warn("Invalid GitHub URL: %s", gitHubUrl)
	}
}

func (gR *GitRepo) openRepo() error {
	var err error
	gR.gitRepo, err = git.PlainOpen(gR.repo.GetRepoRoot())
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	originRemote, err := gR.gitRepo.Remote("origin")
	if err != nil {
		return fmt.Errorf("failed to get origin remote: %w", err)
	}

	gR.originURL = originRemote.Config().URLs[0]
	gR.isGitHubUrl = strings.Contains(gR.originURL, "github.com")
	if gR.isGitHubUrl {
		gR.parseGitHubURL()
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

func (gR *GitRepo) sortReferences(references []*plumbing.Reference) {
	slices.SortFunc(references, func(i, j *plumbing.Reference) int {
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
}

func sortTagNameHashes(tagNameHashes []tagNameHash) {
	slices.SortFunc(tagNameHashes, func(e1, e2 tagNameHash) int {
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
}

type tagNameHash struct {
	Name string
	Hash plumbing.Hash
}

func newTagNameHash(name string, hash plumbing.Hash) tagNameHash {
	return tagNameHash{Name: name, Hash: hash}
}

// GetProjectTag retrieves a tag associated with HEAD and provides both an “always” (fallback) and the most recent tag.
// It returns two values: siTag (for the exact tag) and siTagAbbrev (for the abbreviated form).
func (gR *GitRepo) getProjectTag() (tag7 string, tag0 string, err error) {
	var githubRefName, githubTag7, githubTag0 string
	if githubRef := os.Getenv("GITHUB_REF"); strings.HasPrefix(githubRef, "refs/tags/") {
		logger.Warn("Detected GitHub Run")
		githubRefName = os.Getenv("GITHUB_REF_NAME")
	}

	refIter, err := gR.gitRepo.Tags()
	if err != nil {
		return "", "", fmt.Errorf("failed to get tag objects: %w", err)
	}
	defer refIter.Close()

	matchingTags := make([]*plumbing.Reference, 0)
	tagCount := 0
	tagNameHashes := make([]tagNameHash, 0)
	err = refIter.ForEach(func(t *plumbing.Reference) error {
		tagCount++
		if t.Hash() == gR.headRef.Hash() {
			matchingTags = append(matchingTags, t)
		}

		if t.Type() != plumbing.HashReference {
			// logger.Verbose("Skipping non-hash reference %s", ref.Name())
			return nil
		}

		tagName := t.Name().Short()
		if tagName == "" {
			// logger.Verbose("Skipping empty tag name")
			return nil
		}

		if t.Name().Short() == plumbing.ReferenceName(githubRefName).String() {
			githubTag7 = tagName
			githubTag0 = tagName
		}

		if strings.Contains(tagName, "alpha") || strings.Contains(tagName, "beta") {
			// logger.Verbose("Skipping alpha or beta tag %s", tagName)
			return nil
		}

		tag7 = tagName
		tagNameHashes = append(tagNameHashes, newTagNameHash(tagName, t.Hash()))
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

	sortTagNameHashes(tagNameHashes)

	if githubTag7 != "" || len(matchingTags) > 0 {
		if githubTag7 != "" {
			tag7 = githubTag7
			tag0 = githubTag0
		} else if len(matchingTags) == 1 {
			tag7 = matchingTags[0].Name().Short()
			tag0 = tag7
		} else {
			gR.sortReferences(matchingTags)
			tag7 = matchingTags[0].Name().Short()
			tag0 = tag7
		}

		matchingTagHashIndex := slices.IndexFunc(tagNameHashes, func(e tagNameHash) bool {
			return e.Name == tag7
		})
		if matchingTagHashIndex != -1 && matchingTagHashIndex < len(tagNameHashes)-1 {
			gR.PreviousVersion = tagNameHashes[matchingTagHashIndex+1].Name
			gR.previousVersionHash = tagNameHashes[matchingTagHashIndex+1].Hash.String()
		}

		return tag7, tag0, nil
	}

	var latestTag tagNameHash
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
	gR.PreviousVersion = latestTag.Name
	gR.previousVersionHash = latestTag.Hash.String()

	return tag7, tag0, nil
}

func (gR *GitRepo) buildChangelogHeader(title string) string {
	var header strings.Builder
	header.WriteString(fmt.Sprintf("# %s\n\n", title))

	var versionLink, changeLink string
	previousReleasesLink := fmt.Sprintf("[Previous Releases](%s/releases)", gR.gitHubUrl)
	if gR.gitHubUrl == "" {
		versionLink = gR.ProjectVersion
		changeLink = ""
		previousReleasesLink = ""
	} else if gR.PreviousVersion != "" && gR.CurrentTag != "" {
		versionLink = fmt.Sprintf("[%s](%s/tree/%s)", gR.ProjectVersion, gR.gitHubUrl, gR.CurrentTag)
		changeLink = fmt.Sprintf("[Full Changelog](%s/compare/%s...%s)", gR.gitHubUrl, gR.PreviousVersion, gR.CurrentTag)
	} else if gR.PreviousVersion != "" && gR.CurrentTag == "" {
		versionLink = fmt.Sprintf("[%s](%s/tree/%s)", gR.ProjectVersion, gR.gitHubUrl, gR.projectHash)
		changeLink = fmt.Sprintf("[Full Changelog](%s/compare/%s...%s)", gR.gitHubUrl, gR.PreviousVersion, gR.projectHash)
	} else if gR.PreviousVersion == "" && gR.CurrentTag != "" {
		versionLink = fmt.Sprintf("[%s](%s/tree/%s)", gR.ProjectVersion, gR.gitHubUrl, gR.CurrentTag)
		changeLink = fmt.Sprintf("[Full Changelog](%s/commits/%s)", gR.gitHubUrl, gR.CurrentTag)
	} else {
		versionLink = fmt.Sprintf("[%s](%s/tree/%s)", gR.ProjectVersion, gR.gitHubUrl, gR.projectHash)
		changeLink = fmt.Sprintf("[Full Changelog](%s/commits/%s)", gR.gitHubUrl, gR.projectHash)
	}
	t := time.Unix(gR.projectTimestamp, 0).UTC()
	changelogDate := t.Format("2006-01-02")
	header.WriteString(fmt.Sprintf("## %s (%s)\n", versionLink, changelogDate))
	header.WriteString(fmt.Sprintf("%s %s\n\n", changeLink, previousReleasesLink))

	return header.String()
}

func (gR *GitRepo) GetChangelog(title string) (string, error) {
	commitIter, err := gR.gitRepo.Log(&git.LogOptions{
		From:  gR.headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get commit log: %w", err)
	}

	ErrFoundPrevVersion := fmt.Errorf("found previous version")
	var changelog strings.Builder
	changelog.WriteString(gR.buildChangelogHeader(title))
	err = commitIter.ForEach(func(c *object.Commit) error {
		if c.Hash.String() == gR.previousVersionHash {
			return ErrFoundPrevVersion
		}
		if strings.TrimSpace(c.Message) == "" {
			return nil
		}

		if strings.Contains(c.Message, "git-svn-id:") {
			return nil
		}

		if strings.Contains(c.Message, "This reverts commit") {
			return nil
		}

		normalizedMessage := strings.TrimSpace(c.Message)
		normalizedMessage = strings.ReplaceAll(normalizedMessage, "_", "\\_")
		normalizedMessage = strings.ReplaceAll(normalizedMessage, "[ci skip]", "")
		normalizedMessage = strings.ReplaceAll(normalizedMessage, "[skip ci]", "")

		changelog.WriteString(fmt.Sprintf("- %s  \n", normalizedMessage))
		return nil
	})
	if err != nil && err != ErrFoundPrevVersion {
		return "", fmt.Errorf("failed to iterate commits: %w", err)
	}

	return changelog.String(), nil
}

func (gR *GitRepo) GetInjectionValues(stm *tokens.SimpleTokenMap) error {
	projectHash, err := gR.getProjectHash()
	if err != nil {
		return err
	}
	gR.projectHash = projectHash
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
	gR.projectTimestamp = projectTimestamp
	stm.Add(tokens.ProjectTimestamp, strconv.FormatInt(projectTimestamp, 10))
	t := time.Unix(projectTimestamp, 0).UTC()
	stm.Add(tokens.ProjectDateIso, t.Format("2006-01-02T15:04:05Z"))
	stm.Add(tokens.ProjectDateInteger, t.Format("20060102150405"))

	projectRevision, err := gR.getProjectRevision()
	if err != nil {
		return err
	}
	stm.Add(tokens.ProjectRevision, strconv.Itoa(projectRevision))

	tag7, tag0, err := gR.getProjectTag()
	if err != nil {
		return err
	}
	stm.Add(tokens.ProjectVersion, tag7)
	if tag7 == tag0 {
		gR.CurrentTag = tag0
	}
	gR.ProjectVersion = tag7

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
	gR := GitRepo{repo: r}

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
