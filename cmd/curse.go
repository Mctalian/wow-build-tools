/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	f "github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/toc"
	"github.com/McTalian/wow-build-tools/internal/upload"
	"github.com/spf13/cobra"
)

// curseCmd represents the curse command
var curseCmd = &cobra.Command{
	Use:   "curse",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		tocFile := &toc.Toc{
			Interface: f.UploadInterfaceVersions,
		}

		curseArgs := upload.UploadCurseArgs{
			ZipPath:   f.UploadInput,
			FileLabel: f.UploadLabel,
			TocFiles:  []*toc.Toc{tocFile},
		}

		err := upload.UploadToCurse(curseArgs)
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
}
