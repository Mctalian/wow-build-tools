package tokens

import (
	"fmt"
	"slices"
	"strings"

	"github.com/McTalian/wow-build-tools/internal/logger"
)

type ErrUnknownTokenType struct{}

func (e ErrUnknownTokenType) Error() string {
	return "Unknown token type"
}

type ErrInvalidTokenValue struct{}

func (e ErrInvalidTokenValue) Error() string {
	return "Invalid token value"
}

var FilePrefix = "@file-"

type ValidToken string
type BuildTypeToken ValidToken
type FlagToken ValidToken

const (
	// Simple tokens
	FileRevision       ValidToken = "file-revision"
	FileHash           ValidToken = "file-hash"
	FileAbbrevHash     ValidToken = "file-abbreviated-hash"
	FileAuthor         ValidToken = "file-author"
	FileDateIso        ValidToken = "file-date-iso"
	FileDateInteger    ValidToken = "file-date-integer"
	FileTimestamp      ValidToken = "file-timestamp"
	ProjectRevision    ValidToken = "project-revision"
	ProjectHash        ValidToken = "project-hash"
	ProjectAbbrevHash  ValidToken = "project-abbreviated-hash"
	ProjectAuthor      ValidToken = "project-author"
	ProjectDateIso     ValidToken = "project-date-iso"
	ProjectDateInteger ValidToken = "project-date-integer"
	ProjectTimestamp   ValidToken = "project-timestamp"
	ProjectVersion     ValidToken = "project-version"
	PackageName        ValidToken = "package-name"
	BuildDate          ValidToken = "build-date"
	BuildDateIso       ValidToken = "build-date-iso"
	BuildDateInteger   ValidToken = "build-date-integer"
	BuildYear          ValidToken = "build-year"
	BuildTimestamp     ValidToken = "build-timestamp"
	GameType           ValidToken = "game-type"
	ReleaseType        ValidToken = "release-type"

	AlphaFlag   FlagToken = "alpha"
	BetaFlag    FlagToken = "beta"
	NoLibFlag   FlagToken = "nolib"
	ClassicFlag FlagToken = "classic"
	// BuildType tokens
	Alpha          BuildTypeToken = "alpha"
	Beta           BuildTypeToken = "beta"
	Classic        BuildTypeToken = "classic"
	Debug          BuildTypeToken = "debug"
	DoNotPackage   BuildTypeToken = "do-not-package"
	NoLibStrip     BuildTypeToken = "no-lib-strip"
	Retail         BuildTypeToken = "retail"
	VersionRetail  BuildTypeToken = "version-retail"
	VersionClassic BuildTypeToken = "version-classic"
	VersionBcc     BuildTypeToken = "version-bcc"
	VersionWrath   BuildTypeToken = "version-wrath"
	VersionCata    BuildTypeToken = "version-cata"
	VersionMop     BuildTypeToken = "version-mop" // Just guessing from here on
	VersionWod     BuildTypeToken = "version-wod"
	VersionLegion  BuildTypeToken = "version-legion"
	VersionBfa     BuildTypeToken = "version-bfa"
	VersionSl      BuildTypeToken = "version-sl"
	VersionDf      BuildTypeToken = "version-df"
	VersionTWW     BuildTypeToken = "version-tww"
)

var fileTokens = []ValidToken{
	FileRevision,
	FileHash,
	FileAbbrevHash,
	FileAuthor,
	FileDateIso,
	FileDateInteger,
	FileTimestamp,
}

var projectTokens = []ValidToken{
	ProjectRevision,
	ProjectHash,
	ProjectAbbrevHash,
	ProjectAuthor,
	ProjectDateIso,
	ProjectDateInteger,
	ProjectTimestamp,
	ProjectVersion,
}

var buildTokens = []ValidToken{
	BuildDate,
	BuildDateIso,
	BuildDateInteger,
	BuildTimestamp,
	BuildYear,
}

var buildTypeTokens = []BuildTypeToken{
	Alpha,
	Debug,
	DoNotPackage,
	NoLibStrip,
	Retail,
	VersionRetail,
	VersionClassic,
	VersionBcc,
	VersionWrath,
	VersionCata,
	VersionMop,
	VersionWod,
	VersionLegion,
	VersionBfa,
	VersionSl,
	VersionDf,
	VersionTWW,
}

func uniqueTokens(slice []ValidToken) []ValidToken {
	keys := make(map[ValidToken]bool)
	list := []ValidToken{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

var allTokens = uniqueTokens(
	slices.Concat(
		fileTokens, projectTokens, buildTokens, allTemplateTokens,
	),
)

// var allFlags = []FlagToken{
// 	AlphaFlag,
// 	BetaFlag,
// 	NoLibFlag,
// 	ClassicFlag,
// }

func IsValidToken(token string) bool {
	if token == "" {
		return false
	}

	return slices.Contains(allTokens, ValidToken(token)) || slices.Contains(buildTypeTokens, BuildTypeToken(token))
}

func (token ValidToken) NormalizeToken() string {
	return fmt.Sprintf("@%s@", token)
}

func (token BuildTypeToken) NormalizeToken() string {
	return fmt.Sprintf("@%s@", token)
}

type BuildTypeTokenVariants struct {
	Standard    string
	StandardEnd string
	Negative    string
	NegativeEnd string
}

func (token BuildTypeToken) GetVariants() BuildTypeTokenVariants {
	return BuildTypeTokenVariants{
		Standard:    string(token),
		StandardEnd: "end-" + string(token),
		Negative:    "non-" + string(token),
		NegativeEnd: "end-non-" + string(token),
	}
}

type TokenType int

const (
	Unknown TokenType = iota
	Simple
	BuildType
	//Localization // Probably want to handle this in a separate file
)

type SimpleTokenMap map[ValidToken]string

func (s SimpleTokenMap) Add(token ValidToken, value string) {
	s[token] = value
}

func (s SimpleTokenMap) String() string {
	var str strings.Builder
	for token, value := range s {
		str.WriteString(string(token) + ": " + value + "\n")
	}
	return str.String()
}

type BuildTypeTokenMap map[BuildTypeToken]bool

func (b BuildTypeTokenMap) Add(token BuildTypeToken, value bool) {
	b[token] = value
}

type NormalizedBuildTypeToken struct {
	Normalized       string
	NormalizedEnd    string
	NormalizedNeg    string
	NormalizedNegEnd string
	Value            bool
}

type NormalizedBuildTypeTokenMap map[BuildTypeToken]NormalizedBuildTypeToken

type NormalizedSimpleToken struct {
	Normalized string
	Value      string
}

type NormalizedSimpleTokenMap map[ValidToken]NormalizedSimpleToken

func (n NormalizedSimpleTokenMap) Add(token ValidToken, value string) {
	n[token] = NormalizedSimpleToken{
		Normalized: token.NormalizeToken(),
		Value:      value,
	}
}

func (n NormalizedSimpleTokenMap) ExtendSimpleMap(stm *SimpleTokenMap) {
	if stm == nil {
		logger.Warn("SimpleTokenMap is nil")
		return
	}
	for token, value := range *stm {
		n.Add(token, value)
	}
}
