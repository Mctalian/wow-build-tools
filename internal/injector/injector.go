package injector

import (
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
	simpleTokens tokens.NormalizedSimpleTokenMap
	vcs          repo.VcsRepo
	pkgDir       string
	logGroup     *logger.LogGroup
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

func NewInjector(simpleTokens tokens.SimpleTokenMap, vR repo.VcsRepo, pkgDir string) (*Injector, error) {
	if len(simpleTokens) == 0 {
		return nil, nil
	}

	normalizedMap := make(tokens.NormalizedSimpleTokenMap)

	for token, value := range simpleTokens {
		if !tokens.IsValidToken(token) {
			return nil, tokens.ErrInvalidTokenValue{}
		}

		t := token.NormalizeToken()

		normalizedMap[token] = tokens.NormalizedSimpleToken{
			Normalized: t,
			Value:      value,
		}
	}

	i := Injector{
		simpleTokens: normalizedMap,
		vcs:          vR,
		pkgDir:       pkgDir,
	}

	return &i, nil
}
