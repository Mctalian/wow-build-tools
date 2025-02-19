/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/McTalian/wow-build-tools/internal/cachedir"
	"github.com/McTalian/wow-build-tools/internal/changelog"
	f "github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/McTalian/wow-build-tools/internal/external"
	"github.com/McTalian/wow-build-tools/internal/github"
	"github.com/McTalian/wow-build-tools/internal/injector"
	"github.com/McTalian/wow-build-tools/internal/license"
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
	Long:  `This command packages the addon as specified via a pkgmeta file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()

		err := toc.ParseGameVersionFlag()
		if err != nil {
			logger.Error("Error validating game version input argument: %v", err)
			return err
		}

		var templateTokens *tokens.NameTemplate
		if f.NameTemplate == "help" {
			logger.Info("%s", tokens.NameTemplateUsageInfo())
			return nil
		} else {
			templateTokens, err = tokens.NewNameTemplate(f.NameTemplate)
			if err != nil {
				logger.Error("Error parsing name template: %v", err)
				return err
			}
		}

		timeNow := time.Now()
		timeNowUtc := timeNow.UTC()
		buildTimestamp := timeNow.Unix()
		buildTimestampStr := strconv.FormatInt(buildTimestamp, 10)
		buildDate := timeNowUtc.Format("2006-01-02")
		buildDateIso := timeNowUtc.Format("2006-01-02T15:04:05Z")
		buildDateInteger := timeNowUtc.Format("20060102150405")
		buildYear := timeNowUtc.Format("2006")
		topDir := f.TopDir
		if cmd.Flags().Changed("topDir") && !cmd.Flags().Changed("releaseDir") {
			f.ReleaseDir = topDir + "/.release"
		}

		if _, err := cachedir.Create(); err != nil {
			logger.Error("Cache Error: %v", err)
			return err
		}

		tocFilePaths, err := toc.FindTocFiles(topDir)
		if err != nil {
			logger.Error("TOC Error: %v", err)
			return err
		}

		logger.Verbose("TOC Files: %v", tocFilePaths)

		projectName := toc.DetermineProjectName(tocFilePaths)

		logger.Info("🔨 Building %s...", projectName)

		var tocFiles []*toc.Toc
		for _, tocFilePath := range tocFilePaths {
			t, err := toc.NewToc(tocFilePath)
			if err != nil {
				logger.Error("TOC Error: %v", err)
				return err
			}
			tocFiles = append(tocFiles, t)

			logger.Verbose("%s, %s", t.Filepath, t.Flavor.ToString())
		}

		r, err := repo.NewRepo(topDir)
		if err != nil {
			logger.Error("Repo Error: %v", err)
			return err
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
				return err
			}
		case external.Svn:
			logger.Verbose("SVN repository detected")
		case external.Hg:
			logger.Verbose("Mercurial repository detected")
		default:
			logger.Error("Unknown repository type")
			return err
		}
		logger.Timing("Creating VcsRepo took %s", time.Since(preVr))

		parseArgs := pkg.ParseArgs{
			PkgmetaFile: f.PkgmetaFile,
			PkgDir:      topDir,
		}
		pkgMeta, err := pkg.Parse(&parseArgs)
		if err != nil {
			logger.Error("Pkgmeta Error: %v", err)
			return err
		}

		if pkgMeta.PackageAs != "" && projectName != pkgMeta.PackageAs {
			err = fmt.Errorf("Project name (%s) from TOC filename(s) does not match `package-as` name in pkgmeta file (%s)", projectName, pkgMeta.PackageAs)
			return err
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
			return err
		}

		err = license.EnsureLicensePresent(pkgMeta.License, topDir, packageDir, f.CurseId)
		if err != nil {
			logger.Error("License Error: %v", err)
			return err
		}

		if !f.SkipCopy {
			projCopy := pkg.NewPkgCopy(topDir, packageDir, pkgMeta.Ignore, vR)
			err = projCopy.CopyToPackageDir(copyLogGroup)
			if err != nil {
				logger.Error("Copy Error: %v", err)
				return err
			}
		}
		copyLogGroup.Flush(true)

		tokenMap := tokens.SimpleTokenMap{
			tokens.PackageName:      projectName,
			tokens.BuildTimestamp:   buildTimestampStr,
			tokens.BuildDate:        buildDate,
			tokens.BuildDateIso:     buildDateIso,
			tokens.BuildDateInteger: buildDateInteger,
			tokens.BuildYear:        buildYear,
		}
		err = github.Output(string(tokens.PackageName), tokenMap[tokens.PackageName])
		if err != nil {
			logger.Error("Output Error: %v", err)
			return err
		}

		flags := tokens.FlagMap{
			tokens.NoLibFlag:   "",
			tokens.AlphaFlag:   "",
			tokens.BetaFlag:    "",
			tokens.ClassicFlag: "",
		}

		preGetInjectionValues := time.Now()
		if err = vR.GetInjectionValues(&tokenMap); err != nil {
			logger.Error("GetInjectionValues Error: %v", err)
			return err
		}

		var releaseType string
		bTTM := tokens.BuildTypeTokenMap{
			tokens.Alpha:         false,
			tokens.Beta:          false,
			tokens.Classic:       false,
			tokens.Debug:         false,
			tokens.Retail:        false,
			tokens.VersionRetail: false,
			tokens.VersionBcc:    false,
			tokens.VersionWrath:  false,
			tokens.VersionCata:   false,
		}
		tag := vR.GetCurrentTag()
		if tag != "" {
			if strings.Contains(tag, "alpha") {
				flags[tokens.AlphaFlag] = "-alpha"
				bTTM[tokens.Alpha] = true
				bTTM[tokens.Beta] = false
				releaseType = "alpha"
			} else if strings.Contains(tag, "beta") {
				flags[tokens.BetaFlag] = "-beta"
				bTTM[tokens.Alpha] = false
				bTTM[tokens.Beta] = true
				releaseType = "beta"
			} else {
				bTTM[tokens.Alpha] = false
				bTTM[tokens.Beta] = false
				releaseType = "release"
			}
		} else {
			flags[tokens.AlphaFlag] = "-alpha"
			bTTM[tokens.Alpha] = true
			bTTM[tokens.Beta] = false
			releaseType = "alpha"
		}
		flavors := toc.GetGameFlavors()
		if len(flavors) == 1 {
			switch flavors[0] {
			case toc.Retail:
				bTTM[tokens.Retail] = true
				bTTM[tokens.VersionRetail] = true
			case toc.ClassicEra:
				flags[tokens.ClassicFlag] = "-classic"
				bTTM[tokens.Classic] = true
			case toc.TbcClassic:
				bTTM[tokens.VersionBcc] = true
			case toc.WotlkClassic:
				bTTM[tokens.VersionWrath] = true
			case toc.CataClassic:
				bTTM[tokens.VersionCata] = true
			case toc.MopClassic:
				bTTM[tokens.VersionMop] = true
			case toc.WodClassic:
				bTTM[tokens.VersionWod] = true
			case toc.LegionClassic:
				bTTM[tokens.VersionLegion] = true
			case toc.BfaClassic:
				bTTM[tokens.VersionBfa] = true
			case toc.SlClassic:
				bTTM[tokens.VersionSl] = true
			case toc.DfClassic:
				bTTM[tokens.VersionDf] = true
			default:
				bTTM[tokens.Retail] = true
			}
		}
		// TODO: Handle multiple game versions

		logger.Verbose("%s", tokenMap.String())
		i, err := injector.NewInjector(tokenMap, vR, packageDir, bTTM)
		logger.Timing("Getting Injection Values took %s", time.Since(preGetInjectionValues))
		if err != nil {
			logger.Error("Injector Error: %v", err)
			return err
		}

		var changelogTitle string
		if pkgMeta.ChangelogTitle != "" {
			changelogTitle = pkgMeta.ChangelogTitle
		} else {
			changelogTitle = projectName
		}
		cl, err := changelog.NewChangelog(vR, pkgMeta, changelogTitle, packageDir, topDir)
		if err != nil {
			logger.Error("Changelog Error: %v", err)
			return err
		}
		err = cl.GetChangelog()
		if err != nil {
			logger.Error("GetChangelog Error: %v", err)
			return err
		}

		err = i.Execute()
		if err != nil {
			logger.Error("Injector Execute Error: %v", err)
			return err
		}

		if !f.SkipExternals {
			err = pkgMeta.FetchExternals(packageDir)
			if err != nil {
				logger.Error("Fetch Externals Error: %v", err)
				return err
			}
		}

		err = github.Output(string(tokens.ProjectVersion), tokenMap[tokens.ProjectVersion])
		if err != nil {
			logger.Error("Output Error: %v", err)
			return err
		}

		isNoLib := f.CreateNoLib || pkgMeta.EnableNoLibCreation

		if !f.SkipZip {
			if err != nil {
				logger.Error("Changelog Error: %v", err)
				return err
			}

			zipsToCreate := 1
			if isNoLib {
				zipsToCreate++
			}
			var zipWGroup sync.WaitGroup
			zipErrChan := make(chan error, zipsToCreate)

			zipFileName := templateTokens.GetFileName(&tokenMap, flags)
			zipFilePath := f.ReleaseDir + "/" + zipFileName + ".zip"
			flags[tokens.NoLibFlag] = "-nolib"
			noLibFileName := templateTokens.GetFileName(&tokenMap, flags)
			z := zipper.NewZipper(packageDir)
			zipWGroup.Add(1)
			go func() {
				defer zipWGroup.Done()
				zipPath := f.ReleaseDir + "/" + zipFileName + ".zip"
				err = z.ZipFiles(packageDir, zipPath)
				if err != nil {
					zipErrChan <- err
					return
				}
				err = github.Output("main-zip-path", zipPath)
				if err != nil {
					zipErrChan <- err
					return
				}
			}()

			if isNoLib && !templateTokens.HasNoLib {
				logger.Warn("Provided file and/or label template did not contain %s, but no-lib package requested. Skipping no-lib package since the zip name will not be unique.", tokens.NoLibFlag.NormalizeTemplateToken())
				isNoLib = false
			}

			if isNoLib {
				dirsToExclude := pkgMeta.GetNoLibDirs(packageDir)
				zipWGroup.Add(1)
				go func() {
					defer zipWGroup.Done()
					zipPath := f.ReleaseDir + "/" + noLibFileName + ".zip"
					err = z.ZipFiles(packageDir, zipPath, dirsToExclude, i.NoLibStripFiles)
					if err != nil {
						zipErrChan <- err
						return
					}
					err = github.Output("nolib-zip-path", zipPath)
					if err != nil {
						zipErrChan <- err
						return
					}
				}()
			}

			zipWGroup.Wait()
			close(zipErrChan)
			z.Complete()

			// Collect errors
			for err := range zipErrChan {
				if err != nil {
					logger.Error("Zip Error: %v", err)
					return err
				}
			}
			flags[tokens.NoLibFlag] = ""

			if !f.SkipUpload {
				uploadsToAttempt := 3
				var uploadWGroup sync.WaitGroup
				uploadErrChan := make(chan error, uploadsToAttempt)
				uploadWGroup.Add(uploadsToAttempt)

				go func() {
					defer uploadWGroup.Done()
					curseArgs := upload.UploadCurseArgs{
						ZipPath:     zipFilePath,
						FileLabel:   templateTokens.GetLabel(&tokenMap, flags),
						TocFiles:    tocFiles,
						PkgMeta:     pkgMeta,
						Changelog:   cl,
						ReleaseType: releaseType,
					}
					if err = upload.UploadToCurse(curseArgs); err != nil {
						logger.Error("Curse Upload Error: %v", err)
						uploadErrChan <- err
						return
					}
				}()

				go func() {
					defer uploadWGroup.Done()
					wowiArgs := upload.UploadWowiArgs{
						TocFiles:       tocFiles,
						ProjectVersion: tokenMap[tokens.ProjectVersion],
						ZipPath:        zipFilePath,
						FileLabel:      templateTokens.GetLabel(&tokenMap, flags),
						Changelog:      cl,
						ReleaseType:    releaseType,
					}
					if err = upload.UploadToWowi(wowiArgs); err != nil {
						logger.Error("WoW Interface Upload Error: %v", err)
						uploadErrChan <- err
						return
					}
				}()

				go func() {
					defer uploadWGroup.Done()
					wagoArgs := upload.UploadWagoArgs{
						ZipPath:     zipFilePath,
						FileLabel:   templateTokens.GetLabel(&tokenMap, flags),
						TocFiles:    tocFiles,
						Changelog:   cl,
						ReleaseType: releaseType,
					}
					if err = upload.UploadToWago(wagoArgs); err != nil {
						logger.Error("Wago Upload Error: %v", err)
						uploadErrChan <- err
						return
					}
				}()

				uploadWGroup.Wait()
				close(uploadErrChan)

				// Collect errors
				for err := range uploadErrChan {
					if err != nil {
						logger.Error("Upload Error: %v", err)
						return err
					}
				}
			}
		}

		logger.TimingSummary()

		logger.WarningsEncountered()

		fmt.Println("")
		logger.Info("✨ Successfully packaged %s in ⏱️  %s", projectName, time.Since(start))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().SortFlags = false

	buildCmd.Flags().StringVarP(&f.TopDir, "topDir", "t", ".", "The top level directory of the addon")
	buildCmd.Flags().StringVarP(&f.ReleaseDir, "releaseDir", "r", f.TopDir+"/.release", "The directory to output the release files.")
	buildCmd.Flags().StringVarP(&f.PkgmetaFile, "pkgmetaFile", "m", "", "Set the pkgmeta file to use. (Defaults to {topDir}/pkgmeta.yml, {topDir}/pkgmeta.yaml, or {topDir}/.pkgmeta if one exists.)")
	buildCmd.Flags().BoolVarP(&f.KeepPackageDir, "keepPackageDir", "o", false, "Keep existing package directory, overwriting its contents.")
	buildCmd.Flags().BoolVarP(&f.CreateNoLib, "createNoLib", "s", false, "Create a stripped-down \"nolib\" package.")
	buildCmd.Flags().StringVarP(&f.CurseId, "curseId", "p", "", "Set the CurseForge project ID for localization and uploading. (Use 0 to unset the TOC value)")
	buildCmd.Flags().StringVarP(&f.WowiId, "wowiId", "w", "", "Set the WoWInterface project ID for uploading. (Use 0 to unset the TOC value)")
	buildCmd.Flags().StringVarP(&f.WagoId, "wagoId", "a", "", "Set the Wago project ID for uploading. (Use 0 to unset the TOC value)")
	buildCmd.Flags().BoolVarP(&f.SkipCopy, "skipCopy", "c", false, "Skip copying the files to the output directory.")
	buildCmd.Flags().BoolVarP(&f.SkipExternals, "skipExternals", "e", false, "Skip fetching externals.")
	buildCmd.Flags().BoolVarP(&f.ForceExternals, "forceExternals", "E", false, "Force fetching externals, bypassing the cache.")
	buildCmd.Flags().BoolVarP(&f.SkipZip, "skipZip", "z", false, "Skip zipping the package (and uploading).")
	buildCmd.Flags().BoolVarP(&f.SkipUpload, "skipUpload", "d", false, "Skip uploading.")
	buildCmd.Flags().StringVarP(&f.NameTemplate, "nameTemplate", "n", "", "Set the name template to use for the release file. Use \"-n help\" for more info.")
	buildCmd.Flags().BoolVarP(&f.SkipLocalization, "skipLocalization", "l", false, "Skip @localization@ keyword replacement.")
	buildCmd.Flags().BoolVarP(&f.OnlyLocalization, "onlyLocalization", "L", false, "Only do @localization@ keyword replacement (skip upload to CurseForge).")
	buildCmd.Flags().BoolVarP(&f.SplitToc, "splitToc", "S", false, "Create a package supporting multiple game types from a single TOC file.")
	buildCmd.Flags().BoolVarP(&f.UnixLineEndings, "unixLineEndings", "u", false, "Use Unix line endings in TOC and XML files.")
	buildCmd.Flags().StringVarP(&f.GameVersion, "gameVersion", "g", "", "Set the game version to use for uploading.")
}
