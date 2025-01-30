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
	"github.com/McTalian/wow-build-tools/internal/toc"
	"github.com/McTalian/wow-build-tools/internal/upload"
	"github.com/spf13/cobra"
)

// wowiCmd represents the wowi command
var wowiCmd = &cobra.Command{
	Use:   "wowi",
	Short: "Upload the specified file to WoWInterface",
	Long: `Upload the input zip file to WoWInterface.
	
	Input, label, and WoWInterface project ID are required.
	The WOWI_API_TOKEN environment variable must also be set.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tmp := os.TempDir()
		tmpToc, err := os.CreateTemp(tmp, "wbt*.toc")
		if err != nil {
			logger.Error("Could not create temporary TOC file: %v", err)
			return err
		}
		defer os.Remove(tmpToc.Name())
		defer tmpToc.Close()

		changelogPath := f.UploadChangelog
		if f.UploadChangelog == "" {
			tmpChangelog, err := os.CreateTemp(tmp, "wbtChangelog*.md")
			if err != nil {
				logger.Error("Could not create temporary changelog file: %v", err)
				return err
			}
			defer os.Remove(tmpChangelog.Name())
			defer tmpChangelog.Close()

			_, err = tmpChangelog.WriteString("No changelog provided")
			if err != nil {
				logger.Error("Could not write to temporary changelog file: %v", err)
				return err
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
			return err
		}
		tmpToc.Sync()

		tocFile, err := toc.NewToc(tmpToc.Name())
		if err != nil {
			logger.Error("Could not create TOC file: %v", err)
			return err
		}

		w := upload.UploadWowiArgs{
			TocFiles:       []*toc.Toc{tocFile},
			ProjectVersion: f.UploadProjectVersion,
			ZipPath:        f.UploadInput,
			FileLabel:      f.UploadLabel,
			Changelog:      changelog,
		}

		err = upload.UploadToWowi(w)
		if err != nil {
			logger.Error("Could not upload to WoWInterface: %v", err)
			return err
		}

		return nil
	},
}

func init() {
	uploadCmd.AddCommand(wowiCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// wowiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// wowiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	wowiCmd.Flags().StringVarP(&f.WowiId, "wowiId", "w", "", "Set the WoW Interface project ID for uploading. (Use 0 to unset the TOC value)")
	wowiCmd.MarkFlagRequired("wowiId")
	wowiCmd.Flags().StringVar(&f.UploadProjectVersion, "project-version", "", "Set the project version for uploading")
	wowiCmd.MarkFlagRequired("project-version")
}
