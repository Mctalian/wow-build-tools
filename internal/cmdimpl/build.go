package cmdimpl

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/McTalian/wow-build-tools/internal/changelog"
	"github.com/McTalian/wow-build-tools/internal/configdir"
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

type BuildArgs struct {
	TopDir      string
	ReleaseDir  string
	PkgmetaFile string
	GameVersion string

	WatchMode    bool
	LevelVerbose bool
	LevelDebug   bool

	CurseId string
	WagoId  string
	WowiId  string

	SkipChangelog    bool
	SkipCopy         bool
	SkipExternals    bool
	SkipUpload       bool
	SkipLocalization bool
	SkipZip          bool

	ForceExternals   bool
	OnlyLocalization bool

	CreateNoLib     bool
	KeepPackageDir  bool
	NameTemplate    string
	SplitToc        bool
	UnixLineEndings bool
}

// Build is the implementation of the build command.
func Build(args *BuildArgs) error {
	start := time.Now()
	l := logger.DefaultLogger
	defer l.Clear()

	if args.WatchMode {
		l.SetLogLevel(logger.WARN)
	} else if args.LevelVerbose {
		l.SetLogLevel(logger.VERBOSE)
	} else if args.LevelDebug {
		l.SetLogLevel(logger.DEBUG)
	} else {
		l.SetLogLevel(logger.INFO)
	}

	err := toc.ParseGameVersionFlag(args.GameVersion)
	if err != nil {
		l.Error("Error validating game version input argument: %v", err)
		return err
	}

	var templateTokens *tokens.NameTemplate
	if args.NameTemplate == "help" {
		l.Info("%s", tokens.NameTemplateUsageInfo())
		return nil
	} else {
		templateTokens, err = tokens.NewNameTemplate(args.NameTemplate)
		if err != nil {
			l.Error("Error parsing name template: %v", err)
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
	topDir := args.TopDir

	if _, err := configdir.CreateExternalsCache(); err != nil {
		l.Error("Cache Error: %v", err)
		return err
	}

	tocFilePaths, err := toc.FindTocFiles(topDir)
	if err != nil {
		l.Error("TOC Error: %v", err)
		return err
	}

	l.Verbose("TOC Files: %v", tocFilePaths)

	projectName := toc.DetermineProjectName(tocFilePaths)

	l.Info("🔨 Building %s...", projectName)

	var tocFiles []*toc.Toc
	for _, tocFilePath := range tocFilePaths {
		t, err := toc.NewToc(tocFilePath)
		if err != nil {
			l.Error("TOC Error: %v", err)
			return err
		}
		tocFiles = append(tocFiles, t)

		l.Verbose("%s, %s", t.Filepath, t.Flavor.ToString())
	}

	r, err := repo.NewRepo(topDir)
	if err != nil {
		l.Error("Repo Error: %v", err)
		return err
	}

	l.Debug("%s", r.String())

	preVr := time.Now()
	var vR repo.VcsRepo
	switch r.GetVcsType() {
	case external.Git:
		l.Verbose("Git repository detected")
		vR, err = repo.NewGitRepo(r)
		if err != nil {
			l.Error("GitRepo Error: %v", err)
			return err
		}
	case external.Svn:
		l.Verbose("SVN repository detected")
	case external.Hg:
		l.Verbose("Mercurial repository detected")
	default:
		l.Error("Unknown repository type")
		return err
	}
	l.Timing("Creating VcsRepo took %s", time.Since(preVr))

	parseArgs := pkg.ParseArgs{
		PkgmetaFile: args.PkgmetaFile,
		PkgDir:      topDir,
	}
	pkgMeta, err := pkg.Parse(&parseArgs)
	if err != nil {
		l.Error("Pkgmeta Error: %v", err)
		return err
	}

	if pkgMeta.PackageAs != "" && projectName != pkgMeta.PackageAs {
		err = fmt.Errorf("Project name (%s) from TOC filename(s) does not match `package-as` name in pkgmeta file (%s)", projectName, pkgMeta.PackageAs)
		return err
	}

	l.Verbose("%s", pkgMeta.String())

	if pkgMeta.PackageAs != "" {
		projectName = pkgMeta.PackageAs
	}

	copyLogGroup := logger.NewLogGroup("🗃️  Preparing Package Directory")
	l.Debug("Top Directory: %s", topDir)
	l.Debug("Top Directory from Flags: %s", args.TopDir)
	l.Debug("Release Directory: %s", args.ReleaseDir)
	l.Debug("Project Name: %s", projectName)
	packageDir, err := pkg.PreparePkgDir(projectName, args.ReleaseDir, args.KeepPackageDir)
	l.Debug("Package Directory: %s", packageDir)
	if err != nil {
		l.Error("Error preparing package directory: %v", err)
		return err
	}

	err = license.EnsureLicensePresent(pkgMeta.License, topDir, packageDir, args.CurseId)
	if err != nil {
		l.Error("License Error: %v", err)
		return err
	}

	if !args.SkipCopy {
		projCopy := pkg.NewPkgCopy(topDir, packageDir, pkgMeta.Ignore, vR)
		err = projCopy.CopyToPackageDir(copyLogGroup)
		if err != nil {
			l.Error("Copy Error: %v", err)
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
		l.Error("Output Error: %v", err)
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
		l.Error("GetInjectionValues Error: %v", err)
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

	l.Verbose("%s", tokenMap.String())
	i, err := injector.NewInjector(tokenMap, vR, packageDir, bTTM, args.UnixLineEndings)
	l.Timing("Getting Injection Values took %s", time.Since(preGetInjectionValues))
	if err != nil {
		l.Error("Injector Error: %v", err)
		return err
	}

	var changelogTitle string
	if pkgMeta.ChangelogTitle != "" {
		changelogTitle = pkgMeta.ChangelogTitle
	} else {
		changelogTitle = projectName
	}

	var cl *changelog.Changelog
	if args.SkipChangelog {
		cl = &changelog.Changelog{}
	} else {
		cl, err = changelog.NewChangelog(vR, pkgMeta, changelogTitle, packageDir, topDir)
		if err != nil {
			l.Error("Changelog Error: %v", err)
			return err
		}
		err = cl.GetChangelog()
		if err != nil {
			l.Error("GetChangelog Error: %v", err)
			return err
		}
		defer cl.Cleanup()
	}

	err = i.Execute()
	if err != nil {
		l.Error("Injector Execute Error: %v", err)
		return err
	}

	if !args.SkipExternals {
		err = pkgMeta.FetchExternals(packageDir, args.ForceExternals)
		if err != nil {
			l.Error("Fetch Externals Error: %v", err)
			return err
		}
	}

	err = github.Output(string(tokens.ProjectVersion), tokenMap[tokens.ProjectVersion])
	if err != nil {
		l.Error("Output Error: %v", err)
		return err
	}

	isNoLib := (args.CreateNoLib || pkgMeta.EnableNoLibCreation) && !args.WatchMode

	if !args.SkipZip {
		zipsToCreate := 1
		if isNoLib {
			zipsToCreate++
		}
		var zipWGroup sync.WaitGroup
		zipErrChan := make(chan error, zipsToCreate)

		zipFileName := templateTokens.GetFileName(&tokenMap, flags)
		zipFilePath := filepath.Join(args.ReleaseDir, zipFileName+".zip")
		flags[tokens.NoLibFlag] = "-nolib"
		noLibFileName := templateTokens.GetFileName(&tokenMap, flags)
		z := zipper.NewZipper(packageDir, args.ReleaseDir, topDir, args.UnixLineEndings)
		zipWGroup.Add(1)
		go func() {
			defer zipWGroup.Done()
			zipPath := zipFilePath
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
			l.Warn("Provided file and/or label template did not contain %s, but no-lib package requested. Skipping no-lib package since the zip name will not be unique.", tokens.NoLibFlag.NormalizeTemplateToken())
			isNoLib = false
		}

		if isNoLib {
			dirsToExclude := pkgMeta.GetNoLibDirs(packageDir)
			zipWGroup.Add(1)
			go func() {
				defer zipWGroup.Done()
				zipPath := filepath.Join(args.ReleaseDir, noLibFileName+".zip")
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
				l.Error("Zip Error: %v", err)
				return err
			}
		}
		flags[tokens.NoLibFlag] = ""

		if !args.SkipUpload && !args.WatchMode {
			uploadsToAttempt := 4
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
					l.Error("Curse Upload Error: %v", err)
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
					l.Error("WoW Interface Upload Error: %v", err)
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
					l.Error("Wago Upload Error: %v", err)
					uploadErrChan <- err
					return
				}
			}()

			go func() {
				defer uploadWGroup.Done()
				githubArgs := upload.UploadGitHubArgs{
					ZipPaths:       []string{zipFilePath},
					ProjectName:    projectName,
					ProjectVersion: tokenMap[tokens.ProjectVersion],
					Repo:           vR,
					Changelog:      cl,
					ReleaseType:    releaseType,
				}
				if isNoLib {
					githubArgs.ZipPaths = append(githubArgs.ZipPaths, filepath.Join(args.ReleaseDir, noLibFileName+".zip"))
				}

				if err = upload.UploadToGitHub(githubArgs); err != nil {
					l.Error("GitHub Upload Error: %v", err)
					uploadErrChan <- err
					return
				}
			}()

			uploadWGroup.Wait()
			close(uploadErrChan)

			// Collect errors
			for err := range uploadErrChan {
				if err != nil {
					l.Error("Upload Error: %v", err)
					return err
				}
			}
		}
	}

	l.TimingSummary()

	l.WarningsEncountered()

	fmt.Println("")
	l.Success("✨ Successfully packaged %s in ⏱️  %s", projectName, time.Since(start))
	return nil
}
