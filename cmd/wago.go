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

// wagoCmd represents the wago command
var wagoCmd = &cobra.Command{
	Use:   "wago",
	Short: "Upload the specified file to Wago.io",
	Long: `Upload the input zip file to Wago.io.
	
	Input, label, and Wago.io project ID are required.
	The WAGO_API_TOKEN environment variable must also be set.`,
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

		wagoArgs := upload.UploadWagoArgs{
			ZipPath:     f.UploadInput,
			FileLabel:   f.UploadLabel,
			ReleaseType: f.UploadReleaseType,
			TocFiles:    []*toc.Toc{tocFile},
			Changelog:   changelog,
		}

		err = upload.UploadToWago(wagoArgs)
		if err != nil {
			logger.Error("Could not upload to wago: %v", err)
			return err
		}

		return nil
	},
}

func init() {
	uploadCmd.AddCommand(wagoCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// wagoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// wagoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	wagoCmd.Flags().StringVarP(&f.WagoId, "wagoId", "a", "", "Set the Wago project ID for uploading. (Use 0 to unset the TOC value)")
	wagoCmd.MarkFlagRequired("wagoId")
}
