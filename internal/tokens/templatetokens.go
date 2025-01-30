package tokens

import (
	"fmt"
	"regexp"
	"strings"
)

type NormalizedTemplateToken struct {
	TemplateToken string
	Value         string
}

type TemplateTokenMap map[ValidToken]NormalizedTemplateToken

func (v ValidToken) NormalizeTemplateToken() string {
	return fmt.Sprintf("{%s}", v)
}

func (t TemplateTokenMap) FromSTM(stm *SimpleTokenMap) {
	for token, value := range *stm {
		t[token] = NormalizedTemplateToken{
			TemplateToken: token.NormalizeTemplateToken(),
			Value:         value,
		}
	}
}

var allTemplateTokens = []ValidToken{
	PackageName,
	ProjectRevision,
	ProjectHash,
	ProjectAbbrevHash,
	ProjectAuthor,
	ProjectDateIso,
	ProjectDateInteger,
	ProjectTimestamp,
	ProjectVersion,
	GameType,
	ReleaseType,
}

var allTemplateFlags = []ValidToken{
	Alpha,
	Beta,
	NoLib,
	Classic,
}

var DefaultFile = fmt.Sprintf("%s-%s%s%s", PackageName.NormalizeTemplateToken(), ProjectVersion.NormalizeTemplateToken(), NoLib.NormalizeTemplateToken(), Classic.NormalizeTemplateToken())
var DefaultLabel = fmt.Sprintf("%s%s%s", ProjectVersion.NormalizeTemplateToken(), Classic.NormalizeTemplateToken(), NoLib.NormalizeTemplateToken())

func tokenSection() string {
	return fmt.Sprintf(`
	Tokens:
		%s%s%s%s
		%s%s%s%s
		%s%s%s`,
		convertToInterfaceSlice(allTemplateTokens)...)
}

func flagSection() string {
	return fmt.Sprintf(`
	Flags:
		%s%s%s%s
`, convertToInterfaceSlice(allTemplateFlags)...)
}

func NameTemplateUsageInfo() string {
	return fmt.Sprintf(`
Name Template Help:

	Set the package zip file name and upload file label. There are several string
	substitutions you can use to include version control and build type information in
	the file name and upload label.

	The default file name is "%s".
	The default upload label is "%s".

	To set both, separate with a ":", i.e, "{file template}:{label template}".
	If either side of the ":" is blank, the default will be used. Not including a
	":" will set the file name template, leaving upload label as default.
%s
%s
	Tokens are always replaced with their value. Flags are shown prefixed with a dash
	depending on the build type.
`, DefaultFile, DefaultLabel, tokenSection(), flagSection())
}

func convertToInterfaceSlice(tokens []ValidToken) []interface{} {
	converted := make([]interface{}, len(tokens))
	for i, token := range tokens {
		converted[i] = token.NormalizeTemplateToken()
	}
	return converted
}

func validateTemplate(template string) error {
	templateCopy := template

	for _, token := range allTemplateTokens {
		templateCopy = strings.ReplaceAll(templateCopy, string(token.NormalizeTemplateToken()), "")
	}

	for _, flag := range allTemplateFlags {
		templateCopy = strings.ReplaceAll(templateCopy, string(flag.NormalizeTemplateToken()), "")
	}

	if strings.Contains(templateCopy, "{") || strings.Contains(templateCopy, "}") {
		re := regexp.MustCompile(`(\{[^{}]+\})`)
		matches := re.FindAllStringSubmatch(templateCopy, -1)

		// Collect only the pieces inside the braces.
		var remainingTokens []string
		for _, match := range matches {
			// match[0] is the whole match (e.g. "{token}"), match[1] is just "token".
			remainingTokens = append(remainingTokens, match[1])
		}

		// Optionally print the tokens or include them in your error message.
		return fmt.Errorf("invalid template substitutions: %s", strings.Join(remainingTokens, ", "))
	}

	return nil
}

type NameTemplate struct {
	FileTemplate  string
	LabelTemplate string
	HasNoLib      bool
}

func (n *NameTemplate) GetFileName(stm *SimpleTokenMap, noLib bool) string {
	ttm := make(TemplateTokenMap)
	ttm.FromSTM(stm)

	filename := n.FileTemplate

	if noLib {
		filename = strings.Replace(filename, NoLib.NormalizeTemplateToken(), "-nolib", -1)
	} else {
		filename = strings.Replace(filename, NoLib.NormalizeTemplateToken(), "", -1)
	}

	for _, value := range ttm {
		filename = strings.ReplaceAll(filename, value.TemplateToken, value.Value)
	}

	return filename
}

func (n *NameTemplate) GetLabel(stm *SimpleTokenMap, noLib bool) string {
	ttm := make(TemplateTokenMap)
	ttm.FromSTM(stm)

	label := n.LabelTemplate

	if noLib {
		label = strings.Replace(label, NoLib.NormalizeTemplateToken(), "-nolib", -1)
	} else {
		label = strings.Replace(label, NoLib.NormalizeTemplateToken(), "", -1)
	}

	for _, value := range ttm {
		label = strings.ReplaceAll(label, value.TemplateToken, value.Value)
	}

	return label
}

func NewNameTemplate(template string) (*NameTemplate, error) {
	var fileTemplate, labelTemplate string
	splitTemplate := strings.Split(template, ":")
	if len(splitTemplate) > 2 {
		return nil, fmt.Errorf("invalid template format, found more than one \":\" in %s", template)
	} else if len(splitTemplate) == 2 {
		fileTemplate = splitTemplate[0]
		labelTemplate = splitTemplate[1]
	} else if len(splitTemplate) == 1 {
		fileTemplate = splitTemplate[0]
	}

	if fileTemplate == "" {
		fileTemplate = DefaultFile
	}

	if labelTemplate == "" {
		labelTemplate = DefaultLabel
	}

	if err := validateTemplate(fileTemplate); err != nil {
		return nil, fmt.Errorf("file template: %s", err)
	}

	if err := validateTemplate(labelTemplate); err != nil {
		return nil, fmt.Errorf("label template: %s", err)
	}

	// Check if the template includes the NoLib flag.
	// if it does not, we can't create noLib versions of the package, if requested.
	hasNoLib := strings.Contains(fileTemplate, NoLib.NormalizeTemplateToken()) && strings.Contains(labelTemplate, NoLib.NormalizeTemplateToken())

	return &NameTemplate{
		FileTemplate:  fileTemplate,
		LabelTemplate: labelTemplate,
		HasNoLib:      hasNoLib,
	}, nil
}
