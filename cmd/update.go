/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/McTalian/wow-build-tools/internal/update"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update wow-build-tools",
	Long:  `Fetch the latest version of wow-build-tools from the repository and update the binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		update.DoSelfUpdate()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
