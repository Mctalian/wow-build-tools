package zipper

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/tokens"
)

type Zipper struct {
	pkgDir          string
	releaseDir      string
	topDir          string
	logGroup        *logger.LogGroup
	unixLineEndings bool
}

func (z *Zipper) Complete() {
	z.logGroup.Flush(true)
}

func (z *Zipper) ZipFiles(srcPath string, destPath string, noLibArgs ...[]string) error {
	z.logGroup.Info("📦 Creating %s", destPath)
	dirsToExclude := []string{}
	noLibStripPaths := []string{}
	if len(noLibArgs) > 0 {
		dirsToExclude = noLibArgs[0]
	}
	if len(noLibArgs) > 1 {
		noLibStripPaths = noLibArgs[1]
	}

	// Delete the destination file if it already exists
	if _, err := os.Stat(destPath); err == nil {
		z.logGroup.Verbose("Removing existing file: %s", destPath)
		err := os.Remove(destPath)
		if err != nil {
			return err
		}
	}

	// Create the zip file
	zipFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// Initialize the zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk the source directory
	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the directory should be excluded
		if info.IsDir() {
			for _, dir := range dirsToExclude {
				if path == dir {
					return filepath.SkipDir
				}
			}
		}

		// Create a header based on the file info
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Use a relative path so that the files are not stored with full system paths
		relPath, err := filepath.Rel(filepath.Dir(srcPath), path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// If it's a directory, we need to end the header name with a "/"
		if info.IsDir() {
			header.Name += "/"
		} else {
			// Use deflate compression for files
			header.Method = zip.Deflate
		}

		// Create writer for this file/directory header
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// For directories, no need to copy file content
		if info.IsDir() {
			return nil
		}

		// Check file size and warn if it seems too large
		if info.Size() > 1000000 {
			abbrevSize := float64(info.Size()) / 1000000.0
			trimmedPath := strings.ReplaceAll(path, z.pkgDir, z.topDir)
			trimmedDestPath := strings.TrimPrefix(destPath, z.releaseDir+string(os.PathSeparator))
			z.logGroup.Warn("%s: %s is large (%f MB), consider adding it to ignores", trimmedDestPath, trimmedPath, abbrevSize)
		}

		if len(noLibStripPaths) > 0 {
			noLibStripVariants := tokens.NoLibStrip.GetVariants()
			for _, noLibStripPath := range noLibStripPaths {
				if path == noLibStripPath {
					// TODO, read the file and comment out the lib strip line
					contents, err := os.ReadFile(path)
					if err != nil {
						return err
					}

					contentsStr := string(contents)
					// Comment out the lib strip line
					var lineEnding string
					if z.unixLineEndings {
						lineEnding = "\n"
					} else {
						lineEnding = "\r\n"
					}
					contentsLines := strings.Split(contentsStr, lineEnding)
					var lineStart = -1
					var newContents []string
					for i, line := range contentsLines {
						if strings.Contains(line, fmt.Sprintf("@%s@", noLibStripVariants.Standard)) {
							lineStart = i
							continue
						}
						if strings.Contains(line, fmt.Sprintf("@%s@", noLibStripVariants.StandardEnd)) {
							lineStart = -1
							continue
						}
						if lineStart != -1 && i > lineStart {
							continue
						}
						newContents = append(newContents, line)
					}
					contentsStr = strings.Join(newContents, lineEnding)

					_, err = writer.Write([]byte(contentsStr))
					if err != nil {
						return err
					}
					return nil
				}
			}
		}

		// Open the file to be added
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Copy the file content into the zip writer
		_, err = io.Copy(writer, file)
		return err
	})
}

func NewZipper(pkgDir string, releaseDir string, topDir string, unixLineEndings bool) *Zipper {
	logGroup := logger.NewLogGroup("💼 Creating Zip File(s)")
	return &Zipper{
		pkgDir:          pkgDir,
		releaseDir:      releaseDir,
		topDir:          topDir,
		logGroup:        logGroup,
		unixLineEndings: unixLineEndings,
	}
}
