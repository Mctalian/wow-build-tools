package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/McTalian/wow-build-tools/internal/external"
	"github.com/McTalian/wow-build-tools/internal/logger"
)

type PkgMetaManualChangelog struct {
	Filename   string `yaml:"filename"`
	MarkupType string `yaml:"markup-type"`
}

// PkgMeta represents the structure of the .pkgmeta file
type PkgMeta struct {
	PackageAs            string                            `yaml:"package-as"`
	Externals            map[string]external.ExternalEntry `yaml:"externals"`
	MoveFolders          map[string]string                 `yaml:"move-folders"`
	Ignore               []string                          `yaml:"ignore"`
	EnableNoLibCreation  bool                              `yaml:"enable-nolib-creation"`
	RequiredDependencies []string                          `yaml:"required-dependencies"`
	EmbeddedLibraries    []string                          `yaml:"embedded-libraries"`
	OptionalDependencies []string                          `yaml:"optional-dependencies"`
	ToolsUsed            []string                          `yaml:"tools-used"`
	ManualChangelog      PkgMetaManualChangelog            `yaml:"manual-changelog"`
	ChangelogTitle       string                            `yaml:"changelog-title"`
}

type PkgMetaFileNotFound struct{}

func (e PkgMetaFileNotFound) Error() string {
	return "no .pkgmeta or pkgmeta.yml file found"
}

func StringList(s []string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	str := ""
	for _, item := range s {
		str += fmt.Sprintf("\n%s- %s", indent, item)
	}
	return str
}

func (i *PkgMeta) RequiredDependenciesString(spaces int) string {
	return StringList(i.RequiredDependencies, spaces)
}

func (i *PkgMeta) IgnoreString(spaces int) string {
	return StringList(i.Ignore, spaces)
}

func (p *PkgMeta) String() string {
	str := fmt.Sprintf("Package-As: %s\n", p.PackageAs)
	str += fmt.Sprintf("Enable No Lib Creation: %t\n", p.EnableNoLibCreation)
	str += fmt.Sprintf("Required Dependencies: %v\n", p.RequiredDependenciesString(4))
	str += "Move Folders:\n"
	for src, dest := range p.MoveFolders {
		str += fmt.Sprintf("%s- %s -> %s\n", strings.Repeat(" ", 4), src, dest)
	}
	str += fmt.Sprintf("Ignore: %s\n", p.IgnoreString(4))
	str += "Externals:\n"
	for path, entry := range p.Externals {
		str += fmt.Sprintf("- %s: %s\n", path, entry.String(4))
	}
	return str
}

func (p *PkgMeta) FetchExternals(packageDir string) error {
	externalLogger := logger.GetSubLog("EXT")
	externalLogger.Debug("Fetching external dependencies")

	var checkoutWg sync.WaitGroup
	checkoutErrChan := make(chan error, len(p.Externals))

	numOrigEmbeds := len(p.EmbeddedLibraries)
	missingSlugEncountered := false
	start := time.Now()
	for path, entry := range p.Externals {
		// Capture the current loop variables.
		currentEntry := entry
		currentPath := path

		currentEntry.LogGroup = logger.NewLogGroup(fmt.Sprintf("🌐 External %s", path))

		var ext external.Vcs
		var err error
		switch currentEntry.EType {
		case external.Git:
			checkoutWg.Add(1)
			currentEntry.LogGroup.Info("📥 Processing external for %s", currentPath)
			ext, err = external.NewGitExternal(&currentEntry)
			if err != nil {
				currentEntry.LogGroup.Error("Failed to create git external: %v", err)
				currentEntry.LogGroup.Flush()
				checkoutErrChan <- fmt.Errorf("failed to create git external: %w", err)
				continue
			}
		case external.Svn:
			checkoutWg.Add(1)
			currentEntry.LogGroup.Info("📥 Processing external for %s", currentPath)
			ext, err = external.NewSvnExternal(&currentEntry)
			if err != nil {
				currentEntry.LogGroup.Error("Failed to create svn external: %v", err)
				currentEntry.LogGroup.Flush()
				checkoutErrChan <- fmt.Errorf("failed to create svn external: %w", err)
				continue
			}
		case external.Hg:
			externalLogger.Warn("Mercurial externals are not supported yet for %s", currentPath)
			continue
		default:
			externalLogger.Warn("Unknown external type %s for %s", currentEntry.EType.ToString(), currentPath)
			continue
		}
		if ext == nil {
			externalLogger.Warn("Failed to create external for %s", currentPath)
			continue
		}

		// Pass the captured copy of currentEntry into the goroutine.
		go func(ext external.Vcs, entry external.ExternalEntry) {
			defer entry.LogGroup.Flush()
			defer checkoutWg.Done()
			if err := ext.Checkout(); err != nil {
				checkoutErrChan <- fmt.Errorf("failed to checkout external: %w", err)
				return
			}
			if err := copyExternal(&entry, packageDir); err != nil {
				checkoutErrChan <- fmt.Errorf("failed to copy external: %w", err)
				return
			}

			if entry.CurseSlug != "" {
				p.EmbeddedLibraries = append(p.EmbeddedLibraries, entry.CurseSlug)
			} else if entry.CurseSlug == "" {
				if numOrigEmbeds == 0 {
					entry.LogGroup.Warn("No CurseSlug found for %s and you have no embedded-libraries specified in your pkgmeta file.", entry.DestPath)
					entry.LogGroup.Warn("Please add the CurseSlug to the external entry or add it to the embedded-libraries list to support the hard work of the author(s).")
				} else {
					entry.LogGroup.Warn("No CurseSlug found for %s", entry.DestPath)
					missingSlugEncountered = true
				}
			}
		}(ext, currentEntry)
	}

	checkoutWg.Wait()
	close(checkoutErrChan)

	// Collect errors
	for err := range checkoutErrChan {
		if err != nil {
			return fmt.Errorf("error fetching externals: %v", err)
		}
	}

	externalLogger.Timing("All External dependencies fetched in %s", time.Since(start))

	if missingSlugEncountered && len(p.EmbeddedLibraries) < len(p.Externals) {
		externalLogger.Warn("CurseSlugs could not be determined for one or more externals above and it may not be specified in your pkgmeta embedded-libraries.")
		externalLogger.Warn("Please ensure all external libraries have a CurseSlug or add them to the embedded-libraries list to support the hard work of the author(s).")
	}

	var uniqueEmbeds = make(map[string]bool)
	for _, embed := range p.EmbeddedLibraries {
		uniqueEmbeds[embed] = true
	}
	p.EmbeddedLibraries = make([]string, 0, len(uniqueEmbeds))
	for embed := range uniqueEmbeds {
		p.EmbeddedLibraries = append(p.EmbeddedLibraries, embed)
	}

	return nil
}

func (p *PkgMeta) GetNoLibDirs(pkgDir string) []string {
	noLibDirs := make([]string, 0)
	for path := range p.Externals {
		noLibDirs = append(noLibDirs, fmt.Sprintf("%s%s%s", pkgDir, string(os.PathSeparator), path))
	}
	return noLibDirs
}

// ParsePkgMeta reads and parses the .pkgmeta or pkgmeta.yml file
func parsePkgMeta(filename string) (*PkgMeta, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var pkgMeta PkgMeta
	err = yaml.Unmarshal(data, &pkgMeta)
	if err != nil {
		return nil, err
	}

	for path, entry := range pkgMeta.Externals {
		entry.DestPath = path
		pkgMeta.Externals[path] = entry
	}

	if !slices.Contains(pkgMeta.ToolsUsed, "wow-build-tools") {
		pkgMeta.ToolsUsed = append(pkgMeta.ToolsUsed, "wow-build-tools")
	}

	return &pkgMeta, nil
}

type ParseArgs struct {
	PkgmetaFile string
	PkgDir      string
	LogGroup    *logger.LogGroup
}

func Parse(args *ParseArgs) (*PkgMeta, error) {
	var pkgDir string
	if args.PkgDir != "" {
		pkgDir = args.PkgDir
	} else {
		pkgDir, _ = os.Getwd()
	}
	var pkgmetaFile string
	if args.PkgmetaFile != "" {
		pkgmetaFile = args.PkgmetaFile
		if !strings.Contains(pkgmetaFile, pkgDir) {
			pkgmetaFile = filepath.Join(pkgDir, pkgmetaFile)
		}
		if args.LogGroup != nil {
			args.LogGroup.Verbose("Using pkgmeta file: %s", pkgmetaFile)
		} else {
			logger.Verbose("Using pkgmeta file: %s", pkgmetaFile)
		}
	} else {
		ymlFile := filepath.Join(pkgDir, "pkgmeta.yml")
		pkgFile := filepath.Join(pkgDir, ".pkgmeta")
		subPath := strings.Split(pkgDir, "/")[len(strings.Split(pkgDir, "/"))-1]
		if args.LogGroup != nil {
			args.LogGroup.Verbose("Looking for pkgmeta files in %s", subPath)
		} else {
			logger.Verbose("Looking for pkgmeta files in %s", subPath)
		}
		if _, err := os.Stat(ymlFile); err == nil {
			pkgmetaFile = ymlFile
		} else if _, err := os.Stat(pkgFile); err == nil {
			pkgmetaFile = pkgFile
		} else {
			return nil, &PkgMetaFileNotFound{}
		}
	}

	return parsePkgMeta(pkgmetaFile)
}
