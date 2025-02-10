package cachedir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGet(t *testing.T) {
	cacheDir = "" // Reset cacheDir for testing
	expectedDir := filepath.Join(os.Getenv("HOME"), ".wow-build-tools", ".cache", "externals")

	dir, err := Get()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if dir != expectedDir {
		t.Errorf("expected %s, got %s", expectedDir, dir)
	}
}

func TestCreate(t *testing.T) {
	cacheDir = "" // Reset cacheDir for testing
	expectedDir := filepath.Join(os.Getenv("HOME"), ".wow-build-tools", ".cache", "externals")

	dir, err := Create()
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
	cacheDir = "" // Reset cacheDir for testing
	expectedDir := filepath.Join(os.Getenv("HOME"), ".wow-build-tools", ".cache", "externals")

	// Create directory for testing
	if err := os.MkdirAll(expectedDir, os.ModePerm); err != nil {
		t.Fatalf("failed to create directory for testing: %v", err)
	}

	if err := Delete(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check if directory was deleted
	if _, err := os.Stat(expectedDir); !os.IsNotExist(err) {
		t.Errorf("expected directory %s to be deleted", expectedDir)
	}
}
