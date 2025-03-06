package external

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/McTalian/wow-build-tools/internal/logger"
)

// SvnExternal implements the eType.External interface for SVN repositories.
type SvnExternal struct {
	BaseVcs
	metadata       *ExternalEntry
	forceExternals bool
}

func (s *SvnExternal) lookForCurseSlug() error {
	e := s.metadata
	if e.CurseSlug != "" {
		return nil
	}

	repoCachePath := s.getRepoCachePath()

	// Walk repoCachePath and look for the string @curseforge-project-slug in any file.
	return filepath.Walk(repoCachePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk path: %w", err)
		}
		if e.CurseSlug != "" {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}

		fileStr := string(file)
		if strings.Contains(fileStr, "@curseforge-project-slug") {
			e.LogGroup.Debug("Found @curseforge-project-slug in %s", path)
			slug := strings.Split(fileStr, "@curseforge-project-slug")[1]
			slug = strings.TrimSpace(slug)
			slug = strings.TrimPrefix(slug, ":")
			slug = strings.TrimSpace(slug)
			slug = strings.Split(slug, "@")[0]
			e.CurseSlug = strings.TrimSpace(slug)
			e.LogGroup.Debug("Updated CurseSlug to %s", e.CurseSlug)
			return nil
		}

		return nil
	})
}

// NewSvnExternal creates a new instance of SvnExternal.
func NewSvnExternal(e *ExternalEntry, forceExternals bool) (*SvnExternal, error) {
	if e.EType != Svn {
		return nil, fmt.Errorf("external entry is not an svn type")
	}

	if _, err := exec.LookPath("svn"); err != nil {
		return nil, fmt.Errorf("svn is not installed")
	}

	return &SvnExternal{
		metadata:       e,
		forceExternals: forceExternals,
	}, nil
}

func (s *SvnExternal) getRepoCachePath() string {
	return s.metadata.RepoCacheDir
}

// GetURL returns the SVN repository URL.
func (s *SvnExternal) GetURL() string {
	return s.metadata.URL
}

type svnTagMeta struct {
	Tag    string
	TagUrl string
}

func (s *SvnExternal) getSvnTag() (*svnTagMeta, error) {
	tagUrl := strings.Split(s.GetURL(), "/trunk")[0] + "/tags"

	// Create a helper for lastUpdated markers specific to tag lookups.
	helper := NewLastUpdatedHelper(s.metadata.RepoCacheDir, ".lastUpdated_GetTag", s.forceExternals, s.metadata.LogGroup)

	// Search for any existing lastUpdated marker files.
	pattern := filepath.Join(s.metadata.RepoCacheDir, helper.FilePrefix+"_*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob lastUpdated file: %w", err)
	}

	var lastUpdatePath string
	if len(matches) == 1 {
		lastUpdatePath = matches[0]
	} else if len(matches) > 1 {
		// Remove duplicate marker files.
		for _, match := range matches {
			if err := os.Remove(match); err != nil {
				return nil, fmt.Errorf("failed to remove lastUpdated file: %w", err)
			}
		}
	}

	// If forcing an update, delete any existing marker.
	if helper.Force && lastUpdatePath != "" {
		if err := helper.Delete(lastUpdatePath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("SVN: Failed to remove lastUpdated file: %w", err)
		}
	} else if lastUpdatePath != "" {
		// Check if the marker is still valid (not stale).
		stale, err := helper.IsStale(lastUpdatePath, 24*time.Hour)
		if err != nil {
			return nil, err
		}
		if !stale {
			// Extract the tag from the marker filename.
			base := filepath.Base(lastUpdatePath)
			tag := strings.TrimPrefix(base, helper.FilePrefix+"_")
			return &svnTagMeta{
				Tag:    tag,
				TagUrl: tagUrl,
			}, nil
		}
	}

	// No valid marker found, so query the SVN repository for the latest tag.
	var cmdOutput string
	for i := 0; i < 5; i++ {
		cmd := exec.Command("svn", "log", "--verbose", "--limit", "1", tagUrl)
		output, err := cmd.CombinedOutput()
		if err != nil {
			if i >= 4 {
				return nil, fmt.Errorf("failed to get latest tag: %w, output: %s", err, string(output))
			}
			logger.Verbose("SVN: Failed to get latest tag: %v, retrying...", err)
			time.Sleep(50 * time.Millisecond)
			continue
		}
		cmdOutput = string(output)
		break
	}

	// Parse the command output to extract the tag.
	parts := strings.Split(cmdOutput, "A /tags/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("failed to parse svn log output")
	}
	tagPart := parts[1]
	tag := strings.Split(tagPart, " ")[0]
	if tag == "" {
		return nil, nil
	}

	// Write a new marker file with the obtained tag.
	lastUpdatePath = helper.FilePath(tag)
	if _, err = os.Stat(helper.CacheDir); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(helper.CacheDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
	}
	if err := helper.Write(lastUpdatePath); err != nil {
		return nil, err
	}

	return &svnTagMeta{
		Tag:    tag,
		TagUrl: tagUrl,
	}, nil
}

// Checkout performs an SVN checkout or update using the cache marker.
func (s *SvnExternal) Checkout() error {
	repoCachePath := s.getRepoCachePath()
	e := s.metadata

	newRepoCachePath := repoCachePath
	if e.Path != "" && !strings.Contains(e.URL, e.Path) {
		newRepoCachePath = filepath.Join(repoCachePath, e.Path)
		defer func() { e.RepoCacheDir = newRepoCachePath }()
		if e.EType == Svn {
			e.LogGroup.Warn("%s: Path %s not found in URL %s - having a specific URL is generally more performant for svn checkouts.", e.DestPath, e.Path, e.URL)
		}
		if strings.Contains(e.URL, "/trunk") {
			e.LogGroup.Warn(`Example:
	# .pkgmeta	
	externals:
	  %s: %s/%s
`,
				e.DestPath, e.URL, e.Path,
			)
		}
	}

	// Determine the proper checkout URL based on the type.
	var checkoutURL string
	switch e.CheckoutType {
	case "branch":
		checkoutURL = fmt.Sprintf("%s/branches/%s", e.URL, e.Tag)
	case "tag":
		if e.Tag == "latest" || e.Tag == "" {
			tagMeta, err := s.getSvnTag() // still SVN-specific tag lookup
			if err != nil {
				return err
			}
			if tagMeta == nil {
				return fmt.Errorf("failed to get latest tag")
			}
			e.URL = tagMeta.TagUrl
			e.Tag = tagMeta.Tag
			if e.Path == "" {
				checkoutURL = fmt.Sprintf("%s/%s", e.URL, e.Tag)
			} else {
				checkoutURL = fmt.Sprintf("%s/%s/%s", e.URL, e.Tag, e.Path)
			}
		} else {
			checkoutURL = strings.Replace(e.URL, "/trunk", "/tags/"+e.Tag, 1)
		}
	case "commit":
		checkoutURL = e.URL
	default:
		checkoutURL = e.URL
		e.Tag = "trunk"
	}

	// Prepare the marker filename. For instance, use ".lastUpdated" with an optional tag suffix.
	helper := NewLastUpdatedHelper(repoCachePath, ".lastUpdated", s.forceExternals, e.LogGroup)
	lastUpdatedPath := helper.FilePath(e.Tag)

	// If forcing externals, delete the marker.
	if helper.Force {
		if err := helper.Delete(lastUpdatedPath); err != nil {
			return fmt.Errorf("SVN: %w", err)
		}
	}

	// If the cache directory does not exist, perform an initial checkout.
	if _, err := os.Stat(lastUpdatedPath); os.IsNotExist(err) {
		e.LogGroup.Verbose("SVN: Checking out %s into cache: %s", checkoutURL, repoCachePath)
		args := []string{"checkout"}
		if e.CheckoutType == "commit" && e.Tag != "" {
			args = append(args, "-r", e.Tag)
		}
		args = append(args, checkoutURL, repoCachePath)
		// Add retry logic to handle transient network issues, specifically if E175002 is in the output string.
		for i := 0; i < 5; i++ {
			cmd := exec.Command("svn", args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				outputStr := string(output)
				if i < 4 && strings.Contains(outputStr, "E175002") {
					e.LogGroup.Warn("500 Internal Server Error detected (output: %s), retrying...", outputStr)
					continue
				}
				if i < 4 && strings.Contains(outputStr, "E155000") {
					// If the checkout fails due to a non-empty directory, remove the directory and retry.
					e.LogGroup.Warn("SVN: Non-empty directory detected (output:%s), removing %s and retrying...", outputStr, repoCachePath)
					err = os.RemoveAll(repoCachePath)
					if err != nil {
						return fmt.Errorf("failed to remove existing cache dir at %s: %w", repoCachePath, err)
					}
					continue
				}
				return fmt.Errorf("failed to checkout repository %s: %w, output: %s", e.DestPath, err, outputStr)
			}
		}

		// Write the marker after a successful checkout.
		if err := helper.Write(lastUpdatedPath); err != nil {
			return err
		}
	} else {
		// Otherwise, check if the cache is stale.
		stale, err := helper.IsStale(lastUpdatedPath, 24*time.Hour)
		if err != nil {
			return err
		}
		if !stale {
			e.LogGroup.Verbose("SVN: Cache is up-to-date for %s", e.DestPath)
			if err := s.lookForCurseSlug(); err != nil {
				return err
			}
			return nil
		}

		e.LogGroup.Verbose("SVN: Updating repository cache for %s", checkoutURL)
		args := []string{"update"}
		if e.CheckoutType == "commit" && e.Tag != "" {
			args = append(args, "-r", e.Tag)
		}
		for i := 0; i < 5; i++ {
			cmd := exec.Command("svn", args...)
			cmd.Dir = repoCachePath
			if output, err := cmd.CombinedOutput(); err != nil {
				if i >= 4 {
					return fmt.Errorf("failed to update repository: %w, output: %s", err, string(output))
				}
				e.LogGroup.Verbose("SVN: Failed to update repository: %v, retrying...", err)
				time.Sleep(50 * time.Millisecond)
				continue
			}
			break
		}
		// Write the marker after a successful update.
		if err := helper.Write(lastUpdatedPath); err != nil {
			return err
		}
	}

	e.LogGroup.Debug("SVN: %s checkout successful: %s", e.DestPath, e.Tag)
	if err := s.lookForCurseSlug(); err != nil {
		return err
	}
	return nil
}
