package toc

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type Toc struct {
	Filepath  string
	Interface []int
	Title     string
	Notes     string
	Version   string
	Files     []string
	CurseId   string
	WowiId    string
	WagoId    string
	Flavor    GameFlavor
}

func (t *Toc) addGameVersionsFromToc() map[GameFlavor][]string {
	for _, interfaceVersion := range t.Interface {
		// Grab the right-most 2 digits for the patch version
		patchVersion := interfaceVersion % 100
		// Grab the middle 2 digits for the minor version
		minorVersion := (interfaceVersion / 100) % 100
		// Grab the left-most digits for the major version
		majorVersion := interfaceVersion / 10000

		flavor := getFlavorFromMajorVersion(majorVersion)
		AddGameVersion(flavor, fmt.Sprintf("%d.%d.%d", majorVersion, minorVersion, patchVersion))
	}

	return gameVersions
}

func TocFileToGameFlavor(noExt string) (flavor GameFlavor, suffix string) {
	if strings.Contains(noExt, "-") {
		postDash := strings.Split(noExt, "-")
		if len(postDash) > 1 {
			suffix = postDash[len(postDash)-1]
		}
	} else if strings.Contains(noExt, "_") {
		postUnderscore := strings.Split(noExt, "_")
		if len(postUnderscore) > 1 {
			suffix = postUnderscore[len(postUnderscore)-1]
		}
	}

	normalSuffix := strings.ToLower(suffix)

	switch normalSuffix {
	case "classic", "vanilla":
		flavor = ClassicEra
	case "tbc", "bcc":
		flavor = TbcClassic
	case "wrath", "wotlk", "wotlkc":
		flavor = WotlkClassic
	case "cata":
		flavor = CataClassic
	case "mop":
		flavor = MopClassic // Just a guess
	case "wod":
		flavor = WodClassic
	case "legion":
		flavor = LegionClassic
	case "bfa":
		flavor = BfaClassic
	case "sl":
		flavor = SlClassic
	case "df":
		flavor = DfClassic
	case "", "mainline":
		flavor = Retail
	default:
		flavor = Unknown
	}

	return
}

func FindTocFiles(path string) ([]string, error) {
	tocFiles := []string{}
	matches, err := filepath.Glob(path + "/*.toc")
	if err != nil {
		return tocFiles, fmt.Errorf("error finding TOC file in %s: %v", path, err)
	}

	if len(matches) == 0 {
		return tocFiles, fmt.Errorf("no TOC file found in %s", path)
	}

	tocFiles = append(tocFiles, matches...)

	slices.Sort(tocFiles)

	return tocFiles, nil
}

func DetermineProjectName(tocFiles []string) string {
	projectName := ""
	for _, tocFile := range tocFiles {
		tocFilePath := filepath.Base(tocFile)
		var flavor GameFlavor
		noExt := strings.TrimSuffix(tocFilePath, filepath.Ext(tocFilePath))

		if !strings.Contains(noExt, "-") && !strings.Contains(noExt, "_") {
			projectName = noExt

			break
		}

		flavor, suffix := TocFileToGameFlavor(noExt)
		if flavor != Unknown {
			projectName = strings.ReplaceAll(noExt, "_"+suffix, "")
			projectName = strings.ReplaceAll(projectName, "-"+suffix, "")
			break
		}

	}

	return projectName
}

func parse(filePath, tocContents string) (*Toc, error) {
	toc := &Toc{}
	toc.Filepath = filePath
	baseFilename := filepath.Base(filePath)
	toc.Flavor, _ = TocFileToGameFlavor(strings.TrimSuffix(baseFilename, filepath.Ext(baseFilename)))
	lines := strings.Split(tocContents, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## Interface:") {
			interfaceLine := strings.TrimPrefix(line, "## Interface:")
			interfaceLine = strings.TrimSpace(interfaceLine)
			interfaceValues := strings.Split(interfaceLine, ",")
			for _, interfaceValue := range interfaceValues {
				interfaceValue = strings.TrimSpace(interfaceValue)
				interfaceVersion, err := strconv.Atoi(interfaceValue)
				if err != nil {
					return nil, fmt.Errorf("error parsing Interface version: %v", err)
				}
				toc.Interface = append(toc.Interface, interfaceVersion)
			}
		} else if strings.HasPrefix(line, "## Title:") {
			toc.Title = strings.TrimPrefix(line, "## Title:")
			toc.Title = strings.TrimSpace(toc.Title)
		} else if strings.HasPrefix(line, "## Notes:") {
			toc.Notes = strings.TrimPrefix(line, "## Notes:")
			toc.Notes = strings.TrimSpace(toc.Notes)
		} else if strings.HasPrefix(line, "## Version:") {
			toc.Version = strings.TrimPrefix(line, "## Version:")
			toc.Version = strings.TrimSpace(toc.Version)
		} else if !strings.HasPrefix(line, "#") {
			file := strings.TrimSpace(line)
			if file == "" {
				continue
			}
			toc.Files = append(toc.Files, file)
		} else if strings.HasPrefix(line, "## X-Curse-Project-ID:") {
			toc.CurseId = strings.TrimPrefix(line, "## X-Curse-Project-ID:")
			toc.CurseId = strings.TrimSpace(toc.CurseId)
		} else if strings.HasPrefix(line, "## X-WoWI-ID:") {
			toc.WowiId = strings.TrimPrefix(line, "## X-WoWI-ID:")
			toc.WowiId = strings.TrimSpace(toc.WowiId)
		} else if strings.HasPrefix(line, "## X-Wago-ID:") {
			toc.WagoId = strings.TrimPrefix(line, "## X-Wago-ID:")
			toc.WagoId = strings.TrimSpace(toc.WagoId)
		}
	}

	toc.addGameVersionsFromToc()

	return toc, nil
}

func NewToc(path string) (*Toc, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading TOC file: %v", err)
	}

	toc, err := parse(path, string(contents))
	if err != nil {
		return nil, fmt.Errorf("error parsing TOC file: %v", err)
	}

	return toc, nil
}
