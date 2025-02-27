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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/McTalian/wow-build-tools/internal/cmdimpl"
)

var (
	topDir           string
	releaseDir       string
	pkgmetaFile      string
	keepPackageDir   bool
	createNoLib      bool
	curseId          string
	wowiId           string
	wagoId           string
	skipCopy         bool
	skipChangelog    bool
	skipExternals    bool
	forceExternals   bool
	skipZip          bool
	skipUpload       bool
	nameTemplate     string
	skipLocalization bool
	onlyLocalization bool
	splitToc         bool
	unixLineEndings  bool
	gameVersion      string
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Builds a World of Warcraft addon",
	Long:  `This command packages the addon as specified via a pkgmeta file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("topDir") && !cmd.Flags().Changed("releaseDir") {
			releaseDir = filepath.Join(topDir, ".release")
		}

		buildArgs := &cmdimpl.BuildArgs{
			TopDir:           topDir,
			ReleaseDir:       releaseDir,
			CurseId:          curseId,
			WowiId:           wowiId,
			WagoId:           wagoId,
			PkgmetaFile:      pkgmetaFile,
			KeepPackageDir:   keepPackageDir,
			CreateNoLib:      createNoLib,
			SkipCopy:         skipCopy,
			SkipChangelog:    skipChangelog,
			SkipExternals:    skipExternals,
			ForceExternals:   forceExternals,
			SkipZip:          skipZip,
			SkipUpload:       skipUpload,
			NameTemplate:     nameTemplate,
			SkipLocalization: skipLocalization,
			OnlyLocalization: onlyLocalization,
			SplitToc:         splitToc,
			UnixLineEndings:  unixLineEndings,
			GameVersion:      gameVersion,
			LevelVerbose:     LevelVerbose,
			LevelDebug:       LevelDebug,
		}

		return cmdimpl.Build(buildArgs)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().SortFlags = false

	buildCmd.Flags().StringVarP(&topDir, "topDir", "t", ".", "The top level directory of the addon")
	buildCmd.Flags().StringVarP(&releaseDir, "releaseDir", "r", topDir+string(os.PathSeparator)+".release", "The directory to output the release files.")
	buildCmd.Flags().StringVarP(&pkgmetaFile, "pkgmetaFile", "m", "", "Set the pkgmeta file to use. (Defaults to {topDir}/pkgmeta.yml, {topDir}/pkgmeta.yaml, or {topDir}/.pkgmeta if one exists.)")
	buildCmd.Flags().BoolVarP(&keepPackageDir, "keepPackageDir", "o", false, "Keep existing package directory, overwriting its contents.")
	buildCmd.Flags().BoolVarP(&createNoLib, "createNoLib", "s", false, "Create a stripped-down \"nolib\" package.")
	buildCmd.Flags().StringVarP(&curseId, "curseId", "p", "", "Set the CurseForge project ID for localization and uploading. (Use 0 to unset the TOC value)")
	buildCmd.Flags().StringVarP(&wowiId, "wowiId", "w", "", "Set the WoWInterface project ID for uploading. (Use 0 to unset the TOC value)")
	buildCmd.Flags().StringVarP(&wagoId, "wagoId", "a", "", "Set the Wago project ID for uploading. (Use 0 to unset the TOC value)")
	buildCmd.Flags().BoolVarP(&skipCopy, "skipCopy", "c", false, "Skip copying the files to the output directory.")
	buildCmd.Flags().BoolVar(&skipChangelog, "skipChangelog", false, "Skip changelog generation.")
	buildCmd.Flags().BoolVarP(&skipExternals, "skipExternals", "e", false, "Skip fetching externals.")
	buildCmd.Flags().BoolVarP(&forceExternals, "forceExternals", "E", false, "Force fetching externals, bypassing the cache.")
	buildCmd.Flags().BoolVarP(&skipZip, "skipZip", "z", false, "Skip zipping the package (and uploading).")
	buildCmd.Flags().BoolVarP(&skipUpload, "skipUpload", "d", false, "Skip uploading.")
	buildCmd.Flags().StringVarP(&nameTemplate, "nameTemplate", "n", "", "Set the name template to use for the release file. Use \"-n help\" for more info.")
	buildCmd.Flags().BoolVarP(&skipLocalization, "skipLocalization", "l", false, "Skip @localization@ keyword replacement.")
	buildCmd.Flags().BoolVarP(&onlyLocalization, "onlyLocalization", "L", false, "Only do @localization@ keyword replacement (skip upload to CurseForge).")
	buildCmd.Flags().BoolVarP(&splitToc, "splitToc", "S", false, "Create a package supporting multiple game types from a single TOC file.")
	buildCmd.Flags().BoolVarP(&unixLineEndings, "unixLineEndings", "u", false, "Use Unix line endings in TOC and XML files.")
	buildCmd.Flags().StringVarP(&gameVersion, "gameVersion", "g", "", "Set the game version to use for uploading.")
}
