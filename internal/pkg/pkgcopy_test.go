package pkg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/McTalian/wow-build-tools/internal/external"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/repo"
	"github.com/stretchr/testify/require"
)

func TestClearDestDir(t *testing.T) {
	testDir := "testdir"
	err := os.Mkdir(testDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	err = clearDestDir(testDir)
	require.NoError(t, err)

	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Fatalf("Expected directory to be removed")
	}
}

func TestCreateDestDir(t *testing.T) {
	testDir := "testdir"
	defer os.RemoveAll(testDir)

	err := createDestDir(testDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Fatalf("Expected directory to be created")
	}
}

func TestCopyFromCacheToPackageDir(t *testing.T) {
	destDir := "destdir"
	cacheDir := "cachedir"
	defer os.RemoveAll(destDir)
	defer os.RemoveAll(cacheDir)

	err := os.Mkdir(cacheDir, 0755)
	require.NoError(t, err)
	logGroup := logger.NewLogGroup("test")

	err = copyFromCacheToPackageDir(destDir, cacheDir, logGroup)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		t.Fatalf("Expected destination directory to be created")
	}
}

func TestCopyExternal(t *testing.T) {
	packageDir := "packagedir"
	cacheDir := "cachedir"
	defer os.RemoveAll(packageDir)
	defer os.RemoveAll(cacheDir)

	err := os.Mkdir(cacheDir, 0755)
	require.NoError(t, err)

	logGroup := logger.NewLogGroup("test")
	externalEntry := &external.ExternalEntry{
		RepoCacheDir: cacheDir,
		DestPath:     "destpath",
		LogGroup:     logGroup,
	}

	err = copyExternal(externalEntry, packageDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	destPath := filepath.Join(packageDir, externalEntry.DestPath)
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Fatalf("Expected destination path to be created")
	}
}

func TestCopyToPackageDir(t *testing.T) {
	topDir := "topdir"
	packageDir := "packagedir"
	defer os.RemoveAll(topDir)
	defer os.RemoveAll(packageDir)

	err := os.Mkdir(topDir, 0755)
	require.NoError(t, err)

	logGroup := logger.NewLogGroup("test")
	vR := repo.BaseVcsRepo{}
	pkgCopy := NewPkgCopy(topDir, packageDir, []string{}, &vR)

	err = pkgCopy.CopyToPackageDir(logGroup)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		t.Fatalf("Expected package directory to be created")
	}
}
