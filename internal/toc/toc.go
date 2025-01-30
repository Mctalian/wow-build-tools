package toc

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Toc struct {
	Interface []int
	Title     string
	Notes     string
	Version   string
	Files     []string
}

type GameFlavors int

const (
	Unknown GameFlavors = iota
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

func (g GameFlavors) ToString() string {
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

func TocFileToGameFlavor(suffix string) GameFlavors {
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

	return tocFiles, nil
}

func DetermineProjectName(tocFiles []string) string {
	projectName := ""
	for _, tocFile := range tocFiles {
		tocFile = filepath.Base(tocFile)
		var flavor GameFlavors = Unknown
		noExt := strings.TrimSuffix(tocFile, filepath.Ext(tocFile))

		if !strings.Contains(noExt, "-") && !strings.Contains(noExt, "_") {
			projectName = noExt
			break
		}

		if strings.Contains(noExt, "-") {
			postDash := strings.Split(noExt, "-")
			if len(postDash) > 1 {
				flavor = TocFileToGameFlavor(postDash[len(postDash)-1])
			}
			if flavor != Unknown {
				projectName = strings.TrimSuffix(noExt, "-"+postDash[len(postDash)-1])
				break
			}
		}

		if strings.Contains(noExt, "_") {
			postUnderscore := strings.Split(noExt, "_")
			if len(postUnderscore) > 1 {
				flavor = TocFileToGameFlavor(postUnderscore[len(postUnderscore)-1])
			}
			if flavor != Unknown {
				projectName = strings.TrimSuffix(noExt, "_"+postUnderscore[len(postUnderscore)-1])
				break
			}
		}

	}

	return projectName
}
