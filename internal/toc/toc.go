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

type GameFlavor int

const (
	Unknown GameFlavor = iota
	ClassicEra
	TbcClassic
	WotlkClassic
	CataClassic
	MopClassic // Just a guess
	WodClassic
	LegionClassic
	BfaClassic
	SlClassic
	DfClassic
	Mainline
)

func (g GameFlavor) ToString() string {
	switch g {
	case ClassicEra:
		return "Classic"
	case TbcClassic:
		return "TBC"
	case WotlkClassic:
		return "WotLK"
	case CataClassic:
		return "Cata"
	case MopClassic: // Just a guess
		return "MoP"
	case WodClassic:
		return "WoD"
	case LegionClassic:
		return "Legion"
	case BfaClassic:
		return "BfA"
	case SlClassic:
		return "SL"
	case DfClassic:
		return "DF"
	default:
		return "Mainline"
	}
}

func TocFileToGameFlavor(noExt string) GameFlavor {
	var suffix string
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
		return ClassicEra
	case "tbc", "bcc":
		return TbcClassic
	case "wrath", "wotlk":
		return WotlkClassic
	case "cata":
		return CataClassic
	case "mop":
		return MopClassic // Just a guess
	case "wod":
		return WodClassic
	case "legion":
		return LegionClassic
	case "bfa":
		return BfaClassic
	case "sl":
		return SlClassic
	case "df":
		return DfClassic
	case "", "mainline":
		return Mainline
	default:
		return Unknown
	}
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
		var flavor GameFlavor = Unknown
		noExt := strings.TrimSuffix(tocFilePath, filepath.Ext(tocFilePath))

		if !strings.Contains(noExt, "-") && !strings.Contains(noExt, "_") {
			projectName = noExt

			break
		}

		flavor = TocFileToGameFlavor(noExt)
		if flavor != Unknown {
			projectName = strings.ReplaceAll(noExt, "_"+flavor.ToString(), "")
			projectName = strings.ReplaceAll(projectName, "-"+flavor.ToString(), "")
			break
		}

	}

	return projectName
}

func parse(filePath, tocContents string) (*Toc, error) {
	toc := &Toc{}
	toc.Filepath = filePath
	baseFilename := filepath.Base(filePath)
	toc.Flavor = TocFileToGameFlavor(strings.TrimSuffix(baseFilename, filepath.Ext(baseFilename)))
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
