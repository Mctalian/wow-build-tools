package toc

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/McTalian/wow-build-tools/internal/cliflags"
	"github.com/McTalian/wow-build-tools/internal/logger"
)

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
	Retail
)

func (g GameFlavor) ToString() string {
	switch g {
	case ClassicEra:
		return "classic"
	case TbcClassic:
		return "bcc"
	case WotlkClassic:
		return "wrath"
	case CataClassic:
		return "cata"
	case MopClassic: // Just a guess
		return "mop"
	case WodClassic:
		return "wod"
	case LegionClassic:
		return "legion"
	case BfaClassic:
		return "bfa"
	case SlClassic:
		return "sl"
	case DfClassic:
		return "df"
	default:
		return "retail"
	}
}

type GameVersions map[GameFlavor][]string

var gameVersions GameVersions = make(GameVersions)
var gameInterfaces GameVersions = make(GameVersions)

func AddGameInterface(flavor GameFlavor, version string) {
	gameInterfaces[flavor] = append(gameInterfaces[flavor], version)
}

func AddGameVersion(flavor GameFlavor, version string) {
	gameVersions[flavor] = append(gameVersions[flavor], version)
}

func getFlavorFromMajorVersion(majorVersion int) GameFlavor {
	switch majorVersion {
	case 1:
		return ClassicEra
	case 2:
		return TbcClassic
	case 3:
		return WotlkClassic
	case 4:
		return CataClassic
	case 5:
		return MopClassic // Just a guess
	case 6:
		return WodClassic
	case 7:
		return LegionClassic
	case 8:
		return BfaClassic
	case 9:
		return SlClassic
	case 10:
		return DfClassic
	default:
		return Retail
	}
}

func parseGameVersionSegment(version string) error {
	orig := strings.ToLower(version)
	switch strings.ToLower(orig) {
	case "retail", "classic", "bcc", "wrath", "cata":
		return nil
	case "mainline":
		return nil
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
		minor, err := strconv.Atoi(segments[1])
		if err != nil {
			logger.Error("%v", err)
			return fmt.Errorf("Invalid argument for game version: %s", orig)
		}
		patch, err := strconv.Atoi(segments[2])
		if err != nil {
			logger.Error("%v", err)
			return fmt.Errorf("Invalid argument for game version: %s", orig)
		}

		flavor := getFlavorFromMajorVersion(major)

		AddGameVersion(flavor, fmt.Sprintf("%d.%d.%d", major, minor, patch))
		AddGameInterface(flavor, fmt.Sprintf("%d%02d%02d", major, minor, patch))
	}

	return nil
}

func normalizeGameVersion(gameVersion string) error {
	if gameVersion == "" {
		return nil
	}

	var versions []string
	if strings.Contains(gameVersion, ",") {
		// If it contains a comma, split it into multiple versions
		v := strings.Split(gameVersion, ",")
		for _, version := range v {
			versions = append(versions, strings.TrimSpace(version))
		}

		for _, version := range versions {
			err := parseGameVersionSegment(version)
			if err != nil {
				return err
			}
		}
	} else {
		// Only one version specified
		err := parseGameVersionSegment(gameVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func ParseGameVersionFlag() error {
	return normalizeGameVersion(cliflags.GameVersion)
}

func GetGameFlavorVersionsMap() map[GameFlavor][]string {
	return gameVersions
}

func GetGameFlavorInterfacesMap() map[GameFlavor][]string {
	return gameInterfaces
}

func GetGameVersions() []string {
	var versions []string
	for _, version := range gameVersions {
		versions = append(versions, version...)
	}

	var uniqueVersions []string
	uniqueMap := make(map[string]bool)
	for _, version := range versions {
		if _, ok := uniqueMap[version]; !ok {
			uniqueMap[version] = true
			uniqueVersions = append(uniqueVersions, version)
		}
	}

	return uniqueVersions
}

func GetGameFlavors() []GameFlavor {
	var flavors []GameFlavor
	for flavor := range gameVersions {
		flavors = append(flavors, flavor)
	}
	return flavors
}
