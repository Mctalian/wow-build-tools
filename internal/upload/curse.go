package upload

import (
	f "github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/toc"
)

func UploadToCurse(path string, tocFiles []*toc.Toc) error {
	if f.SkipUpload || f.OnlyLocalization {
		return nil
	}

	var curseId string
	if f.CurseId != "" {
		if f.CurseId == "0" {
			curseId = ""
		} else {
			curseId = f.CurseId
		}
	} else {
		curseId = tocFiles[0].CurseId
	}
	if curseId == "" {
		return nil
	}

	logger.Verbose("Uploading %s to CurseForge (project %s)", path, curseId)
	return nil
}
