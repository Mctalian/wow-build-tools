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
var PkgmetaFile string
var NameTemplate string

var UploadInput string
var UploadLabel string
var UploadInterfaceVersions []int

func normalizeGameVersion() error {
	orig := strings.ToLower(GameVersion)
	switch strings.ToLower(orig) {
	case "retail", "classic", "bcc", "wrath", "cata":
		GameVersion = orig
	case "mainline":
		GameVersion = "retail"
	default:
		segments := strings.Split(orig, ".")
		if len(segments) < 3 {
			return fmt.Errorf("Invalid argument for game version: %s", orig)
		}
		major, err := strconv.Atoi(segments[0])
		if err != nil {
			logger.Error("%v", err)
			return fmt.Errorf("Invalid argument for game version: %s", orig)
		}

		switch major {
		case 1:
			GameVersion = "classic"
		case 2:
			GameVersion = "bcc"
		case 3:
			GameVersion = "wrath"
		case 4:
			GameVersion = "cata"
		default:
			GameVersion = "retail"
		}
	}

	return nil
}

func ValidateInputArgs() error {
	err := normalizeGameVersion()

	return err
}
