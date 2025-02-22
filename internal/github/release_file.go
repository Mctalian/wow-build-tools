package github

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/McTalian/wow-build-tools/internal/toc"
)

type metadata struct {
	Flavor    string `json:"flavor"`
	Interface string `json:"interface"`
}

type wbtRelease struct {
	Name     string     `json:"name"`
	Version  string     `json:"version"`
	Filename string     `json:"filename"`
	NoLib    bool       `json:"nolib"`
	Metadata []metadata `json:"metadata"`
}

type wbtReleaseMetadata struct {
	Releases []wbtRelease `json:"releases"`
}

func GetReleaseMetadataContents(name string, version string, gameVersions toc.GameVersions, zipFileNames ...string) (string, error) {
	releaseMetadata := wbtReleaseMetadata{}
	for _, zipFileName := range zipFileNames {
		noLib := strings.Contains(zipFileName, "nolib")
		release := wbtRelease{
			Name:     name,
			Version:  version,
			Filename: zipFileName,
			NoLib:    noLib,
		}

		for flavor, versions := range gameVersions {
			for _, v := range versions {
				flavorStr := flavor.ToString()
				if flavor == toc.Retail {
					flavorStr = "mainline"
				}
				release.Metadata = append(release.Metadata, metadata{
					Flavor:    flavorStr,
					Interface: v,
				})
			}
		}
		releaseMetadata.Releases = append(releaseMetadata.Releases, release)
	}

	contents, err := json.Marshal(&releaseMetadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal release metadata: %w", err)
	}

	return string(contents), nil
}
