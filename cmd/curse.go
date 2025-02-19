/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/McTalian/wow-build-tools/internal/changelog"
	f "github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/pkg"
	"github.com/McTalian/wow-build-tools/internal/toc"
	"github.com/McTalian/wow-build-tools/internal/upload"
	"github.com/spf13/cobra"
)

// curseCmd represents the curse command
var curseCmd = &cobra.Command{
	Use:   "curse",
	Short: "Upload the specified file to CurseForge",
	Long: `Upload the input zip file to CurseForge.
	
	Input, label, interface versions, and CurseForge project ID are required.
	The CF_API_KEY environment variable must also be set.`,
	Run: func(cmd *cobra.Command, args []string) {
		tmp := os.TempDir()
		tmpToc, err := os.CreateTemp(tmp, "wbt*.toc")
		if err != nil {
			logger.Error("Could not create temporary TOC file: %v", err)
			return
		}
		defer os.Remove(tmpToc.Name())
		defer tmpToc.Close()

		changelogPath := f.UploadChangelog
		if f.UploadChangelog == "" {
			tmpChangelog, err := os.CreateTemp(tmp, "wbtChangelog*.md")
			if err != nil {
				logger.Error("Could not create temporary changelog file: %v", err)
				return
			}
			defer os.Remove(tmpChangelog.Name())
			defer tmpChangelog.Close()

			_, err = tmpChangelog.WriteString("No changelog provided")
			if err != nil {
				logger.Error("Could not write to temporary changelog file: %v", err)
				return
			}
			tmpChangelog.Sync()

			changelogPath = tmpChangelog.Name()
		}

		changelog := &changelog.Changelog{
			PreExistingFilePath: changelogPath,
			MarkupType:          changelog.MarkdownMT,
		}

		interfaceStringList := []string{}
		for _, i := range f.UploadInterfaceVersions {
			interfaceStringList = append(interfaceStringList, fmt.Sprintf("%d", i))
		}

		interfaceString := strings.Join(interfaceStringList, ",")
		_, err = tmpToc.WriteString(fmt.Sprintf("## Interface: %s", interfaceString))
		if err != nil {
			logger.Error("Could not write to temporary TOC file: %v", err)
			return
		}
		tmpToc.Sync()

		tocFile, err := toc.NewToc(tmpToc.Name())
		if err != nil {
			logger.Error("Could not create TOC file: %v", err)
			return
		}

		pkgMeta := &pkg.PkgMeta{}

		curseArgs := upload.UploadCurseArgs{
			TocFiles:    []*toc.Toc{tocFile},
			ZipPath:     f.UploadInput,
			FileLabel:   f.UploadLabel,
			PkgMeta:     pkgMeta,
			Changelog:   changelog,
			ReleaseType: f.UploadReleaseType,
		}

		err = upload.UploadToCurse(curseArgs)
		if err != nil {
			logger.Error("Could not upload to curse: %v", err)
			return
		}
	},
}

func init() {
	uploadCmd.AddCommand(curseCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// curseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:

	curseCmd.Flags().StringVarP(&f.CurseId, "curseId", "p", "", "Set the CurseForge project ID for localization and uploading. (Use 0 to unset the TOC value)")
	curseCmd.MarkFlagRequired("curseId")
}
