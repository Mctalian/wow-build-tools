package upload

import (
	"fmt"
	"os"

	f "github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/McTalian/wow-build-tools/internal/toc"
)

var ErrNoWowiId = fmt.Errorf("no WoW Interface ID provided")
var ErrMultipleWowiIds = fmt.Errorf("multiple WoW Interface IDs found")
var ErrNoWowiUpload = fmt.Errorf("WoW Interface upload is disabled")
var ErrNoWowiApiKey = fmt.Errorf("WOWI_API_TOKEN not set")

var wowiApiUrl = "https://api.wowinterface.com/addons/"
var wowiGameVersionsUrl = fmt.Sprintf("%scompatible.json", wowiApiUrl)
var wowiUploadUrl = fmt.Sprintf("%supdate", wowiApiUrl)

var wowiGameVersionsEntry struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Game      string `json:"game"`
	Interface string `json:"interface"`
	Default   bool   `json:"default"`
}

func locateWowiId(tocFiles []*toc.Toc) (wowiId string, err error) {
	var foundWowiId string
	for _, tocFile := range tocFiles {
		if tocFile.WowiId != "" {
			if foundWowiId != "" && foundWowiId != tocFile.WowiId {
				err = ErrMultipleWowiIds
				return
			}
			foundWowiId = tocFile.WowiId
		}
	}

	if foundWowiId == "" {
		err = ErrNoWowiId
		return
	}

	wowiId = foundWowiId
	return
}

func getWowiId(tocFiles []*toc.Toc) (wowiId string, err error) {
	if f.SkipUpload {
		err = ErrNoWowiUpload
		return
	}

	if f.WowiId != "" {
		if f.WowiId == "0" {
			wowiId = ""
		} else {
			wowiId = f.WowiId
		}
	} else {
		wowiId, err = locateWowiId(tocFiles)
		if err == ErrMultipleWowiIds {
			return
		}
	}
	if wowiId == "" {
		err = ErrNoWowiId
		return
	}

	return
}

type wowiUpload struct {
	token string
}

func (w *wowiUpload) lookupWowiToken() (err error) {
	token, found := os.LookupEnv("WOWI_API_TOKEN")
	if !found {
		err = ErrNoWowiApiKey
		return
	}

	w.token = token

	return
}
