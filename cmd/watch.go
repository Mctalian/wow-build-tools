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
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/McTalian/wow-build-tools/internal/cmdimpl"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/osutil"
	"github.com/McTalian/wow-build-tools/internal/toc"
)

var copyToWowDirs bool

var addonDirs []string
var destinationPaths []string
var wowPaths map[string]string

func copyToWow(l *logger.Logger, done chan error) {
	if copyToWowDirs {
		l.Info("Copying to WoW directories...")
		lg := logger.NewLogGroup("Copy to WoW Directories", l)

		var copyWg sync.WaitGroup
		for _, path := range destinationPaths {
			copyWg.Add(1)
			go func(path string) {
				defer copyWg.Done()
				interfaceDir := filepath.Join(path, "Interface", "AddOns")
				if _, err := os.Stat(interfaceDir); os.IsNotExist(err) {
					os.MkdirAll(interfaceDir, os.ModePerm)
				}

				for _, dir := range addonDirs {
					src := filepath.Join(releaseDir, dir)
					dst := filepath.Join(interfaceDir, dir)
					l.Debug("Copying %s to %s", src, dst)
					err := copyDir(src, dst)
					if err != nil {
						l.Error("Error copying %s to %s: %v", src, dst, err)
						done <- err
					}
				}
			}(path)
		}

		copyWg.Wait()
		lg.Flush()
	}
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	sfi, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Skip copy if destination exists and modification times are equal.
	if dfi, err := os.Stat(dst); err == nil {
		if sfi.ModTime().Equal(dfi.ModTime()) && sfi.Size() == dfi.Size() {
			return nil
		}
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure the destination directory exists.
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil && !os.IsExist(err) {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy file contents.
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Preserve modification time.
	return os.Chtimes(dst, sfi.ModTime(), sfi.ModTime())
}

// copyDir recursively copies a directory from src to dst concurrently.
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(entries))

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		info, err := entry.Info()
		if err != nil {
			return err
		}

		if info.IsDir() {
			wg.Add(1)
			// Recurse into directories concurrently.
			go func(srcDir, dstDir string) {
				defer wg.Done()
				if err := copyDir(srcDir, dstDir); err != nil {
					errCh <- err
				}
			}(srcPath, dstPath)
		} else {
			wg.Add(1)
			// Copy files concurrently.
			go func(srcFile, dstFile string) {
				defer wg.Done()
				if err := copyFile(srcFile, dstFile); err != nil {
					errCh <- err
				}
			}(srcPath, dstPath)
		}
	}

	wg.Wait()
	close(errCh)

	// Return the first error encountered, if any.
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func triggerBuild(done chan error) {
	buildArgs := &cmdimpl.BuildArgs{
		TopDir:         topDir,
		ReleaseDir:     releaseDir,
		SkipChangelog:  true,
		SkipUpload:     true,
		SkipZip:        true,
		KeepPackageDir: true,
		WatchMode:      true,
	}
	logger.Clear()
	err := cmdimpl.Build(buildArgs)
	if err != nil {
		logger.Error("Error running build command: %v", err)
		done <- err
		return
	}
	fmt.Println()
}

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Run build when files change",
	Long: dedent.Dedent(`
	Watches the current directory for changes and runs the build command when a change is detected.
	
	Running "wow-build-tools link" before running this command is recommended to ensure that the build output directories are symlinked to your WoW installation directories.
	
	You can enable "--copyToWowDirs" as an alterative. The build output directories will then be copied to configured WoW installation directories.
	When copying from WSL to the host system, the copies can be slower than desired.
	`),
	RunE: func(cmd *cobra.Command, args []string) error {
		l := logger.GetSubLog("WATCH")
		if LevelVerbose {
			l.SetLogLevel(logger.VERBOSE)
		} else if LevelDebug {
			l.SetLogLevel(logger.DEBUG)
		} else {
			l.SetLogLevel(logger.INFO)
		}

		topdir := topDir
		if !cmd.Flags().Changed("releaseDir") {
			l.Warn("No release directory specified, defaulting to .release in top directory %s", topdir)
			releaseDir = filepath.Join(topdir, ".release")
		}

		if _, err := os.Stat(releaseDir); os.IsNotExist(err) {
			err := os.MkdirAll(releaseDir, os.ModePerm)
			if err != nil {
				l.Error("Error creating release directory: %v", err)
				return err
			}
		}

		logger.Warn("IsWSL: %t", osutil.IsWSL())

		if osutil.IsWSL() && !copyToWowDirs {
			winPath, err := osutil.GetWindowsPath(releaseDir)
			if err != nil {
				l.Error("Error getting Windows path: %v", err)
				return err
			}

			l.Warn("To create symlinks to your release directory in WSL, run this command in Windows in an elevated command prompt:")
			l.Warn("wow-build-tools.exe link -w \"%s\"", winPath)
		}

		err := os.RemoveAll(releaseDir)
		if err != nil && !os.IsNotExist(err) {
			logger.Error("Error removing release dir")
			return err
		}

		initialBuildChan := make(chan error, 1)
		triggerBuild(initialBuildChan)
		close(initialBuildChan)
		for err := range initialBuildChan {
			if err != nil {
				return err
			}
		}

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			l.Error("Error creating watcher: %v", err)
			return err
		}
		defer watcher.Close()

		if copyToWowDirs {
			wowPaths = viper.GetStringMapString("wowPath")
			if len(wowPaths) <= 1 {
				l.Error("No WoW paths configured, please run 'wow-build-tools config' to configure your WoW paths")
				return fmt.Errorf("no WoW paths configured")
			}
			destinationPaths = make([]string, 0, len(wowPaths)-1)
			for key, path := range wowPaths {
				if key == "base" {
					continue
				}
				destinationPaths = append(destinationPaths, path)
			}
		}

		debounceDuration := 500 * time.Millisecond
		var debounceTimer *time.Timer

		done := make(chan error)
		go func() {
			// Reset the debounce timer. When it fires, run the build.
			debounceTimer = time.AfterFunc(debounceDuration, func() {
				// It's a good idea to ensure builds don’t run concurrently.
				// You can use a mutex, a channel, or a boolean flag as in your current implementation.
				l.Debug("Debounced change detected, triggering build...")
				triggerBuild(done)

				if copyToWowDirs {
					l.Info("Build complete, determining outputs to copy...")
					dirEntries, err := os.ReadDir(releaseDir)
					if err != nil {
						l.Error("Error reading release directory: %v", err)
						done <- err
					}

					addonDirs = []string{}
					for _, entry := range dirEntries {
						if entry.IsDir() {
							addonDirs = append(addonDirs, entry.Name())
						}
					}

					copyToWow(l, done)
				}

				l.Info("Watching for changes... Press Ctrl+C to stop.")
			})
			debounceTimer.Stop()

			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						done <- fmt.Errorf("error reading from watcher")
					}
					if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
						if strings.Contains(event.Name, releaseDir) {
							l.Debug("Skipping change event on release directory")
							continue
						}

						debounceTimer.Reset(debounceDuration)
						l.Debug("Change %s detected on %s, debouncing...", event.Op, event.Name)
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						done <- fmt.Errorf("error reading from watcher")
					}
					l.Error("Watcher error: %v", err)
					done <- err
				}
			}
		}()

		tree, err := toc.GetTocFileTree(topdir)
		if err != nil {
			l.Error("Error getting TOC file tree: %v", err)
			return err
		}

		l.Verbose("Tree: %v", tree)

		var entries []string
		for _, file := range tree {
			if filepath.Ext(file) == ".xml" {
				l.Verbose("Walking XML file: %s", file)
				xmlEntries, err := toc.WalkXmlFile(file)
				if err != nil {
					l.Error("Error walking XML file: %v", err)
					return err
				}
				entries = append(entries, xmlEntries...)
			} else {
				l.Verbose("Adding file: %s", file)
				entries = append(entries, file)
			}
		}

		var dirsToWatchSet = make(map[string]bool)
		for _, entry := range entries {
			if f, err := os.Stat(entry); err == nil {
				if f.IsDir() {
					dirsToWatchSet[entry] = true
				} else {
					dirsToWatchSet[filepath.Dir(entry)] = true
				}
			}
		}

		for dir := range dirsToWatchSet {
			err = watcher.Add(dir)
			if err != nil {
				l.Error("Error adding directory to watcher: %v", err)
				return err
			}
			l.Debug("Watching directory: %s", dir)
		}

		l.Info("Watching for changes... Press Ctrl+C to stop.")

		<-done

		return nil
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// watchCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	watchCmd.Flags().StringVarP(&topDir, "topDir", "t", ".", "Top level directory to watch for changes")
	watchCmd.Flags().StringVarP(&releaseDir, "releaseDir", "r", "."+string(os.PathSeparator)+".release", "Directory to copy output to")
	watchCmd.Flags().BoolVarP(&copyToWowDirs, "copyToWowDirs", "w", false, "Copy output to configured WoW directories.")
}
