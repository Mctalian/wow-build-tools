package injector

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/repo"
	"github.com/McTalian/wow-build-tools/internal/tokens"
	"github.com/stretchr/testify/assert"
)

func TestNewInjector(t *testing.T) {
	tests := []struct {
		name            string
		simpleTokens    tokens.SimpleTokenMap
		buildTypeTokens tokens.BuildTypeTokenMap
		expectError     bool
	}{
		{
			name:            "Empty simple tokens",
			simpleTokens:    tokens.SimpleTokenMap{},
			buildTypeTokens: tokens.BuildTypeTokenMap{},
			expectError:     true,
		},
		{
			name: "Invalid simple token",
			simpleTokens: tokens.SimpleTokenMap{
				"invalid token": "value",
			},
			buildTypeTokens: tokens.BuildTypeTokenMap{},
			expectError:     true,
		},
		{
			name: "Valid simple tokens and build type tokens",
			simpleTokens: tokens.SimpleTokenMap{
				tokens.BuildDate: "value1",
			},
			buildTypeTokens: tokens.BuildTypeTokenMap{
				tokens.Alpha: true,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vcsRepo := &repo.MockVcsRepo{}
			fmt.Printf("%d", len(tt.simpleTokens))
			injector, err := NewInjector(tt.simpleTokens, vcsRepo, "/some/path", tt.buildTypeTokens)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, injector)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, injector)
				assert.Equal(t, len(tt.simpleTokens), len(injector.simpleTokens))
				assert.Equal(t, len(tt.buildTypeTokens), len(injector.buildTypeTokens))
			}
		})
	}
}

func TestInjector_FindAndReplaceInFile(t *testing.T) {
	f, err := os.CreateTemp(".", "file*.txt")
	name := f.Name()
	filePath := filepath.Join(".", name)
	assert.NoError(t, err)
	f.WriteString("@build-date@")
	assert.NoError(t, f.Close())
	tests := []struct {
		name        string
		filePath    string
		expectError bool
	}{
		{
			name:        "Valid file path",
			filePath:    filePath,
			expectError: false,
		},
		{
			name:        "Invalid file path",
			filePath:    "some/path/file",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vcsRepo := &repo.MockVcsRepo{}
			injector, err := NewInjector(tokens.SimpleTokenMap{
				tokens.BuildDate: "value1",
			}, vcsRepo, ".", tokens.BuildTypeTokenMap{})
			assert.NoError(t, err)

			injector.logGroup = logger.NewLogGroup("💉 Injecting tokens into package directory")

			err = injector.findAndReplaceInFile(tt.filePath)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.FileExists(t, tt.filePath)
				contents, err := os.ReadFile(tt.filePath)
				assert.NoError(t, err)
				assert.Equal(t, "value1", string(contents))
			}
		})
	}
	os.Remove(filePath)
}

func TestInjector_FindAndReplaceInFile_BuildTypeTokens(t *testing.T) {
	tests := []struct {
		name            string
		unixLineEndings bool
		extension       string
		contents        string
		bTTM            tokens.BuildTypeTokenMap
		expected        string
	}{
		{
			name:            "Alpha build type lua",
			extension:       ".lua",
			unixLineEndings: true,
			contents:        "--@alpha@\ntest\n--@end-alpha@\n\n--@beta@\ntest\n--@end-beta@\n\n--[===[@non-alpha@\ntest\n--@end-non-alpha@]===]",
			bTTM: tokens.BuildTypeTokenMap{
				tokens.Alpha:         true,
				tokens.Beta:          false,
				tokens.Classic:       false,
				tokens.Retail:        false,
				tokens.VersionRetail: false,
				tokens.VersionBcc:    false,
				tokens.VersionWrath:  false,
				tokens.VersionCata:   false,
			},
			expected: "--@alpha@\ntest\n--@end-alpha@\n\n--[===[@beta@\ntest\n--@end-beta@]===]\n\n--[===[@non-alpha@\ntest\n--@end-non-alpha@]===]",
		},
		{
			name:            "Beta build type lua",
			extension:       ".lua",
			unixLineEndings: true,
			contents:        "--@alpha@\ntest\n--@end-alpha@\n\n--@beta@\ntest\n--@end-beta@\n\n--[===[@non-alpha@\ntest\n--@end-non-alpha@]===]",
			bTTM: tokens.BuildTypeTokenMap{
				tokens.Alpha:         false,
				tokens.Beta:          true,
				tokens.Classic:       false,
				tokens.Retail:        false,
				tokens.VersionRetail: false,
				tokens.VersionBcc:    false,
				tokens.VersionWrath:  false,
				tokens.VersionCata:   false,
			},
			expected: "--[===[@alpha@\ntest\n--@end-alpha@]===]\n\n--@beta@\ntest\n--@end-beta@\n\n--@non-alpha@\ntest\n--@end-non-alpha@",
		},
		{
			name:            "Debug build type lua",
			extension:       ".lua",
			unixLineEndings: true,
			contents:        "--@debug@\ntest\n--@end-debug@\n\n--[===[@non-alpha@\ntest\n--@end-non-alpha@]===]",
			bTTM: tokens.BuildTypeTokenMap{
				tokens.Alpha:         false,
				tokens.Beta:          false,
				tokens.Debug:         false,
				tokens.Classic:       false,
				tokens.Retail:        false,
				tokens.VersionRetail: false,
				tokens.VersionBcc:    false,
				tokens.VersionWrath:  false,
				tokens.VersionCata:   false,
			},
			expected: "--[===[@debug@\ntest\n--@end-debug@]===]\n\n--@non-alpha@\ntest\n--@end-non-alpha@",
		},
		{
			name:      "Alpha build type xml",
			extension: ".xml",
			contents:  "<!--@alpha@-->\n<test>\n</test>\n<!--@end-alpha@-->\n\n<!--@beta@-->\n<test>\n</test>\n<!--@end-beta@-->\n\n<!--@non-alpha@\n<test>\n</test>\n@end-non-alpha@-->\n",
			bTTM: tokens.BuildTypeTokenMap{
				tokens.Alpha:         true,
				tokens.Beta:          false,
				tokens.Classic:       false,
				tokens.Retail:        false,
				tokens.VersionRetail: false,
				tokens.VersionBcc:    false,
				tokens.VersionWrath:  false,
				tokens.VersionCata:   false,
			},
			expected: "<!--@alpha@-->\r\n<test>\r\n</test>\r\n<!--@end-alpha@-->\r\n\r\n<!--@beta@\r\n<test>\r\n</test>\r\n@end-beta@-->\r\n\r\n<!--@non-alpha@\r\n<test>\r\n</test>\r\n@end-non-alpha@-->\r\n",
		},
		{
			name:      "Beta build type xml",
			extension: ".xml",
			contents:  "<!--@alpha@-->\n<test>\n</test>\n<!--@end-alpha@-->\n\n<!--@beta@-->\n<test>\n</test>\n<!--@end-beta@-->\n\n<!--@non-alpha@\n<test>\n</test>\n@end-non-alpha@-->\n",
			bTTM: tokens.BuildTypeTokenMap{
				tokens.Alpha:         false,
				tokens.Beta:          true,
				tokens.Classic:       false,
				tokens.Retail:        false,
				tokens.VersionRetail: false,
				tokens.VersionBcc:    false,
				tokens.VersionWrath:  false,
				tokens.VersionCata:   false,
			},
			expected: "<!--@alpha@\r\n<test>\r\n</test>\r\n@end-alpha@-->\r\n\r\n<!--@beta@-->\r\n<test>\r\n</test>\r\n<!--@end-beta@-->\r\n\r\n<!--@non-alpha@-->\r\n<test>\r\n</test>\r\n<!--@end-non-alpha@-->\r\n",
		},
		{
			name:      "Alpha build type toc",
			extension: ".toc",
			contents:  "#@alpha@\ntest\n#@end-alpha@\n\n#@beta@\ntest\n#@end-beta@\n\n#@non-alpha@\n#test\n#@end-non-alpha@",
			bTTM: tokens.BuildTypeTokenMap{
				tokens.Alpha:         true,
				tokens.Beta:          false,
				tokens.Classic:       false,
				tokens.Retail:        false,
				tokens.VersionRetail: false,
				tokens.VersionBcc:    false,
				tokens.VersionWrath:  false,
				tokens.VersionCata:   false,
			},
			expected: "#@alpha@\r\ntest\r\n#@end-alpha@\r\n\r\n#@beta@\r\n#test\r\n#@end-beta@\r\n\r\n#@non-alpha@\r\n#test\r\n#@end-non-alpha@",
		},
		{
			name:      "Beta build type toc",
			extension: ".toc",
			contents:  "#@alpha@\ntest\n#@end-alpha@\n\n#@beta@\ntest\n#@end-beta@\n\n#@non-alpha@\n#test\n#@end-non-alpha@",
			bTTM: tokens.BuildTypeTokenMap{
				tokens.Alpha:         false,
				tokens.Beta:          true,
				tokens.Classic:       false,
				tokens.Retail:        false,
				tokens.VersionRetail: false,
				tokens.VersionBcc:    false,
				tokens.VersionWrath:  false,
				tokens.VersionCata:   false,
			},
			expected: "#@alpha@\r\n#test\r\n#@end-alpha@\r\n\r\n#@beta@\r\ntest\r\n#@end-beta@\r\n\r\n#@non-alpha@\r\ntest\r\n#@end-non-alpha@",
		},
		{
			name:      "Do not package",
			extension: ".lua",
			contents:  "--@do-not-package@\ntest\n--@end-do-not-package@\ntest\ntest\ntest\n",
			bTTM:      tokens.BuildTypeTokenMap{},
			expected:  "test\r\ntest\r\ntest\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unixLineEndings {
				cliflags.UnixLineEndings = true
			} else {
				cliflags.UnixLineEndings = false
			}

			f, err := os.CreateTemp(".", "file*"+tt.extension)
			name := f.Name()
			filePath := filepath.Join(".", name)
			assert.NoError(t, err)
			f.WriteString(tt.contents)
			assert.NoError(t, f.Close())

			vcsRepo := &repo.MockVcsRepo{}
			injector, err := NewInjector(tokens.SimpleTokenMap{
				tokens.BuildDate: "value1",
			}, vcsRepo, ".", tt.bTTM)
			assert.NoError(t, err)

			injector.logGroup = logger.NewLogGroup("💉 Injecting tokens into package directory")

			err = injector.findAndReplaceInFile(filePath)
			assert.NoError(t, err)

			contents, err := os.ReadFile(filePath)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(contents))
			os.Remove(filePath)
		})
	}
}
