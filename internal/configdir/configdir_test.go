package configdir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/McTalian/wow-build-tools/internal/github"
)

func TestGet(t *testing.T) {
	externalsCacheDir = "" // Reset cacheDir for testing
	var expectedDir string
	if github.IsGitHubAction() {
		expectedDir = filepath.Join(os.Getenv("RUNNER_TEMP"), externalsPath)
	} else {
		expectedDir = filepath.Join(os.Getenv("HOME"), externalsPath)
	}

	dir, err := GetExternalsCache()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if dir != expectedDir {
		t.Errorf("expected %s, got %s", expectedDir, dir)
	}
}

func TestCreate(t *testing.T) {
	externalsCacheDir = "" // Reset cacheDir for testing
	var expectedDir string
	if github.IsGitHubAction() {
		expectedDir = filepath.Join(os.Getenv("RUNNER_TEMP"), externalsPath)
	} else {
		expectedDir = filepath.Join(os.Getenv("HOME"), externalsPath)
	}

	dir, err := CreateExternalsCache()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if dir != expectedDir {
		t.Errorf("expected %s, got %s", expectedDir, dir)
	}

	// Check if directory was created
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("expected directory %s to be created", expectedDir)
	}

	// Clean up
	os.RemoveAll(expectedDir)
}

func TestDelete(t *testing.T) {
	externalsCacheDir = "" // Reset cacheDir for testing
	var expectedDir string
	if github.IsGitHubAction() {
		expectedDir = filepath.Join(os.Getenv("RUNNER_TEMP"), externalsPath)
	} else {
		expectedDir = filepath.Join(os.Getenv("HOME"), externalsPath)
	}

	// Create directory for testing
	if err := os.MkdirAll(expectedDir, os.ModePerm); err != nil {
		t.Fatalf("failed to create directory for testing: %v", err)
	}

	if err := DeleteExternalsCache(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check if directory was deleted
	if _, err := os.Stat(expectedDir); !os.IsNotExist(err) {
		t.Errorf("expected directory %s to be deleted", expectedDir)
	}
}
