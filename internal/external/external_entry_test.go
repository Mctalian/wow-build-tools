package external

import (
	"os"
	"testing"

	"github.com/McTalian/wow-build-tools/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Valid URL",
			input:   "url: https://github.com/user/repo",
			wantErr: false,
		},
		{
			name:    "Invalid format",
			input:   "invalid: data",
			wantErr: true,
		},
		{
			name:    "Empty URL",
			input:   "url: ",
			wantErr: true,
		},
		{
			name:    "Valid tag",
			input:   "url: https://github.com/user/repo\ntag: v1.0.0",
			wantErr: false,
		},
		{
			name:    "Valid branch",
			input:   "url: https://github.com/user/repo\nbranch: main",
			wantErr: false,
		},
		{
			name:    "Valid commit",
			input:   "url: https://github.com/user/repo\ncommit: abc123",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e ExternalEntry
			err := yaml.Unmarshal([]byte(tt.input), &e)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetRepoCachePath(t *testing.T) {
	e := ExternalEntry{
		URL: "https://github.com/user/repo",
		Tag: "v1.0.0",
	}
	var cachePath string
	var err error
	if github.IsGitHubAction() {
		cachePath = os.Getenv("RUNNER_TEMP")
	} else {
		cachePath, err = os.UserHomeDir()
		require.NoError(t, err)
	}
	expected := cachePath + "/.wow-build-tools/.cache/externals/https:__github.com_user_repo_v1.0.0"

	if got := e.GetRepoCachePath(); got != expected {
		t.Errorf("GetRepoCachePath() = %v, want %v", got, expected)
	}
}

func TestDeterminePath(t *testing.T) {
	tests := []struct {
		name     string
		entry    ExternalEntry
		expected string
	}{
		{
			name: "GitHub URL",
			entry: ExternalEntry{
				URL:   "https://github.com/user/repo/path/to/dir",
				EType: Git,
			},
			expected: "path/to/dir",
		},
		{
			name: "CurseForge URL",
			entry: ExternalEntry{
				URL:   "https://repos.curseforge.com/wow/addon/trunk/sublib",
				EType: Svn,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.entry.determinePath()
			if tt.entry.Path != tt.expected {
				t.Errorf("determinePath() = %v, want %v", tt.entry.Path, tt.expected)
			}
		})
	}
}
func TestHandleCurseUrl(t *testing.T) {
	tests := []struct {
		name     string
		entry    ExternalEntry
		expected ExternalEntry
	}{
		{
			name: "CurseForge URL with trunk",
			entry: ExternalEntry{
				URL: "https://repos.curseforge.com/wow/addon/trunk/libdir",
			},
			expected: ExternalEntry{
				URL:       "https://repos.curseforge.com/wow/addon/trunk/libdir",
				CurseSlug: "addon",
				EType:     Svn,
				Path:      "libdir",
			},
		},
		{
			name: "CurseForge URL with tag",
			entry: ExternalEntry{
				URL: "https://repos.curseforge.com/wow/addon/tags/v1.0.0",
			},
			expected: ExternalEntry{
				URL:       "https://repos.curseforge.com/wow/addon/trunk",
				CurseSlug: "addon",
				EType:     Svn,
				Tag:       "v1.0.0",
			},
		},
		{
			name: "CurseForge URL without path",
			entry: ExternalEntry{
				URL: "https://repos.curseforge.com/wow/addon/trunk",
			},
			expected: ExternalEntry{
				URL:       "https://repos.curseforge.com/wow/addon/trunk",
				CurseSlug: "addon",
				EType:     Svn,
			},
		},
		{
			name: "Non-CurseForge URL",
			entry: ExternalEntry{
				URL: "https://github.com/user/repo",
			},
			expected: ExternalEntry{
				URL: "https://github.com/user/repo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.entry.handleCurseUrl()
			assert.Equal(t, tt.expected.URL, tt.entry.URL)
			assert.Equal(t, tt.expected.CurseSlug, tt.entry.CurseSlug)
			assert.Equal(t, tt.expected.EType, tt.entry.EType)
			assert.Equal(t, tt.expected.Path, tt.entry.Path)
			assert.Equal(t, tt.expected.Tag, tt.entry.Tag)
		})
	}
}
