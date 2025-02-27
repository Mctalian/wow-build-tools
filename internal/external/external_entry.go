package external

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"

	"github.com/McTalian/wow-build-tools/internal/configdir"
	"github.com/McTalian/wow-build-tools/internal/logger"
)

// ExternalEntry represents an entry in the "externals" section
type ExternalEntry struct {
	Vcs
	URL          string `yaml:"url"`
	Tag          string `yaml:"tag"`
	Branch       string `yaml:"branch"`
	CheckoutType string
	Commit       string `yaml:"commit"`
	Type         string `yaml:"type"`
	EType        VcsType
	CurseSlug    string `yaml:"curse-slug"`
	Path         string `yaml:"path"`
	DestPath     string
	LogGroup     *logger.LogGroup
	RepoCacheDir string
}

// Known URL patterns for different repo types
var repoTypePatterns = map[string]VcsType{
	"git.curseforge.com": Git,
	"git.wowace.com":     Git,
	"svn.curseforge.com": Svn,
	"svn.wowace.com":     Svn,
	"hg.curseforge.com":  Hg,
	"hg.wowace.com":      Hg,
}

// Known prefixes for CurseForge/WowAce repositories
var cursePrefixes = []string{
	"https://repos.curseforge.com/wow/",
	"https://repos.wowace.com/wow/",
}

func TypeColor(t VcsType) string {
	s := t.ToString()
	switch t {
	case Git:
		return color.GreenString(s)
	case Svn:
		return color.YellowString(s)
	case Hg:
		return color.BlueString(s)
	default:
		return s
	}
}

var urlPathSeparator = "/"
var protocolSeparator = "://"

// UnmarshalYAML allows ExternalEntry to handle both string and object forms.
func (e *ExternalEntry) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		e.URL = value.Value
	} else if value.Kind == yaml.MappingNode {
		type Alias ExternalEntry
		var alias Alias
		if err := value.Decode(&alias); err != nil {
			return err
		}
		*e = ExternalEntry(alias)
		if e.Tag != "" {
			e.CheckoutType = "tag"
		} else if e.Branch != "" {
			e.CheckoutType = "branch"
			e.Tag = e.Branch
		} else if e.Commit != "" {
			e.CheckoutType = "commit"
			e.Tag = e.Commit
		}
	} else {
		return fmt.Errorf("invalid external entry format")
	}

	if e.URL == "" {
		return fmt.Errorf("URL is required")
	}

	e.EType = ToVcsType(e.Type)

	// Detect repo type based on known patterns
	for key, repoType := range repoTypePatterns {
		if strings.Contains(e.URL, key) {
			e.EType = repoType
			switch e.EType {
			case Git:
				e.URL = strings.TrimSuffix(e.URL, "/mainline.git")
				e.URL = strings.Split(e.URL, protocolSeparator)[1]
				e.URL = strings.Replace(e.URL, "git", "https://repos", 1)
			case Svn:
				e.URL = strings.Replace(e.URL, "/mainline", "", 1)
				e.URL = strings.Split(e.URL, protocolSeparator)[1]
				e.URL = strings.Replace(e.URL, "svn", "https://repos", 1)
			case Hg:
				e.URL = strings.TrimSuffix(e.URL, "/mainline")
				e.URL = strings.Split(e.URL, protocolSeparator)[1]
				e.URL = strings.Replace(e.URL, "hg", "https://repos", 1)
			default:
				return fmt.Errorf("unknown repo type: %s", e.EType.ToString())
			}
		}
	}

	// If no repo type is detected and URL starts with "svn:", assume it's SVN
	if e.EType == Unknown && strings.HasPrefix(e.URL, "svn:") {
		e.EType = Svn
	}

	// Extract CurseSlug if applicable
	e.handleCurseUrl()

	// Default to Git if no type was determined
	if e.EType == Unknown {
		e.EType = Git
	}

	e.determinePath()

	e.RepoCacheDir = e.GetRepoCachePath()

	return nil
}

// Extract CurseSlug from URL
func (e *ExternalEntry) handleCurseUrl() {
	for _, prefix := range cursePrefixes {
		if strings.HasPrefix(e.URL, prefix) {
			// Remove the known prefix
			remainingPath := strings.TrimPrefix(e.URL, prefix)

			// Split by urlPathSeparator and take the first segment as the slug
			parts := strings.SplitN(remainingPath, urlPathSeparator, 2)
			e.CurseSlug = parts[0]

			// If there's no additional path after the slug, return early
			if len(parts) < 2 {
				return
			}

			// Check for SVN-specific paths (trunk/tags)
			svnPathParts := strings.SplitN(parts[1], urlPathSeparator, 2)
			svnRoot := svnPathParts[0] // "trunk" or "tags/<tag>"

			if svnRoot == "trunk" {
				e.EType = Svn
				if len(svnPathParts) > 1 {
					e.Path = svnPathParts[1] // Gets "path/to/addon"
				}
			} else if svnRoot == "tags" {
				e.EType = Svn

				// Extract the tag name
				if len(svnPathParts) > 1 {
					e.Tag = svnPathParts[1] // Gets "<tag>"
				}

				// Reconstruct the SVN trunk path (removing /tags/X)
				e.URL = strings.Replace(e.URL, "/tags/"+e.Tag, "/trunk", 1)
			}
			return
		}
	}
}

func (e *ExternalEntry) determinePath() {
	if e.EType == Git {
		if e.CurseSlug != "" && strings.Contains(e.URL, e.CurseSlug) {
			// Remove the known prefixes
			for _, prefix := range cursePrefixes {
				e.Path = strings.TrimPrefix(e.URL, prefix)
				if e.Path != e.URL {
					break
				}
			}
		} else if strings.Contains(e.URL, "github.com") {
			// Remove the known prefixes
			UriNoProtocol := strings.SplitN(e.URL, protocolSeparator, 2)[1]
			segments := strings.SplitN(UriNoProtocol, urlPathSeparator, 4)
			if len(segments) < 4 {
				return
			}
			// owner := segments[1]
			// repo := segments[2]
			e.Path = segments[3]
			e.URL = strings.TrimSuffix(e.URL, fmt.Sprintf("%s%s", urlPathSeparator, e.Path))
		}
	}
}

func (e *ExternalEntry) GetRepoCachePath() string {
	cacheDir, _ := configdir.GetExternalsCache()

	safeName := strings.ReplaceAll(e.URL+"_"+e.Tag, urlPathSeparator, "_")
	return filepath.Join(cacheDir, safeName)
}

func (e *ExternalEntry) String(spaces int) string {
	indent := strings.Repeat(" ", spaces)
	str := fmt.Sprintf("\n%sURL=%s\n%sType=%s", indent, e.URL, indent, TypeColor(e.EType))
	if e.Tag != "" {
		str += fmt.Sprintf("\n%sTag=%s", indent, e.Tag)
	}
	if e.CheckoutType != "" {
		str += fmt.Sprintf("\n%sCheckoutType=%s", indent, e.CheckoutType)
	}
	if e.CurseSlug != "" {
		str += fmt.Sprintf("\n%sCurseSlug=%s", indent, e.CurseSlug)
	}
	if e.Path != "" {
		str += fmt.Sprintf("\n%sPath=%s", indent, e.Path)
	}
	return str
}
