/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/McTalian/wow-build-tools/internal/cachedir"
	f "github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/McTalian/wow-build-tools/internal/external"
	"github.com/McTalian/wow-build-tools/internal/github"
	"github.com/McTalian/wow-build-tools/internal/injector"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/pkg"
	"github.com/McTalian/wow-build-tools/internal/repo"
	"github.com/McTalian/wow-build-tools/internal/toc"
	"github.com/McTalian/wow-build-tools/internal/tokens"
	"github.com/McTalian/wow-build-tools/internal/upload"
	"github.com/McTalian/wow-build-tools/internal/zipper"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Builds a World of Warcraft addon",
	Long:  `This command packages the addon as specified via a configuration file.`,
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		if LevelVerbose {
			logger.SetLogLevel(logger.VERBOSE)
		} else if LevelDebug {
			logger.SetLogLevel(logger.DEBUG)
		} else {
			logger.SetLogLevel(logger.INFO)
		}

		err := f.ValidateInputArgs()
		if err != nil {
			logger.Error("Error validating input arguments: %v", err)
			return
		}

		classic := false
		var templateTokens *tokens.NameTemplate
		if f.NameTemplate == "help" {
			logger.Info("%s", tokens.NameTemplateUsageInfo())
			return
		} else {
			templateTokens, err = tokens.NewNameTemplate(f.NameTemplate)
			if err != nil {
				logger.Error("Error parsing name template: %v", err)
				return
			}
		}

		buildTimestamp := time.Now().Unix()
		buildDate := time.Now().UTC().Format("2006-01-02")
		buildDateIso := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		buildDateInteger := time.Now().UTC().Format("20060102150405")
		buildTimestampStr := strconv.FormatInt(buildTimestamp, 10)
		topDir := f.TopDir
		if cmd.Flags().Changed("topDir") && !cmd.Flags().Changed("releaseDir") {
			f.ReleaseDir = topDir + "/.release"
		}

		if _, err := cachedir.Create(); err != nil {
			logger.Error("Cache Error: %v", err)
			return
		}

		tocFilePaths, err := toc.FindTocFiles(topDir)
		if err != nil {
			logger.Error("TOC Error: %v", err)
			return
		}

		logger.Verbose("TOC Files: %v", tocFilePaths)

		projectName := toc.DetermineProjectName(tocFilePaths)

		logger.Info("🔨 Building %s...", projectName)

		var tocFiles []*toc.Toc
		for _, tocFilePath := range tocFilePaths {
			t, err := toc.NewToc(tocFilePath)
			if err != nil {
				logger.Error("TOC Error: %v", err)
				return
			}
			tocFiles = append(tocFiles, t)

			logger.Verbose("%s, %s", t.Filepath, t.Flavor.ToString())
		}

		r, err := repo.NewRepo(topDir)
		if err != nil {
			logger.Error("Repo Error: %v", err)
			return
		}

		logger.Debug("%s", r.String())

		preVr := time.Now()
		var vR repo.VcsRepo
		switch r.GetVcsType() {
		case external.Git:
			logger.Verbose("Git repository detected")
			vR, err = repo.NewGitRepo(r)
			if err != nil {
				logger.Error("GitRepo Error: %v", err)
				return
			}
		case external.Svn:
			logger.Verbose("SVN repository detected")
		case external.Hg:
			logger.Verbose("Mercurial repository detected")
		default:
			logger.Error("Unknown repository type")
			return
		}
		logger.Timing("Creating VcsRepo took %s", time.Since(preVr))

		parseArgs := pkg.ParseArgs{
			PkgmetaFile: f.PkgmetaFile,
			PkgDir:      topDir,
		}
		pkgMeta, err := pkg.Parse(&parseArgs)
		if err != nil {
			logger.Error("Pkgmeta Error: %v", err)
			return
		}

		if pkgMeta.PackageAs != "" && projectName != pkgMeta.PackageAs {
			logger.Error("Project name (%s) from TOC filename(s) does not match `package-as` name in pkgmeta file (%s)", projectName, pkgMeta.PackageAs)
			return
		}

		logger.Verbose("%s", pkgMeta.String())

		if pkgMeta.PackageAs != "" {
			projectName = pkgMeta.PackageAs
		}

		copyLogGroup := logger.NewLogGroup("🗃️  Preparing Package Directory")
		logger.Debug("Top Directory: %s", topDir)
		logger.Debug("Top Directory from Flags: %s", f.TopDir)
		logger.Debug("Release Directory: %s", f.ReleaseDir)
		logger.Debug("Project Name: %s", projectName)
		packageDir, err := pkg.PreparePkgDir(projectName)
		logger.Debug("Package Directory: %s", packageDir)
		if err != nil {
			logger.Error("Error preparing package directory: %v", err)
			return
		}

		if !f.SkipCopy {
			projCopy := pkg.NewPkgCopy(topDir, packageDir, pkgMeta.Ignore, vR)
			err = projCopy.CopyToPackageDir(copyLogGroup)
			if err != nil {
				logger.Error("Copy Error: %v", err)
				return
			}
		}
		copyLogGroup.Flush(true)

		tokenMap := tokens.SimpleTokenMap{
			tokens.PackageName:      projectName,
			tokens.BuildTimestamp:   buildTimestampStr,
			tokens.BuildDate:        buildDate,
			tokens.BuildDateIso:     buildDateIso,
			tokens.BuildDateInteger: buildDateInteger,
		}
		github.Output(string(tokens.PackageName), tokenMap[tokens.PackageName])

		if classic {
			tokenMap[tokens.Classic] = "classic"
		} else {
			tokenMap[tokens.Classic] = ""
		}

		preGetInjectionValues := time.Now()
		if err = vR.GetInjectionValues(&tokenMap); err != nil {
			logger.Error("GetInjectionValues Error: %v", err)
			return
		}
		logger.Verbose("%s", tokenMap.String())
		i, err := injector.NewInjector(tokenMap, vR, packageDir)
		logger.Timing("Getting Injection Values took %s", time.Since(preGetInjectionValues))
		if err != nil {
			logger.Error("Injector Error: %v", err)
			return
		}
		err = i.Execute()
		if err != nil {
			logger.Error("Injector Execute Error: %v", err)
			return
		}

		if !f.SkipExternals {
			err = pkgMeta.FetchExternals(packageDir)
			if err != nil {
				logger.Error("Fetch Externals Error: %v", err)
				return
			}
		}

		github.Output(string(tokens.ProjectVersion), tokenMap[tokens.ProjectVersion])

		isNoLib := f.CreateNoLib || pkgMeta.EnableNoLibCreation

		if !f.SkipZip {
			zipsToCreate := 1
			if isNoLib {
				zipsToCreate++
			}
			var zipWGroup sync.WaitGroup
			zipErrChan := make(chan error, zipsToCreate)
			zipFileName := templateTokens.GetFileName(&tokenMap, false)
			zipFilePath := f.ReleaseDir + "/" + zipFileName + ".zip"
			noLibFileName := templateTokens.GetFileName(&tokenMap, true)
			z := zipper.NewZipper(packageDir)
			zipWGroup.Add(1)
			go func() {
				defer zipWGroup.Done()
				zipPath := f.ReleaseDir + "/" + zipFileName + ".zip"
				err = z.ZipFiles(packageDir, zipPath, []string{})
				if err != nil {
					zipErrChan <- err
					return
				}
				github.Output("main-zip-path", zipPath)
			}()

			if isNoLib && !templateTokens.HasNoLib {
				logger.Warn("Provided file and/or label template did not contain %s, but no-lib package requested. Skipping no-lib package since the zip name will not be unique.", tokens.NoLib.NormalizeTemplateToken())
				isNoLib = false
			}

			if isNoLib {
				dirsToExclude := pkgMeta.GetNoLibDirs(packageDir)
				zipWGroup.Add(1)
				go func() {
					defer zipWGroup.Done()
					zipPath := f.ReleaseDir + "/" + noLibFileName + ".zip"
					err = z.ZipFiles(packageDir, zipPath, dirsToExclude)
					if err != nil {
						zipErrChan <- err
						return
					}
					github.Output("nolib-zip-path", zipPath)
				}()
			}

			zipWGroup.Wait()
			close(zipErrChan)
			z.Complete()

			// Collect errors
			for err := range zipErrChan {
				if err != nil {
					logger.Error("Zip Error: %v", err)
					return
				}
			}

			curseArgs := upload.UploadCurseArgs{
				ZipPath:   zipFilePath,
				FileLabel: templateTokens.GetLabel(&tokenMap, false),
				TocFiles:  tocFiles,
				PkgMeta:   pkgMeta,
			}
			if err = upload.UploadToCurse(curseArgs); err != nil {
				logger.Error("Curse Upload Error: %v", err)
				return
			}
		}

		logger.TimingSummary()

		logger.WarningsEncountered()

		fmt.Println("")
		logger.Info("✨ Successfully packaged %s in ⏱️  %s", projectName, time.Since(start))
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().BoolVarP(&f.SkipCopy, "skipCopy", "c", false, "Skip copying the files to the output directory.")
	buildCmd.Flags().BoolVarP(&f.SkipUpload, "skipUpload", "d", false, "Skip uploading.")
	buildCmd.Flags().BoolVarP(&f.SkipExternals, "skipExternals", "e", false, "Skip fetching externals.")
	buildCmd.Flags().BoolVarP(&f.ForceExternals, "forceExternals", "E", false, "Force fetching externals, bypassing the cache.")
	buildCmd.Flags().BoolVarP(&f.SkipLocalization, "skipLocalization", "l", false, "Skip @localization@ keyword replacement.")
	buildCmd.Flags().BoolVarP(&f.OnlyLocalization, "onlyLocalization", "L", false, "Only do @localization@ keyword replacement (skip upload to CurseForge).")
	buildCmd.Flags().BoolVarP(&f.KeepPackageDir, "keepPackageDir", "o", false, "Keep existing package directory, overwriting its contents.")
	buildCmd.Flags().BoolVarP(&f.CreateNoLib, "createNoLib", "s", false, "Create a stripped-down \"nolib\" package.")
	buildCmd.Flags().BoolVarP(&f.SplitToc, "splitToc", "S", false, "Create a package supporting multiple game types from a single TOC file.")
	buildCmd.Flags().BoolVarP(&f.UnixLineEndings, "unixLineEndings", "u", false, "Use Unix line endings in TOC and XML files.")
	buildCmd.Flags().BoolVarP(&f.SkipZip, "skipZip", "z", false, "Skip zipping the package.")
	buildCmd.Flags().StringVarP(&f.TopDir, "topDir", "t", ".", "The top level directory of the addon")
	buildCmd.Flags().StringVarP(&f.ReleaseDir, "releaseDir", "r", f.TopDir+"/.release", "The directory to output the release files.")
	buildCmd.Flags().StringVarP(&f.CurseId, "curseId", "p", "", "Set the CurseForge project ID for localization and uploading. (Use 0 to unset the TOC value)")
	buildCmd.Flags().StringVarP(&f.WowiId, "wowiId", "w", "", "Set the WoWInterface project ID for uploading. (Use 0 to unset the TOC value)")
	buildCmd.Flags().StringVarP(&f.WagoId, "wagoId", "a", "", "Set the Wago project ID for uploading. (Use 0 to unset the TOC value)")
	buildCmd.Flags().StringVarP(&f.GameVersion, "gameVersion", "g", "", "Set the game version to use for uploading.")
	buildCmd.Flags().StringVarP(&f.PkgmetaFile, "pkgmetaFile", "m", "", "Set the pkgmeta file to use.")
	buildCmd.Flags().StringVarP(&f.NameTemplate, "nameTemplate", "n", "", "Set the name template to use for the release file. Use \"-n help\" for more info.")
}
