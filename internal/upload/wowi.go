package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/McTalian/wow-build-tools/internal/changelog"
	f "github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/toc"
)

var ErrNoWowiId = fmt.Errorf("no WoW Interface ID provided")
var ErrMultipleWowiIds = fmt.Errorf("multiple WoW Interface IDs found")
var ErrNoWowiUpload = fmt.Errorf("WoW Interface upload is disabled")
var ErrNoWowiApiKey = fmt.Errorf("WOWI_API_TOKEN not set")

var wowiApiUrl = "https://api.wowinterface.com/addons/"
var wowiGameVersionsUrl = fmt.Sprintf("%scompatible.json", wowiApiUrl)
var wowiUploadUrl = fmt.Sprintf("%supdate", wowiApiUrl)

type wowiGameVersionsEntry struct {
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
	token       string
	projectId   string
	zipFile     string
	displayName string
	changelog   *changelog.Changelog
	compatible  []string
	version     string
	archiveOld  bool
	logGroup    *logger.LogGroup
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

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func (w *wowiUpload) validateGameVersions(gameVersions []string) error {
	resp, err := http.Get(wowiGameVersionsUrl)
	if err != nil {
		w.logGroup.Error("Could not fetch game versions: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.logGroup.Error("Could not fetch game versions: %v", err)
		return fmt.Errorf("could not fetch game versions: %v", err)
	}

	var versionResp []wowiGameVersionsEntry
	err = json.NewDecoder(resp.Body).Decode(&versionResp)
	if err != nil {
		w.logGroup.Error("Could not fetch game versions: %v", err)
		return err
	}

	versionIdList := make([]string, len(versionResp))
	for i, version := range versionResp {
		versionIdList[i] = version.Id
	}

	for _, gameVersion := range gameVersions {
		if !stringInSlice(gameVersion, versionIdList) {
			w.logGroup.Warn("Game version %s is not supported by WoW Interface, skipping", gameVersion)
		} else {
			w.compatible = append(w.compatible, gameVersion)
		}
	}

	return nil
}

func (w *wowiUpload) upload() error {
	w.logGroup.Info("Uploading to WoW Interface")

	file, err := os.Open(w.zipFile)
	if err != nil {
		return fmt.Errorf("could not open zip file: %v", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err = writer.WriteField("id", w.projectId); err != nil {
		return fmt.Errorf("could not write id: %v", err)
	}

	if err = writer.WriteField("version", w.version); err != nil {
		return fmt.Errorf("could not write version: %v", err)
	}

	if err = writer.WriteField("compatible", strings.Join(w.compatible, ",")); err != nil {
		return fmt.Errorf("could not write compatible: %v", err)
	}

	if !w.archiveOld {
		if err = writer.WriteField("archive", "No"); err != nil {
			return fmt.Errorf("could not write archive: %v", err)
		}
	}

	return nil
}

type UploadWowiArgs struct {
	TocFiles       []*toc.Toc
	ProjectVersion string
	ZipPath        string
	FileLabel      string
	Changelog      *changelog.Changelog
	ReleaseType    string
	WowiArchiveOld bool
}

func UploadToWowi(args UploadWowiArgs) error {
	logGroup := logger.NewLogGroup("🛜  Uploading to WoW Interface")
	defer logGroup.Flush(true)

	tocFiles := args.TocFiles

	wowiId, err := getWowiId(tocFiles)
	if err != nil {
		if err == ErrNoWowiId || err == ErrNoWowiUpload {
			logGroup.Verbose("Skipping WoW Interface upload")
			return nil
		}
		return err
	}

	wowiUpload := wowiUpload{
		projectId:   wowiId,
		zipFile:     args.ZipPath,
		displayName: args.FileLabel,
		changelog:   args.Changelog,
		version:     args.ProjectVersion,
		archiveOld:  args.WowiArchiveOld,
		logGroup:    logGroup,
	}

	if err := wowiUpload.lookupWowiToken(); err != nil {
		logGroup.Info("Skipping WoW Interface upload: %s", err)
		return nil
	}

	gameVersions := toc.GetGameVersions()

	if err := wowiUpload.validateGameVersions(gameVersions); err != nil {
		logGroup.Error("Could not validate game versions: %v", err)
		return err
	}

	if err := wowiUpload.upload(); err != nil {
		return err
	}

	return nil
}
