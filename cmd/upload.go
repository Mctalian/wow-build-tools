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

	"github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/spf13/cobra"
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("upload called")
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// uploadCmd.PersistentFlags().String("foo", "", "A help for foo")
	uploadCmd.PersistentFlags().StringVarP(&cliflags.UploadInput, "input", "i", "", "Path to the addon zip file to upload")
	err := uploadCmd.MarkPersistentFlagFilename("input")
	if err != nil {
		panic(err)
	}
	err = uploadCmd.MarkPersistentFlagRequired("input")
	if err != nil {
		panic(err)
	}
	uploadCmd.PersistentFlags().StringVarP(&cliflags.UploadLabel, "label", "l", "", "Label for the uploaded file")
	err = uploadCmd.MarkPersistentFlagRequired("label")
	if err != nil {
		panic(err)
	}
	uploadCmd.PersistentFlags().IntSliceVar(&cliflags.UploadInterfaceVersions, "interface-versions", []int{}, "Interface versions that your addon supports.")
	err = uploadCmd.MarkPersistentFlagRequired("interface-versions")
	if err != nil {
		panic(err)
	}
	uploadCmd.PersistentFlags().StringVarP(&cliflags.UploadChangelog, "changelog", "c", "", "Path to the changelog file")
	err = uploadCmd.MarkPersistentFlagFilename("changelog")
	if err != nil {
		panic(err)
	}
	uploadCmd.PersistentFlags().StringVarP(&cliflags.UploadReleaseType, "release-type", "r", "alpha", "Release type for the uploaded file")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// uploadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
