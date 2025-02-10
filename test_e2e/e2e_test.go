package test_e2e

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/McTalian/wow-build-tools/cmd"
	"github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/stretchr/testify/assert"
)

const (
	legacyTool     = "../bin/old_tool" // Path to the existing tool
	integrationDir = "integration_tests"
)

func TestAddonProcessing(t *testing.T) {
	tests := []struct {
		name       string
		testDir    string
		arrange    func(t *testing.T)
		assertions func(t *testing.T, output string)
	}{
		{
			"TestIgnores",
			"test_ignores",
			func(t *testing.T) {
				cliflags.SkipZip = true
				cliflags.ForceExternals = false
			},
			func(t *testing.T, output string) {
				matches, err := filepath.Glob(filepath.Join(output, "*.zip"))
				assert.NoError(t, err)
				assert.Len(t, matches, 0, "Expected 0 zip files, got %d", len(matches))
				assert.DirExists(t, filepath.Join(output, "TestIgnores"))
				assert.FileExists(t, filepath.Join(output, "TestIgnores", "TestIgnores.toc"))
				assert.FileExists(t, filepath.Join(output, "TestIgnores", "Core.lua"))
				assert.FileExists(t, filepath.Join(output, "TestIgnores", "embed.xml"))
				assert.NoFileExists(t, filepath.Join(output, "TestIgnores", "ignore_me.old"), "Ignored ignore_me.old file found")
				assert.NoFileExists(t, filepath.Join(output, "TestIgnores", "ignore_me.new"), "Ignored ignore_me.new file found")
				assert.NoFileExists(t, filepath.Join(output, "TestIgnores", "example.jpg"), "Ignored example.jpg file found")
				assert.DirExists(t, filepath.Join(output, "TestIgnores", "Modules"))
				assert.NoFileExists(t, filepath.Join(output, "TestIgnores", "Modules", "Debug.lua"), "Ignored Debug.lua file found")
				assert.NoFileExists(t, filepath.Join(output, "TestIgnores", "Modules", "debug.jpg"), "Ignored debug.jpg file found")
				assert.NoFileExists(t, filepath.Join(output, "TestIgnores", "Modules", "ignore_me.always"), "Ignored ignore_me.always file found")
				assert.DirExists(t, filepath.Join(output, "TestIgnores", "Modules", "Suit"))
				assert.FileExists(t, filepath.Join(output, "TestIgnores", "Modules", "Suit", "Core.lua"))
				assert.DirExists(t, filepath.Join(output, "TestIgnores", "Modules", "Hat"))
				assert.FileExists(t, filepath.Join(output, "TestIgnores", "Modules", "Hat", "Core.lua"))
			},
		},
		{
			"TestSvnExternals",
			"test_svn_externals",
			func(t *testing.T) {
				cliflags.SkipZip = true
				cliflags.ForceExternals = true
			},
			func(t *testing.T, output string) {
				assert.DirExists(t, filepath.Join(output, "TestSvnExternals"))
				assert.FileExists(t, filepath.Join(output, "TestSvnExternals", "TestSvnExternals.toc"))
				assert.FileExists(t, filepath.Join(output, "TestSvnExternals", "Core.lua"))
				assert.FileExists(t, filepath.Join(output, "TestSvnExternals", "embed.xml"))
				assert.DirExists(t, filepath.Join(output, "TestSvnExternals", "Libs"))
				assert.DirExists(t, filepath.Join(output, "TestSvnExternals", "Libs", "LibStub"))
				assert.FileExists(t, filepath.Join(output, "TestSvnExternals", "Libs", "LibStub", "LibStub.lua"))
				assert.DirExists(t, filepath.Join(output, "TestSvnExternals", "Libs", "CallbackHandler-1.0"))
				assert.FileExists(t, filepath.Join(output, "TestSvnExternals", "Libs", "CallbackHandler-1.0", "CallbackHandler-1.0.lua"))
				assert.FileExists(t, filepath.Join(output, "TestSvnExternals", "Libs", "CallbackHandler-1.0", "CallbackHandler-1.0.xml"))
				// assert.DirExists(t, filepath.Join(output, "TestSvnExternals", "Libs", "AceAddon-3.0"))
				// assert.FileExists(t, filepath.Join(output, "TestSvnExternals", "Libs", "AceAddon-3.0", "AceAddon-3.0.lua"))
				// assert.FileExists(t, filepath.Join(output, "TestSvnExternals", "Libs", "AceAddon-3.0", "AceAddon-3.0.xml"))
				assert.DirExists(t, filepath.Join(output, "TestSvnExternals", "Libs", "AceBucket-3.0"))
				assert.FileExists(t, filepath.Join(output, "TestSvnExternals", "Libs", "AceBucket-3.0", "AceBucket-3.0.lua"))
				assert.FileExists(t, filepath.Join(output, "TestSvnExternals", "Libs", "AceBucket-3.0", "AceBucket-3.0.xml"))
			},
		},
		{
			"TestGitExternals",
			"test_git_externals",
			func(t *testing.T) {
				cliflags.SkipZip = true
				cliflags.ForceExternals = true
			},
			func(t *testing.T, output string) {
				assert.DirExists(t, filepath.Join(output, "TestGitExternals"))
				assert.DirExists(t, filepath.Join(output, "TestGitExternals", "Libs"))
				assert.DirExists(t, filepath.Join(output, "TestGitExternals", "Libs", "LibClassicSwingTimerAPI"))
				assert.FileExists(t, filepath.Join(output, "TestGitExternals", "Libs", "LibClassicSwingTimerAPI", "LibClassicSwingTimerAPI.lua"))
				assert.DirExists(t, filepath.Join(output, "TestGitExternals", "Libs", "LibDataBroker-1.1"))
				assert.FileExists(t, filepath.Join(output, "TestGitExternals", "Libs", "LibDataBroker-1.1", "LibDataBroker-1.1.lua"))
				assert.DirExists(t, filepath.Join(output, "TestGitExternals", "Libs", "LibDeflate"))
				assert.FileExists(t, filepath.Join(output, "TestGitExternals", "Libs", "LibDeflate", "LibDeflate.lua"))
				assert.DirExists(t, filepath.Join(output, "TestGitExternals", "Libs", "LibSpellRange-1.0"))
				assert.FileExists(t, filepath.Join(output, "TestGitExternals", "Libs", "LibSpellRange-1.0", "LibSpellRange-1.0.lua"))
			},
		},
		{
			"TestZip",
			"test_zip",
			func(t *testing.T) {
				cliflags.SkipZip = false
				cliflags.ForceExternals = false
			},
			func(t *testing.T, output string) {
				// time.Sleep(1 * time.Second) // Wait for the zip file to be created
				matches, err := filepath.Glob(filepath.Join(output, "*.zip"))
				assert.NoError(t, err)
				assert.Len(t, matches, 1, "Expected 1 zip file, got %d", len(matches))
				assert.DirExists(t, filepath.Join(output, "TestZip"))
				assert.FileExists(t, filepath.Join(output, "TestZip", "TestZip.toc"))
				assert.FileExists(t, filepath.Join(output, "TestZip", "Core.lua"))
			},
		},
		{
			"TestZipNoLib",
			"test_zip_nolib",
			func(t *testing.T) {
				cliflags.SkipZip = false
				cliflags.ForceExternals = false
			},
			func(t *testing.T, output string) {
				// time.Sleep(1 * time.Second) // Wait for the zip file to be created
				matches, err := filepath.Glob(filepath.Join(output, "*.zip"))
				assert.NoError(t, err)
				assert.Len(t, matches, 2, "Expected 2 zip file(s), got %d", len(matches))
				assert.DirExists(t, filepath.Join(output, "TestZipNoLib"))
				assert.FileExists(t, filepath.Join(output, "TestZipNoLib", "TestZipNoLib.toc"))
				assert.FileExists(t, filepath.Join(output, "TestZipNoLib", "Core.lua"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempNewOutput, err := filepath.Abs(filepath.Join(".", tt.testDir, ".release"))
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}
			if _, err := os.Stat(tempNewOutput); err == nil {
				if err := os.RemoveAll(tempNewOutput); err != nil {
					t.Fatalf("Failed to remove old output directory: %v", err)
				}
			}

			tt.arrange(t)

			// Run the new CLI directly
			runNewCLI(tt.testDir, tempNewOutput)

			tt.assertions(t, tempNewOutput)
		})
	}
}

var argsMutex sync.Mutex

func runNewCLI(input, output string) {
	// Set test arguments as if they were passed via CLI
	os.Args = []string{"wow-build-tools", "build", "-t", input, "-r", output}

	// Capture stdout/stderr if needed
	oldStdout, oldStderr := os.Stdout, os.Stderr
	defer func() { os.Stdout, os.Stderr = oldStdout, oldStderr }() // Restore after execution

	_, wOut, _ := os.Pipe()
	_, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr

	// Call the CLI’s main function directly
	// cmd.GetRootCmd().SetArgs(testArgs)
	logger.InitLogger()
	cmd.Execute()

	// Close the write ends of the pipes
	wOut.Close()
	wErr.Close()
}
