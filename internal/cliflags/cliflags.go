package cliflags

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/McTalian/wow-build-tools/internal/logger"
)

var SkipCopy bool
var SkipUpload bool
var SkipExternals bool
var ForceExternals bool
var SkipLocalization bool
var OnlyLocalization bool
var KeepPackageDir bool
var CreateNoLib bool
var SplitToc bool
var UnixLineEndings bool
var SkipZip bool
var TopDir string
var ReleaseDir string
var PackageDir string
var CurseId string
var WowiId string
var WagoId string
var GameVersion string
var GameVerList []string
var PkgmetaFile string
var NameTemplate string

var UploadInput string
var UploadLabel string
var UploadInterfaceVersions []int

func parseGameVersionSegment(version string) (string, error) {
	orig := strings.ToLower(version)
	switch strings.ToLower(orig) {
	case "retail", "classic", "bcc", "wrath", "cata":
		return orig, nil
	case "mainline":
		return "retail", nil
	default:
		segments := strings.Split(orig, ".")
		if len(segments) < 3 {
			return "", fmt.Errorf("Invalid argument for game version: %s", orig)
		}
		major, err := strconv.Atoi(segments[0])
		if err != nil {
			logger.Error("%v", err)
			return "", fmt.Errorf("Invalid argument for game version: %s", orig)
		}
		minor, err := strconv.Atoi(segments[1])
		if err != nil {
			logger.Error("%v", err)
			return "", fmt.Errorf("Invalid argument for game version: %s", orig)
		}
		patch, err := strconv.Atoi(segments[2])
		if err != nil {
			logger.Error("%v", err)
			return "", fmt.Errorf("Invalid argument for game version: %s", orig)
		}

		GameVerList = append(GameVerList, fmt.Sprintf("%d.%d.%d", major, minor, patch))

		switch major {
		case 1:
			return "classic", nil
		case 2:
			return "bcc", nil
		case 3:
			return "wrath", nil
		case 4:
			return "cata", nil
		default:
			return "retail", nil
		}
	}
}

func normalizeGameVersion() error {
	if GameVersion == "" {
		return nil
	}

	var versions []string
	if strings.Contains(GameVersion, ",") {
		// If it contains a comma, split it into multiple versions
		v := strings.Split(GameVersion, ",")
		for _, version := range v {
			versions = append(versions, strings.TrimSpace(version))
		}

		for _, version := range versions {
			_, err := parseGameVersionSegment(version)
			if err != nil {
				GameVersion = ""
				return err
			}
		}
	} else {
		// Only one version specified
		g, err := parseGameVersionSegment(GameVersion)
		if err != nil {
			GameVersion = ""
			return err
		}
		GameVersion = g
	}

	if len(GameVerList) > 0 {
		GameVersion = ""
	}

	return nil
}

func ValidateInputArgs() error {
	err := normalizeGameVersion()

	return err
}
