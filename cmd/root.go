/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/spf13/cobra"
)

var LevelVerbose bool
var LevelDebug bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wow-build-tools",
	Short: "World of Warcraft addon build tools",
	Long: `This tool is used to build World of Warcraft addons both for local development
	and for release/distribution via CurseForge, WoWInterface, and Wago.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.wow-build-tools.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&LevelVerbose, "verbose", "V", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&LevelDebug, "debug", "v", false, "Enable debug output")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if LevelVerbose {
			logger.SetLogLevel(logger.VERBOSE)
		} else if LevelDebug {
			logger.SetLogLevel(logger.DEBUG)
		} else {
			logger.SetLogLevel(logger.INFO)
		}
		return nil
	}
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
