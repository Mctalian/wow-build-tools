package changelog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/McTalian/wow-build-tools/internal/repo"
)

func TestVerifyManualChangelog(t *testing.T) {
	tests := []struct {
		name                string
		preExistingFilePath string
		markupType          MarkupType
		setup               func() (string, string)
		expectedError       error
	}{
		{
			name:                "NoPreExistingFilePath",
			preExistingFilePath: "",
			markupType:          MarkdownMT,
			setup:               func() (string, string) { return "", "" },
			expectedError:       ErrManualChangelogNotFound,
		},
		{
			name:                "FileNotFoundInTopDir",
			preExistingFilePath: "CHANGELOG.md",
			markupType:          MarkdownMT,
			setup: func() (string, string) {
				topDir := t.TempDir()
				pkgDir := t.TempDir()
				return topDir, pkgDir
			},
			expectedError: ErrManualChangelogNotFound,
		},
		{
			name:                "FileFoundInTopDir",
			preExistingFilePath: "CHANGELOG.md",
			markupType:          MarkdownMT,
			setup: func() (string, string) {
				topDir := t.TempDir()
				pkgDir := t.TempDir()
				filePath := filepath.Join(topDir, "CHANGELOG.md")
				os.WriteFile(filePath, []byte("changelog content"), 0644)
				return topDir, pkgDir
			},
			expectedError: nil,
		},
		{
			name:                "FileFoundInPkgDir",
			preExistingFilePath: "CHANGELOG.md",
			markupType:          MarkdownMT,
			setup: func() (string, string) {
				topDir := t.TempDir()
				pkgDir := t.TempDir()
				topDirFilePath := filepath.Join(topDir, "CHANGELOG.md")
				os.WriteFile(topDirFilePath, []byte("topdir changelog content"), 0644)
				filePath := filepath.Join(pkgDir, "CHANGELOG.md")
				os.WriteFile(filePath, []byte("changelog content"), 0644)
				return topDir, pkgDir
			},
			expectedError: nil,
		},
		{
			name:                "InvalidMarkupType",
			preExistingFilePath: "CHANGELOG.md",
			markupType:          "invalid",
			setup: func() (string, string) {
				topDir := t.TempDir()
				pkgDir := t.TempDir()
				filePath := filepath.Join(topDir, "CHANGELOG.md")
				os.WriteFile(filePath, []byte("changelog content"), 0644)
				return topDir, pkgDir
			},
			expectedError: ErrInvalidMarkupType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topDir, pkgDir := tt.setup()
			changelog := &Changelog{
				topDir:              topDir,
				pkgDir:              pkgDir,
				PreExistingFilePath: tt.preExistingFilePath,
				MarkupType:          tt.markupType,
			}
			err := changelog.verifyManualChangelog()
			if err != tt.expectedError {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}
		})
	}
}

func TestGetChangelog(t *testing.T) {
	tests := []struct {
		name                string
		preExistingFilePath string
		markupType          MarkupType
		generateChangelog   bool
		setup               func() (string, string, repo.VcsRepo)
		expectedError       error
	}{
		{
			name:                "ManualChangelogExistsInPkgDir",
			preExistingFilePath: "{pkgDir}/CHANGELOG.md",
			markupType:          MarkdownMT,
			generateChangelog:   false,
			setup: func() (string, string, repo.VcsRepo) {
				topDir := t.TempDir()
				pkgDir := t.TempDir()
				filePath := filepath.Join(pkgDir, "CHANGELOG.md")
				os.WriteFile(filePath, []byte("changelog content"), 0644)
				return topDir, pkgDir, nil
			},
			expectedError: nil,
		},
		{
			name:                "ManualChangelogExistsInTopDir",
			preExistingFilePath: "{topDir}/CHANGELOG.md",
			markupType:          MarkdownMT,
			generateChangelog:   false,
			setup: func() (string, string, repo.VcsRepo) {
				topDir := t.TempDir()
				pkgDir := t.TempDir()
				filePath := filepath.Join(topDir, "CHANGELOG.md")
				os.WriteFile(filePath, []byte("changelog content"), 0644)
				return topDir, pkgDir, nil
			},
			expectedError: nil,
		},
		{
			name:                "ManualChangelogNotFound",
			preExistingFilePath: "CHANGELOG.md",
			markupType:          MarkdownMT,
			generateChangelog:   false,
			setup: func() (string, string, repo.VcsRepo) {
				topDir := t.TempDir()
				pkgDir := t.TempDir()
				mockRepo := &repo.MockVcsRepo{
					GetChangelogFunc: func(title string) (string, error) {
						return "generated changelog contents", nil
					},
				}
				return topDir, pkgDir, mockRepo
			},
			expectedError: nil,
		},
		{
			name:                "GenerateChangelog",
			preExistingFilePath: "",
			markupType:          MarkdownMT,
			generateChangelog:   true,
			setup: func() (string, string, repo.VcsRepo) {
				topDir := t.TempDir()
				pkgDir := t.TempDir()
				mockRepo := &repo.MockVcsRepo{
					GetChangelogFunc: func(title string) (string, error) {
						return "generated changelog contents", nil
					},
				}
				return topDir, pkgDir, mockRepo
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topDir, pkgDir, mockRepo := tt.setup()
			filePath := strings.ReplaceAll(tt.preExistingFilePath, "{topDir}", topDir)
			filePath = strings.ReplaceAll(filePath, "{pkgDir}", pkgDir)
			changelog := &Changelog{
				topDir:              topDir,
				pkgDir:              pkgDir,
				PreExistingFilePath: filePath,
				MarkupType:          tt.markupType,
				generateChangelog:   tt.generateChangelog,
				repo:                mockRepo,
			}
			err := changelog.GetChangelog()
			if err != tt.expectedError {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}
			if tt.generateChangelog {
				expectedPath := filepath.Join(pkgDir, "CHANGELOG.md")
				if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
					t.Errorf("expected changelog file to be generated at %s", expectedPath)
				}
			}
		})
	}
}
