package external

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/McTalian/wow-build-tools/internal/logger"
)

// LastUpdatedHelper centralizes cache marker file logic.
// It is generic enough for all VCS implementations.
type LastUpdatedHelper struct {
	CacheDir   string // The directory where the marker is stored.
	FilePrefix string // The prefix used for the marker filename.
	Force      bool   // If true, the marker will be deleted to force an update.
	Log        *logger.LogGroup
}

// NewLastUpdatedHelper returns an instance configured for the given cache directory and marker prefix.
func NewLastUpdatedHelper(cacheDir string, prefix string, forceExternals bool, log *logger.LogGroup) *LastUpdatedHelper {
	return &LastUpdatedHelper{
		CacheDir:   cacheDir,
		FilePrefix: prefix,
		Force:      forceExternals,
		Log:        log,
	}
}

// FilePath returns the full path to the lastUpdated marker file.
// If tag is non-empty, it is appended to the filename.
func (l *LastUpdatedHelper) FilePath(tag string) string {
	filename := l.FilePrefix
	if tag != "" {
		filename += "_" + tag
	}
	return filepath.Join(l.CacheDir, filename)
}

// Delete removes the marker file (ignoring if it doesn't exist).
func (l *LastUpdatedHelper) Delete(filePath string) error {
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("%s: failed to remove lastUpdated file: %v", l.FilePrefix, err)
	}
	return nil
}

// Write writes the current time (RFC3339) to the marker file.
func (l *LastUpdatedHelper) Write(filePath string) error {
	err := os.WriteFile(filePath, []byte(time.Now().Format(time.RFC3339)), 0644)
	if err != nil {
		return fmt.Errorf("%s: failed to write lastUpdated file: %v", l.FilePrefix, err)
	}
	return nil
}

// IsStale checks the marker file at filePath. It returns false if the file exists and the timestamp
// is within the given duration; otherwise it returns true. In case of errors reading or parsing the file,
// it logs an error, deletes the marker, and returns true.
func (l *LastUpdatedHelper) IsStale(filePath string, validDuration time.Duration) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		// If the file doesn't exist, the cache is considered stale.
		if os.IsNotExist(err) {
			return true, nil
		}
		l.Log.Error("%s: failed to read lastUpdated file: %v", l.FilePrefix, err)
		if derr := l.Delete(filePath); derr != nil {
			return true, derr
		}
		return true, nil
	}

	t, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		l.Log.Error("%s: failed to parse lastUpdated time: %v", l.FilePrefix, err)
		if derr := l.Delete(filePath); derr != nil {
			return true, derr
		}
		return true, nil
	}

	if time.Since(t) > validDuration {
		l.Log.Verbose("%s: Cache is stale, updating...", l.FilePrefix)
		if derr := l.Delete(filePath); derr != nil {
			return true, derr
		}
		return true, nil
	}

	l.Log.Verbose("%s: Cache is up-to-date", l.FilePrefix)
	return false, nil
}
