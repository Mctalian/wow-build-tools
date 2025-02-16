package injector

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/repo"
	"github.com/McTalian/wow-build-tools/internal/tokens"
)

type supportedFileType string

const (
	LuaFile supportedFileType = ".lua"
	XmlFile supportedFileType = ".xml"
	TocFile supportedFileType = ".toc"
	MdFile  supportedFileType = ".md"
	TxtFile supportedFileType = ".txt"
)

var injectableExtensions = []supportedFileType{
	LuaFile,
	XmlFile,
	TocFile,
	MdFile,
	TxtFile,
}

func isInjectableExtension(ext string) bool {
	for _, e := range injectableExtensions {
		if string(e) == ext {
			return true
		}
	}
	return false
}

type Injector struct {
	simpleTokens    tokens.NormalizedSimpleTokenMap
	buildTypeTokens tokens.NormalizedBuildTypeTokenMap
	vcs             repo.VcsRepo
	pkgDir          string
	logGroup        *logger.LogGroup
}

func (i *Injector) findAndReplaceInFile(filePath string) error {
	input, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	output := string(input)

	if strings.Contains(output, tokens.FilePrefix) {
		// Need to get the file info from VCS
		origFilePath := strings.TrimPrefix(filePath, i.pkgDir+"/")
		stm, err := i.vcs.GetFileInjectionValues(origFilePath)
		if err != nil {
			return err
		}

		i.simpleTokens.ExtendSimpleMap(stm)
	}

	for token, n := range i.simpleTokens {
		if !strings.Contains(output, string(token)) {
			continue
		}
		i.logGroup.Verbose("Replaced %s with %s in %s", token, n.Value, filePath)
		output = strings.ReplaceAll(output, n.Normalized, n.Value)
	}

	for token, n := range i.buildTypeTokens {
		if n.Normalized != "" && n.NormalizedEnd != "" && strings.Contains(output, n.Normalized) && strings.Contains(output, n.NormalizedEnd) {
			ext := filepath.Ext(filePath)
			if ext == ".toc" {
				// Need to comment out all the lines between the start and end tokens
				lines := strings.Split(output, "\n")
				var foundStart, foundEnd bool
				for i, line := range lines {
					if strings.Contains(line, "#"+n.Normalized) {
						foundStart = true
						continue
					}
					if strings.Contains(line, "#"+n.NormalizedEnd) {
						foundEnd = true
					}
					if foundStart && !foundEnd {
						lines[i] = "#" + line
					}
				}
				i.logGroup.Verbose("Handled %s block (%s) in %s", token, n.Normalized, filePath)
				output = strings.Join(lines, "\n")
			} else {
				var findStart, findEnd, replaceStart, replaceEnd string
				if filepath.Ext(filePath) == ".lua" {
					findStart = fmt.Sprintf("%s", n.Normalized)
					findEnd = fmt.Sprintf("%s", n.NormalizedEnd)
					replaceStart = fmt.Sprintf("[===[%s", n.Normalized)
					replaceEnd = fmt.Sprintf("%s]===]", n.NormalizedEnd)
				} else if filepath.Ext(filePath) == ".xml" {
					findStart = fmt.Sprintf("%s-->", n.Normalized)
					findEnd = fmt.Sprintf("<!--%s", n.NormalizedEnd)
					replaceStart = fmt.Sprintf("%s", n.Normalized)
					replaceEnd = fmt.Sprintf("%s", n.NormalizedEnd)
				}
				i.logGroup.Verbose("Handled %s block (%s) in %s", token, n.Normalized, filePath)
				output = strings.ReplaceAll(output, findStart, replaceStart)
				output = strings.ReplaceAll(output, findEnd, replaceEnd)
			}
		}
		if n.NormalizedNeg != "" && n.NormalizedNegEnd != "" && strings.Contains(output, n.NormalizedNeg) && strings.Contains(output, n.NormalizedNegEnd) {
			ext := filepath.Ext(filePath)
			if ext == ".toc" {
				// Need to comment out all the lines between the start and end tokens
				lines := strings.Split(output, "\n")
				var foundStart, foundEnd bool
				for i, line := range lines {
					if strings.Contains(line, "#"+n.NormalizedNeg) {
						foundStart = true
						continue
					}
					if strings.Contains(line, "#"+n.NormalizedNegEnd) {
						foundEnd = true
					}
					if foundStart && !foundEnd {
						if strings.Contains(line, "#") {
							lines[i] = strings.Replace(line, "#", "", 1)
						}
					}
				}
				i.logGroup.Verbose("Handled non-%s block (%s) in %s", token, n.NormalizedNeg, filePath)
				output = strings.Join(lines, "\n")
			} else {
				var findStart, findEnd, replaceStart, replaceEnd string
				if filepath.Ext(filePath) == ".lua" {
					findStart = fmt.Sprintf("[===[%s", n.NormalizedNeg)
					findEnd = fmt.Sprintf("%s]===]", n.NormalizedNegEnd)
					replaceStart = fmt.Sprintf("%s", n.NormalizedNeg)
					replaceEnd = fmt.Sprintf("%s", n.NormalizedNegEnd)
				} else if filepath.Ext(filePath) == ".xml" {
					findStart = fmt.Sprintf("%s", n.NormalizedNeg)
					findEnd = fmt.Sprintf("%s", n.NormalizedNegEnd)
					replaceStart = fmt.Sprintf("%s-->", n.NormalizedNeg)
					replaceEnd = fmt.Sprintf("<!--%s", n.NormalizedNegEnd)
				}
				i.logGroup.Verbose("Handled non-%s block (%s) in %s", token, n.NormalizedNeg, filePath)
				output = strings.ReplaceAll(output, findStart, replaceStart)
				output = strings.ReplaceAll(output, findEnd, replaceEnd)
			}
		}
	}

	if strings.Contains(output, tokens.DoNotPackage.NormalizeToken()) {
		i.logGroup.Verbose("Removing %s from %s", tokens.DoNotPackage, filePath)
		lines := strings.Split(output, "\n")
		variants := tokens.DoNotPackage.GetVariants()
		startToken := fmt.Sprintf("@%s@", variants.Standard)
		endToken := fmt.Sprintf("@%s@", variants.StandardEnd)
		var newLines []string
		var inSection bool
		for _, line := range lines {
			if strings.Contains(line, startToken) {
				inSection = true
				continue
			}
			if strings.Contains(line, endToken) {
				inSection = false
				continue
			}
			if !inSection {
				newLines = append(newLines, line)
			}
		}
		output = strings.Join(newLines, "\n")
	}

	err = os.WriteFile(filePath, []byte(output), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (i *Injector) Execute() error {
	i.logGroup = logger.NewLogGroup("💉 Injecting tokens into package directory")
	defer i.logGroup.Flush(true)

	return filepath.WalkDir(i.pkgDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Only process supported file types
		ext := filepath.Ext(path)
		if ext == "" {
			return nil
		}

		if !isInjectableExtension(ext) {
			return nil
		}

		return i.findAndReplaceInFile(path)
	})
}

func NewInjector(simpleTokens tokens.SimpleTokenMap, vR repo.VcsRepo, pkgDir string, buildTypeTokens tokens.BuildTypeTokenMap) (*Injector, error) {
	if len(simpleTokens) == 0 {
		return nil, fmt.Errorf("no simple tokens provided")
	}

	normalizedMap := make(tokens.NormalizedSimpleTokenMap)

	for token, value := range simpleTokens {
		if !tokens.IsValidToken(string(token)) {
			return nil, tokens.ErrInvalidTokenValue{}
		}

		t := token.NormalizeToken()

		normalizedMap[token] = tokens.NormalizedSimpleToken{
			Normalized: t,
			Value:      value,
		}
	}

	normalizeBuildTypeMap := make(tokens.NormalizedBuildTypeTokenMap)

	for token, value := range buildTypeTokens {
		variants := token.GetVariants()
		if value {
			normalizeBuildTypeMap[token] = tokens.NormalizedBuildTypeToken{
				Normalized:       "",
				NormalizedEnd:    "",
				NormalizedNeg:    "",
				NormalizedNegEnd: "",
			}
		} else {
			normalizeBuildTypeMap[token] = tokens.NormalizedBuildTypeToken{
				Normalized:       fmt.Sprintf("@%s@", variants.Standard),
				NormalizedEnd:    fmt.Sprintf("@%s@", variants.StandardEnd),
				NormalizedNeg:    fmt.Sprintf("@%s@", variants.Negative),
				NormalizedNegEnd: fmt.Sprintf("@%s@", variants.NegativeEnd),
			}
		}
	}

	i := Injector{
		simpleTokens:    normalizedMap,
		buildTypeTokens: normalizeBuildTypeMap,
		vcs:             vR,
		pkgDir:          pkgDir,
	}

	return &i, nil
}
