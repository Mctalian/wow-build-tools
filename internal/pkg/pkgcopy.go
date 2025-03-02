package pkg

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/McTalian/wow-build-tools/internal/external"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/repo"
)

type PkgCopy struct {
	TopDir     string
	PackageDir string
	Ignore     []string
	Repo       repo.VcsRepo
}

var copyLogger = logger.GetSubLog("CPY")

func clearDestDir(path string) error {
	// Ensure destination exists and remove any existing directory
	copyLogger.Verbose("Rmdir %s", path)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}
	return nil
}

func createDestDir(path string) error {
	// Copy from cache to destination
	copyLogger.Verbose("Mkdir %s", path)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	return nil
}

func copyFromCacheToPackageDir(destPath string, repoCachePath string, logGroup *logger.LogGroup) error {
	if err := clearDestDir(destPath); err != nil {
		return err
	}

	if err := createDestDir(destPath); err != nil {
		return err
	}

	if _, err := os.Stat(repoCachePath); os.IsNotExist(err) {
		return fmt.Errorf("cache path does not exist (%s): %s", destPath, repoCachePath)
	}

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return fmt.Errorf("destination path already exists: %s", destPath)
	}

	ignores, err := tryParsePkgMetaIgnores(repoCachePath, logGroup)
	if err != nil {
		return err
	}
	vR := repo.BaseVcsRepo{}
	pkgCopy := NewPkgCopy(repoCachePath, destPath, ignores, &vR)
	err = pkgCopy.CopyToPackageDir(logGroup)
	if err != nil {
		return err
	}

	repoCacheSubpath := filepath.Base(repoCachePath)
	pathSegments := strings.Split(destPath, string(filepath.Separator))
	var subDir string
	if len(pathSegments) < 2 {
		subDir = destPath
	} else {
		subDir = filepath.Join(pathSegments[len(pathSegments)-2], pathSegments[len(pathSegments)-1])
	}
	logGroup.Verbose("Copied %s to %s", repoCacheSubpath, subDir)

	return nil
}

func copyExternal(e *external.ExternalEntry, packageDir string) error {
	repoCachePath := e.RepoCacheDir

	destPath := filepath.Join(packageDir, e.DestPath)

	if e.Path != "" && !strings.Contains(e.URL, e.Path) {
		repoCachePath = filepath.Join(repoCachePath, e.Path)
		if e.EType == external.Svn {
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

	return copyFromCacheToPackageDir(destPath, repoCachePath, e.LogGroup)
}

func CopySingleFile(path string, destPath string, logGroup *logger.LogGroup, args ...string) error {
	prettyPath := path
	if len(args) > 0 {
		prettyPath = args[0]
	}

	if logGroup != nil {
		logGroup.Info("📑 Copying file %s", prettyPath)
	}

	// Open source file in read-only mode
	srcFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening file %s: %v", path, err)
	}
	defer srcFile.Close()

	// Get file info for mode preservation
	srcFileInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info for %s: %v", path, err)
	}

	// Create destination file with the same permissions as the source
	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcFileInfo.Mode())
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", destPath, err)
	}
	defer destFile.Close()

	// Copy file contents
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("error copying file %s to %s: %v", path, destPath, err)
	}

	// Ensure data is written to disk
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("error syncing file %s: %v", destPath, err)
	}

	return nil
}

func (p *PkgCopy) CopyToPackageDir(logGroup *logger.LogGroup) error {
	topDir := p.TopDir
	packageDir := p.PackageDir
	vR := p.Repo
	ignores := p.Ignore

	err := filepath.WalkDir(topDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		prettyPath := strings.TrimPrefix(path, topDir+string(os.PathSeparator))

		// Skip hidden files or directories based on their base name.
		if strings.HasPrefix(d.Name(), ".") && d.Name() != "." && d.Name() != ".." {
			logGroup.Debug("⛔ Ignoring hidden file or directory %s", path)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Compute the path relative to the topDir so that we can rebuild the same structure.
		relPath, err := filepath.Rel(topDir, path)
		if err != nil {
			return err
		}

		// Check against ignore patterns.
		for _, ignore := range ignores {
			pattern := ignore
			// For directories, trim the "/*" if present.
			if d.IsDir() && strings.Contains(ignore, "/*") {
				pattern = strings.TrimSuffix(ignore, "/*")
			}
			matched, err := filepath.Match(pattern, relPath)
			if err != nil {
				return fmt.Errorf("error matching ignore pattern: %v", err)
			}
			if matched {
				logGroup.Debug("⛔ Ignoring %s", prettyPath)
				// If it's a directory, skip the whole subtree.
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			matched, err = filepath.Match(pattern, d.Name())
			if err != nil {
				return fmt.Errorf("error matching ignore pattern: %v", err)
			}
			if matched {
				logGroup.Debug("⛔ Ignoring %s", prettyPath)
				// If it's a directory, skip the whole subtree.
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Check the repo's ignore logic.
		if vR.IsIgnored(path, d.IsDir()) {
			logGroup.Debug("⛔ Ignoring %s", prettyPath)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Build the destination path.
		destPath := filepath.Join(packageDir, relPath)

		// If it's a directory, create it.
		if d.IsDir() {
			logGroup.Info("🗂️  Creating directory %s", destPath)
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("error creating directory %s: %v", destPath, err)
			}
		} else {
			err = CopySingleFile(path, destPath, logGroup, prettyPath)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error copying package directory: %v", err)
	}
	return nil
}

func tryParsePkgMetaIgnores(pkgDir string, logGroup *logger.LogGroup) ([]string, error) {
	args := ParseArgs{
		PkgDir:   pkgDir,
		LogGroup: logGroup,
	}
	pkgMeta, err := Parse(&args)
	if err != nil {
		_, ok := err.(*PkgMetaFileNotFound)
		if ok {
			shortenedPath := strings.Split(pkgDir, string(os.PathSeparator))[len(strings.Split(pkgDir, string(os.PathSeparator)))-1]
			logGroup.Verbose("No .pkgmeta file found in %s", shortenedPath)
			return []string{}, nil
		}
		return nil, err
	}
	return pkgMeta.Ignore, nil
}

func NewPkgCopy(topDir, packageDir string, ignores []string, repo repo.VcsRepo) *PkgCopy {
	return &PkgCopy{
		TopDir:     topDir,
		PackageDir: packageDir,
		Ignore:     ignores,
		Repo:       repo,
	}
}
