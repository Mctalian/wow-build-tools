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
		release, err := github.GetRelease(slug, tag)
		if err != nil {
			logger.Error("Failed to get release ID")
			return
		}

		logger.Info("Release ID: %d", release.Id)
		logger.Info("Tag Name: %s", release.TagName)
		logger.Info("Name: %s", release.Name)
		logger.Info("Draft: %t", release.Draft)
		logger.Info("Prerelease: %t", release.Prerelease)
		logger.Info("%s", release.Body)
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
