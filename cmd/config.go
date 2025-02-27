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
	"slices"
	"strconv"
	"strings"

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
		response = strings.TrimSpace(response)

		if response == "y" || response == "Y" {
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

type Flavor string

const (
	retail        Flavor = "retail"
	classic       Flavor = "classic"
	classicEra    Flavor = "classicEra"
	ptr           Flavor = "ptr"
	xptr          Flavor = "xptr"
	classicPtr    Flavor = "classicPtr"
	classicEraPtr Flavor = "classicEraPtr"
)

var knownFlavors = []Flavor{retail, classic, classicEra, ptr, xptr, classicPtr, classicEraPtr}

func (f Flavor) ToDir() string {
	switch f {
	case retail:
		return "_retail_"
	case classic:
		return "_classic_"
	case classicEra:
		return "_classic_era_"
	case ptr:
		return "_ptr_"
	case xptr:
		return "_xptr_"
	case classicPtr:
		return "_classic_ptr_"
	case classicEraPtr:
		return "_classic_era_ptr_"
	default:
		return ""
	}
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func setFlavorPath(reader *bufio.Reader, flavor Flavor, value ...string) error {
	var flavorPath string
	if len(value) == 1 {
		flavorPath = value[0]
	} else {
		basePath := viper.Get("wowPath.base")
		var defaultPath string
		if basePath != nil {
			defaultPath = viper.GetString("wowPath.base") + string(os.PathSeparator)
		} else if os.PathSeparator == '\\' {
			defaultPath = "C:\\Program Files (x86)\\World of Warcraft\\"
		} else {
			defaultPath = "/mnt/c/Program Files (x86)/World of Warcraft/"
		}

		if viper.Get("wowPath."+string(flavor)) != nil {
			defaultPath = viper.GetString("wowPath." + string(flavor))
		} else {
			defaultPath += flavor.ToDir()
		}

		logger.Prompt("Enter the path to your %s WoW installation [%s]: ", capitalize(string(flavor)), defaultPath)
		flavorPath, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		flavorPath = strings.TrimSpace(flavorPath)
		if flavorPath == "" {
			flavorPath = defaultPath
		}
	}

	viper.Set("wowPath."+string(flavor), flavorPath)
	logger.Success("%s World of Warcraft installation path set to: %s", capitalize(string(flavor)), flavorPath)

	return nil
}

func setWoWPath(reader *bufio.Reader, value ...string) error {
	var wowPath string
	var err error
	if len(value) == 1 {
		wowPath = value[0]
	} else {
		var defaultPath string
		if viper.Get("wowPath.base") != nil {
			defaultPath = viper.GetString("wowPath.base")
			// Check if Windows or Unix
		} else if os.PathSeparator == '\\' {
			defaultPath = "C:\\Program Files (x86)\\World of Warcraft"
		} else {
			defaultPath = "/mnt/c/Program Files (x86)/World of Warcraft"
		}

		logger.Prompt("Enter the path to your WoW installation [%s]: ", defaultPath)
		wowPath, err = reader.ReadString('\n')
		if err != nil {
			return err
		}
		wowPath = strings.TrimSpace(wowPath)
		if wowPath == "" {
			wowPath = defaultPath
		}
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
		} else if entry.IsDir() && entry.Name() == "_classic_ptr_" {
			logger.Success("Found Classic PTR World of Warcraft installation at: %s", filepath.Join(wowPath, entry.Name()))
			viper.Set("wowPath.classicPtr", filepath.Join(wowPath, entry.Name()))
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

	var numberFlavorMap = map[int]Flavor{}

	for {
		wowPaths := viper.GetStringMapString("wowPath")
		logger.Info("\nConfiguration Menu:")
		logger.Info("1. Set or update Base World of Warcraft installation path")
		nextNum := 2
		if len(wowPaths) >= 1 {
			for _, flavor := range knownFlavors {
				logger.Info("%d. Update %s World of Warcraft installation path", nextNum, capitalize(string(flavor)))
				numberFlavorMap[nextNum] = flavor
				nextNum++
			}
		}
		logger.Info("%d. Save and exit", nextNum)
		nextNum++
		logger.Info("%d. Exit without saving", nextNum)
		logger.Prompt("Enter your choice: ")

		choice, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		choice = strings.TrimSpace(choice)

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
				choiceNum, err := strconv.Atoi(choice)
				if err != nil {
					logger.Warn("Invalid choice, please try again.")
				} else if flavor, ok := numberFlavorMap[choiceNum]; ok {
					err = setFlavorPath(reader, flavor)
					if err != nil {
						return err
					}
				} else {
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
	Use:   "config [primaryConfigPath] [secondaryConfigPath]",
	Short: "Configure wow-build-tools",
	Long: `Configure the settings for wow-build-tools.
	This includes setting up the path to the World of Warcraft installation(s)
	for watch mode.
	
	If no positional arguments are provided, the configuration wizard will be launched.`,
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

		if err = createConfigFileIfNotExist(localPath); err != nil {
			if err == ErrConfigCreationAborted {
				return nil
			}
			return err
		}

		validPrimaryArgs := []string{"wowPath"}
		if args != nil && len(args) > 0 {
			if slices.Contains(validPrimaryArgs, args[0]) {
				if len(args) == 1 {
					logger.Error("No subcommand provided for %s configuration", args[0])
					return fmt.Errorf("no subcommand provided for %s configuration", args[0])
				}
				validSecondaryArgs := []string{"base"}
				for _, flavor := range knownFlavors {
					validSecondaryArgs = append(validSecondaryArgs, string(flavor))
				}
				if slices.Contains(validSecondaryArgs, args[1]) {
					if args[1] == "base" {
						return setWoWPath(bufio.NewReader(os.Stdin), args[2:]...)
					} else {
						flavor := Flavor(args[1])
						return setFlavorPath(bufio.NewReader(os.Stdin), flavor, args[2:]...)
					}
				} else {
					logger.Error("Invalid subcommand provided for %s configuration, %s. Must be one of %v", args[0], args[1], validSecondaryArgs)
					return fmt.Errorf("invalid subcommand provided for %s configuration", args[0])
				}
			} else {
				logger.Error("Invalid primary argument provided for configuration, %s. Must be one of %v", args[0], validPrimaryArgs)
				return fmt.Errorf("invalid primary argument provided for configuration")
			}
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
