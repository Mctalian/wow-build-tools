package license

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/McTalian/wow-build-tools/internal/tokens"
	"golang.org/x/net/html"
)

var baseUrl = "https://www.wowace.com/project/"
var licensePath = "/license"

func downloadLicense(curseProjectId string) (string, error) {
	// Download license file from curse project
	url := fmt.Sprintf("%s%s%s", baseUrl, curseProjectId, licensePath)

	// Download license file
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	htmlContents, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	doc, err := html.Parse(strings.NewReader(string(htmlContents)))
	if err != nil {
		return "", nil
	}

	docIter := doc.Descendants()
	for node := range docIter {
		if node.Type == html.ElementNode && node.Data == "p" {
			var buf strings.Builder
			for c := node.FirstChild; c != nil; c = c.NextSibling {
				err = html.Render(&buf, c)
				if err != nil {
					return "", err
				}
			}
			raw := buf.String()
			//&lt;year&gt; &lt;copyright holders&gt
			raw = strings.ReplaceAll(raw, "&lt;year&gt;", string(tokens.BuildYear.NormalizeToken()))
			raw = strings.ReplaceAll(raw, "&lt;copyright holders&gt;", string(tokens.ProjectAuthor.NormalizeToken()))
			return raw, nil
		}
	}

	return "", fmt.Errorf("html parse error: structure from license endpoint may have changed")
}

func EnsureLicensePresent(pkgMetaLicense string, topDir string, pkgDir string, curseProjectId string) error {
	if pkgMetaLicense == "" {
		// No license requested via pkgmeta
		return nil
	}

	if curseProjectId == "" {
		return nil
	}

	licensePathCheck := filepath.Join(topDir, pkgMetaLicense)
	_, err := os.Stat(licensePathCheck)
	if err == nil {
		// License file exists
		return nil
	} else if os.IsNotExist(err) {
		// License file does not exist, download it
		licenseContents, err := downloadLicense(curseProjectId)
		if err != nil {
			return fmt.Errorf("error downloading license file: %w", err)
		}

		// Write license file
		destPath := filepath.Join(pkgDir, pkgMetaLicense)
		err = os.WriteFile(destPath, []byte(licenseContents), 0644)
		if err != nil {
			return fmt.Errorf("error writing license file: %w", err)
		}
	} else {
		return fmt.Errorf("error checking for license file: %w", err)
	}

	return nil
}
