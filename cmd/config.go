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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var globalConfig bool = true
var configType string
var configFile string
var wizardMode bool

var ErrConfigCreationAborted = fmt.Errorf("configuration file creation aborted")

func createConfigFileIfNotExist(localPath string) error {
	if configFile == "" || (globalConfig && configFile == localPath) {
		if configFile == localPath {
			logger.Info("Local configuration file already exists and will take precedence: %s", configFile)
		}

		logger.Info("It looks like you haven't run `wow-build-tools config` to set up %s config yet.", configType)
		logger.Prompt("Would you like to create a new %s configuration file? (y/N): ", configType)

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		if response == "y\n" || response == "Y\n" {
			if globalConfig {
				logger.Info("Creating global configuration file...")
				err := viper.SafeWriteConfig()
				if err != nil {
					return err
				}
				err = viper.ReadInConfig()
				if err != nil {
					return err
				}
				logger.Success("Configuration file created: %s", viper.ConfigFileUsed())
			} else {
				logger.Info("--global flag not set, creating local configuration file...")
				logger.Info("If you want to create a global configuration file instead, run `wow-build-tools config --global`.")
				err := viper.SafeWriteConfigAs(".wbt.yaml")
				if err != nil {
					return err
				}
				logger.Success("Configuration file created: %s", localPath)
			}
		} else {
			fmt.Println()
			logger.Info("Configuration file creation aborted.")
			return ErrConfigCreationAborted
		}
	}
	return nil
}

func setWoWPath(reader *bufio.Reader) error {
	var defaultPath string
	// Check if Windows or Unix
	if viper.Get("wowPath.base") != nil {
		defaultPath = viper.GetString("wowPath.base")
	} else if os.PathSeparator == '\\' {
		defaultPath = "C:\\Program Files (x86)\\World of Warcraft"
	} else {
		defaultPath = "/mnt/c/Program Files (x86)/World of Warcraft"
	}

	logger.Prompt("Enter the path to your WoW installation [%s]: ", defaultPath)
	wowPath, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	wowPath = wowPath[:len(wowPath)-1] // Remove newline character
	if wowPath == "" {
		wowPath = defaultPath
	}
	viper.Set("wowPath.base", wowPath)
	logger.Success("World of Warcraft installation path set to: %s", wowPath)

	fmt.Println()

	contents, err := os.ReadDir(wowPath)
	if err != nil {
		logger.Warn("Could not read directory: %v", err)
		logger.Warn("Please make sure the path is correct and try again.")
		return err
	}

	for _, entry := range contents {
		if entry.IsDir() && entry.Name() == "_retail_" {
			logger.Success("Found Retail World of Warcraft installation at: %s", filepath.Join(wowPath, entry.Name()))
			viper.Set("wowPath.retail", filepath.Join(wowPath, entry.Name()))
		} else if entry.IsDir() && entry.Name() == "_classic_" {
			logger.Success("Found Classic World of Warcraft installation at: %s", filepath.Join(wowPath, entry.Name()))
			viper.Set("wowPath.classic", filepath.Join(wowPath, entry.Name()))
		} else if entry.IsDir() && entry.Name() == "_classic_era_" {
			logger.Success("Found Classic Era World of Warcraft installation at: %s", filepath.Join(wowPath, entry.Name()))
			viper.Set("wowPath.classicEra", filepath.Join(wowPath, entry.Name()))
		} else if entry.IsDir() && entry.Name() == "_classic_era_ptr_" {
			logger.Success("Found Classic Era PTR World of Warcraft installation at: %s", filepath.Join(wowPath, entry.Name()))
			viper.Set("wowPath.classicEraPtr", filepath.Join(wowPath, entry.Name()))
		} else if entry.IsDir() && entry.Name() == "_ptr_" {
			logger.Success("Found PTR World of Warcraft installation at: %s", filepath.Join(wowPath, entry.Name()))
			viper.Set("wowPath.ptr", filepath.Join(wowPath, entry.Name()))
		} else if entry.IsDir() && entry.Name() == "_xptr_" {
			logger.Success("Found XPTR World of Warcraft installation at: %s", filepath.Join(wowPath, entry.Name()))
			viper.Set("wowPath.xptr", filepath.Join(wowPath, entry.Name()))
		}
	}

	wowPaths := viper.GetStringMapString("wowPath")
	if len(wowPaths) == 1 {
		logger.Error("No valid World of Warcraft installations found in the specified path.")
		logger.Error("Please make sure the path is correct and try again.")
		return fmt.Errorf("no valid World of Warcraft installations found in the specified path")
	} else {
		logger.Success("World of Warcraft installation paths set successfully!")
	}
	return nil
}

func runConfigWizard() error {
	reader := bufio.NewReader(os.Stdin)

	logger.Info("Welcome to the wow-build-tools configuration wizard!")
	logger.Info("Please follow the prompts to set up your configuration.")

	for {
		wowPaths := viper.GetStringMapString("wowPath")
		logger.Info("\nConfiguration Menu:")
		logger.Info("1. Set or refresh Base World of Warcraft installation path")
		nextNum := 2
		if len(wowPaths) >= 1 {
			logger.Info("%d. Set Retail path", nextNum)
			nextNum++
			logger.Info("%d. Set Classic path", nextNum)
			nextNum++
			logger.Info("%d. Set Classic Era path", nextNum)
			nextNum++
			logger.Info("%d. Set Classic Era PTR path", nextNum)
			nextNum++
			logger.Info("%d. Set PTR path", nextNum)
			nextNum++
			logger.Info("%d. Set XPTR path", nextNum)
			nextNum++
		}
		logger.Info("%d. Save and exit", nextNum)
		nextNum++
		logger.Info("%d. Exit without saving", nextNum)
		logger.Prompt("Enter your choice: ")

		choice, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		choice = choice[:len(choice)-1] // Remove newline character

		switch choice {
		case "1":
			err = setWoWPath(reader)
			if err != nil {
				return err
			}
		case strconv.Itoa(nextNum - 1):
			err = viper.WriteConfig()
			if err != nil {
				return err
			}
			logger.Success("Configuration saved successfully!")
			return nil
		case strconv.Itoa(nextNum):
			logger.Info("Exiting without saving.")
			return nil
		default:
			if len(wowPaths) >= 1 {
				switch choice {
				case "2":
					logger.Warn("Not implemented yet.")
				case "3":
					logger.Warn("Not implemented yet.")
				case "4":
					logger.Warn("Not implemented yet.")
				case "5":
					logger.Warn("Not implemented yet.")
				case "6":
					logger.Warn("Not implemented yet.")
				case "7":
					logger.Warn("Not implemented yet.")
				default:
					logger.Warn("Invalid choice, please try again.")
				}
			} else {
				logger.Warn("Invalid choice, please try again.")
			}
		}
	}
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure wow-build-tools",
	Long: `Configure the settings for wow-build-tools.
	This includes setting up the path to the World of Warcraft installation(s)
	for watch mode.
	
	Configuration can be local (current directory) or global (home directory).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		localPath, err := filepath.Abs(filepath.Join(".", ".wbt.yaml"))
		if err != nil {
			return err
		}

		configType = "local"
		if globalConfig {
			configType = "global"
		}
		configFile = viper.ConfigFileUsed()

		logger.Warn("Config: %s", configFile)

		if err = createConfigFileIfNotExist(localPath); err != nil {
			if err == ErrConfigCreationAborted {
				return nil
			}
			return err
		}

		return runConfigWizard()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	// configCmd.PersistentFlags().BoolVarP(&globalConfig, "global", "g", false, "Use global configuration")
}
