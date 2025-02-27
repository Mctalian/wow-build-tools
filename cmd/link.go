/*
Copyright © 2025 Rob "McTalian" Anderson

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var wslPathToAddonReleaseDir string

// linkCmd represents the link command
var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Create symlinks in World of Warcraft AddOns directory to the addon(s) in the build output directory",
	Long: dedent.Dedent(`
		Create symlinks in the World of Warcraft AddOns directory to the addon(s) in the build output directory.
		
		By default, the release directory is assumed to be a ".release" directory in the top level directory of the addon.
		
		If you are developing in WSL, you will need to run this command in Windows in an elevated command prompt.
		You will also need to provide the path to the addon release directory in WSL using the --wsl-path-to-addon-release-dir flag.
		From WSL, run "wslpath -w <path_to_your_releasedir>" to get the Windows path to your release directory.`),
	RunE: func(cmd *cobra.Command, args []string) error {
		l := logger.GetSubLog("link")

		if cmd.Flags().Changed("topDir") && !cmd.Flags().Changed("releaseDir") {
			releaseDir = filepath.Join(topDir, ".release")
		}

		wowPaths := viper.GetStringMapString("wowPath")
		if len(wowPaths) <= 1 {
			l.Error("Please run `wow-build-tools config` to set up your World of Warcraft paths.")
			return fmt.Errorf("no World of Warcraft paths set")
		}

		if wslPathToAddonReleaseDir != "" {
			l.Debug("Using wslPathToAddonReleaseDir to determine WSL path to addon release directory")
			releaseDir = wslPathToAddonReleaseDir
		}

		l.Debug("Creating symlinks pointing to addons in %s", releaseDir)
		dirEntries, err := os.ReadDir(releaseDir)
		if err != nil {
			l.Error("Error reading release directory: %v", err)
			return err
		}

		addonDirs = []string{}
		for _, entry := range dirEntries {
			if entry.IsDir() {
				addonDirs = append(addonDirs, entry.Name())
			}
		}

		if len(addonDirs) == 0 {
			l.Error("No addon directories found in release directory, please run a build first")
			return fmt.Errorf("no addon directories found in release directory")
		}

		for k, wowPath := range wowPaths {
			if k == "base" {
				continue
			}
			if _, err := os.Stat(filepath.Join(wowPath)); os.IsNotExist(err) {
				l.Error("World of Warcraft path %s does not exist", wowPath)
				return err
			}

			if _, err := os.Stat(filepath.Join(wowPath, "Interface", "AddOns")); os.IsNotExist(err) {
				l.Warn("No AddOns directory found in %s, creating it", wowPath)
				err = os.MkdirAll(filepath.Join(wowPath, "Interface", "AddOns"), 0755)
				if err != nil {
					l.Error("Error creating AddOns directory: %v", err)
					return err
				}
			}
			for _, addonDir := range addonDirs {
				wslPathToAddonReleaseDir = filepath.Join(wowPath, "Interface", "AddOns", addonDir)
				l.Info("Linking %s to %s", releaseDir, wslPathToAddonReleaseDir)
				err = os.Symlink(releaseDir, wslPathToAddonReleaseDir)
				if err != nil {
					l.Error("Error creating symlink: %v", err)
					return err
				}
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(linkCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// linkCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// linkCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	linkCmd.Flags().StringVarP(&topDir, "topDir", "t", ".", "Path to the top level directory of the addon")
	linkCmd.Flags().StringVarP(&releaseDir, "releaseDir", "r", "", "Path to the release directory of the addon")
	linkCmd.Flags().StringVarP(&wslPathToAddonReleaseDir, "wsl-path-to-addon-release-dir", "w", "", "Path to the addon release directory in WSL")
}
