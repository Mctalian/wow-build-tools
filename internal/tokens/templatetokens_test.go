package tokens

import (
	"testing"
)

func TestGetFileName(t *testing.T) {
	tests := []struct {
		name         string
		template     *NameTemplate
		stm          *SimpleTokenMap
		flags        FlagMap
		expectedName string
	}{
		{
			name: "NoLib true",
			template: &NameTemplate{
				FileTemplate: "{package-name}-{project-version}{nolib}{classic}",
			},
			stm: &SimpleTokenMap{
				PackageName:    "TestPackage",
				ProjectVersion: "1.0.0",
			},
			flags: FlagMap{
				NoLibFlag:   "-nolib",
				ClassicFlag: "-classic",
			},
			expectedName: "TestPackage-1.0.0-nolib-classic",
		},
		{
			name: "NoLib false",
			template: &NameTemplate{
				FileTemplate: "{package-name}-{project-version}{nolib}{classic}",
			},
			stm: &SimpleTokenMap{
				PackageName:    "TestPackage",
				ProjectVersion: "1.0.0",
			},
			flags: FlagMap{
				NoLibFlag:   "",
				ClassicFlag: "-classic",
			},
			expectedName: "TestPackage-1.0.0-classic",
		},
		{
			name: "NoLib token not in template",
			template: &NameTemplate{
				FileTemplate: "{package-name}-{project-version}{classic}",
			},
			stm: &SimpleTokenMap{
				PackageName:    "TestPackage",
				ProjectVersion: "1.0.0",
			},
			flags: FlagMap{
				NoLibFlag:   "-nolib",
				ClassicFlag: "-classic",
			},
			expectedName: "TestPackage-1.0.0-classic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.template.GetFileName(tt.stm, tt.flags)
			if got != tt.expectedName {
				t.Errorf("GetFileName() = %v, want %v", got, tt.expectedName)
			}
		})
	}
}
func TestNewNameTemplate(t *testing.T) {
	tests := []struct {
		name          string
		template      string
		expectedError bool
		expected      *NameTemplate
	}{
		{
			name:     "Valid template with file and label",
			template: "{package-name}-{project-version}:{project-version}{classic}",
			expected: &NameTemplate{
				FileTemplate:  "{package-name}-{project-version}",
				LabelTemplate: "{project-version}{classic}",
				HasNoLib:      false,
			},
		},
		{
			name:     "Valid template with only file",
			template: "{package-name}-{project-version}",
			expected: &NameTemplate{
				FileTemplate:  "{package-name}-{project-version}",
				LabelTemplate: DefaultLabel,
				HasNoLib:      false,
			},
		},
		{
			name:     "Valid template with NoLib flag",
			template: "{package-name}-{project-version}{nolib}:{project-version}{nolib}",
			expected: &NameTemplate{
				FileTemplate:  "{package-name}-{project-version}{nolib}",
				LabelTemplate: "{project-version}{nolib}",
				HasNoLib:      true,
			},
		},
		{
			name:          "Invalid template with multiple colons",
			template:      "{package-name}:{project-version}:{classic}",
			expectedError: true,
		},
		{
			name:          "Invalid template with unclosed brace",
			template:      "{package-name-{project-version}",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNameTemplate(tt.template)
			if (err != nil) != tt.expectedError {
				t.Errorf("NewNameTemplate() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if !tt.expectedError && !compareNameTemplates(got, tt.expected) {
				t.Errorf("NewNameTemplate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetLabel(t *testing.T) {
	tests := []struct {
		name          string
		template      *NameTemplate
		stm           *SimpleTokenMap
		flags         FlagMap
		expectedLabel string
	}{
		{
			name: "NoLib true",
			template: &NameTemplate{
				LabelTemplate: "{project-version}{nolib}{classic}",
			},
			stm: &SimpleTokenMap{
				ProjectVersion: "1.0.0",
			},
			flags: FlagMap{
				NoLibFlag:   "-nolib",
				ClassicFlag: "-classic",
			},
			expectedLabel: "1.0.0-nolib-classic",
		},
		{
			name: "NoLib false",
			template: &NameTemplate{
				LabelTemplate: "{project-version}{nolib}{classic}",
			},
			stm: &SimpleTokenMap{
				ProjectVersion: "1.0.0",
			},
			flags: FlagMap{
				NoLibFlag:   "",
				ClassicFlag: "-classic",
			},
			expectedLabel: "1.0.0-classic",
		},
		{
			name: "NoLib token not in template",
			template: &NameTemplate{
				LabelTemplate: "{project-version}{classic}",
			},
			stm: &SimpleTokenMap{
				ProjectVersion: "1.0.0",
			},
			flags: FlagMap{
				NoLibFlag:   "-nolib",
				ClassicFlag: "-classic",
			},
			expectedLabel: "1.0.0-classic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.template.GetLabel(tt.stm, tt.flags)
			if got != tt.expectedLabel {
				t.Errorf("GetLabel() = %v, want %v", got, tt.expectedLabel)
			}
		})
	}
}

func compareNameTemplates(a, b *NameTemplate) bool {
	return a.FileTemplate == b.FileTemplate && a.LabelTemplate == b.LabelTemplate && a.HasNoLib == b.HasNoLib
}
