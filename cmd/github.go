/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/McTalian/wow-build-tools/internal/github"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/spf13/cobra"
)

var slug string
var tag string

// githubCmd represents the github command
var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "GitHub related functionality for wow-build-tools",
	Long: `GitHub related functionality for wow-build-tools.
	
	This includes checking if the current environment is a GitHub Action, getting the temporary directory for the runner, and setting output variables.
	It also handles getting the release ID for a given repository and tag.`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		releaseId, err := github.GetReleaseId(slug, tag)
		if err != nil {
			logger.Error("Failed to get release ID")
			return
		}

		logger.Info("Release ID for %s:%s is %d", slug, tag, releaseId)
		return
	},
}

func init() {
	rootCmd.AddCommand(githubCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// githubCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// githubCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	githubCmd.Flags().StringVarP(&slug, "slug", "s", "", "The slug of the repository to check")
	githubCmd.Flags().StringVarP(&tag, "tag", "t", "", "The tag to check")
}
