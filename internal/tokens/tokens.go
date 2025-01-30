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
	BuildTimestamp     ValidToken = "build-timestamp"
	GameType           ValidToken = "game-type"
	ReleaseType        ValidToken = "release-type"

	// BuildType tokens
	Alpha          ValidToken = "alpha"
	Beta           ValidToken = "beta"
	Classic        ValidToken = "classic"
	Debug          ValidToken = "debug"
	DoNotPackage   ValidToken = "do-not-package"
	NoLibStrip     ValidToken = "no-lib-strip"
	NoLib          ValidToken = "nolib"
	Retail         ValidToken = "retail"
	VersionRetail  ValidToken = "version-retail"
	VersionClassic ValidToken = "version-classic"
	VersionBcc     ValidToken = "version-bcc"
	VersionWrath   ValidToken = "version-wrath"
	VersionCata    ValidToken = "version-cata"
	VersionMop     ValidToken = "version-mop" // Just guessing from here on
	VersionWod     ValidToken = "version-wod"
	VersionLegion  ValidToken = "version-legion"
	VersionBfa     ValidToken = "version-bfa"
	VersionSl      ValidToken = "version-sl"
	VersionDf      ValidToken = "version-df"
	VersionTWW     ValidToken = "version-tww"
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
}

var buildTypeTokens = []ValidToken{
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
		fileTokens, projectTokens, buildTokens, buildTypeTokens, allTemplateTokens, allTemplateFlags,
	),
)

func IsValidToken(token ValidToken) bool {
	return slices.Contains(allTokens, token)
}

func (token ValidToken) NormalizeToken() string {
	return fmt.Sprintf("@%s@", token)
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

type BuildTypeTokenMap map[ValidToken]bool

func (b BuildTypeTokenMap) Add(token ValidToken, value bool) {
	b[token] = value
}

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
