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
	"strings"

	"github.com/McTalian/wow-build-tools/internal/changelog"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/toc"
	"github.com/McTalian/wow-build-tools/internal/upload"
	"github.com/spf13/cobra"
)

var UploadProjectVersion string

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

		changelogPath := UploadChangelog
		if UploadChangelog == "" {
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
			err = tmpChangelog.Sync()
			if err != nil {
				logger.Error("Could not sync temporary changelog file: %v", err)
				return err
			}

			changelogPath = tmpChangelog.Name()
		}

		changelog := &changelog.Changelog{
			PreExistingFilePath: changelogPath,
			MarkupType:          changelog.MarkdownMT,
		}

		interfaceStringList := []string{}
		for _, i := range UploadInterfaceVersions {
			interfaceStringList = append(interfaceStringList, fmt.Sprintf("%d", i))
		}

		interfaceString := strings.Join(interfaceStringList, ",")
		_, err = tmpToc.WriteString(fmt.Sprintf("## Interface: %s", interfaceString))
		if err != nil {
			logger.Error("Could not write to temporary TOC file: %v", err)
			return err
		}
		err = tmpToc.Sync()
		if err != nil {
			logger.Error("Could not sync temporary TOC file: %v", err)
			return err
		}

		tocFile, err := toc.NewToc(tmpToc.Name())
		if err != nil {
			logger.Error("Could not create TOC file: %v", err)
			return err
		}

		w := upload.UploadWowiArgs{
			TocFiles:       []*toc.Toc{tocFile},
			ProjectVersion: UploadProjectVersion,
			ZipPath:        UploadInput,
			FileLabel:      UploadLabel,
			Changelog:      changelog,
			WowiId:         wowiId,
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
	wowiCmd.Flags().StringVarP(&wowiId, "wowiId", "w", "", "Set the WoW Interface project ID for uploading. (Use 0 to unset the TOC value)")
	err := wowiCmd.MarkFlagRequired("wowiId")
	if err != nil {
		panic(err)
	}
	wowiCmd.Flags().StringVar(&UploadProjectVersion, "project-version", "", "Set the project version for uploading")
	err = wowiCmd.MarkFlagRequired("project-version")
	if err != nil {
		panic(err)
	}
}
