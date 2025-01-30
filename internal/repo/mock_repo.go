package repo

import "github.com/McTalian/wow-build-tools/internal/tokens"

type MockVcsRepo struct {
	VcsRepo
	IsIgnoredFunc              func(path string, isDir bool) bool
	GetInjectionValuesFunc     func(stm *tokens.SimpleTokenMap) error
	GetFileInjectionValuesFunc func(filePath string) (*tokens.SimpleTokenMap, error)
	GetRepoRootFunc            func() string
	GetChangelogFunc           func(title string) (string, error)
	GetCurrentTagFunc          func() string
	GetPreviousVersionFunc     func() string
	GetProjectVersionFunc      func() string
}

func (mR *MockVcsRepo) IsIgnored(path string, isDir bool) bool {
	if mR.IsIgnoredFunc != nil {
		return mR.IsIgnoredFunc(path, isDir)
	}
	return false
}

func (mR *MockVcsRepo) GetInjectionValues(stm *tokens.SimpleTokenMap) error {
	if mR.GetInjectionValuesFunc != nil {
		return mR.GetInjectionValuesFunc(stm)
	}
	return nil
}

func (mR *MockVcsRepo) GetFileInjectionValues(filePath string) (*tokens.SimpleTokenMap, error) {
	if mR.GetFileInjectionValuesFunc != nil {
		return mR.GetFileInjectionValuesFunc(filePath)
	}
	return nil, nil
}

func (mR *MockVcsRepo) GetRepoRoot() string {
	if mR.GetRepoRootFunc != nil {
		return mR.GetRepoRootFunc()
	}
	return ""
}

func (mR *MockVcsRepo) GetChangelog(title string) (string, error) {
	if mR.GetChangelogFunc != nil {
		return mR.GetChangelogFunc(title)
	}
	return "", nil
}

func (mR *MockVcsRepo) GetCurrentTag() string {
	if mR.GetCurrentTagFunc != nil {
		return mR.GetCurrentTagFunc()
	}
	return ""
}

func (mR *MockVcsRepo) GetPreviousVersion() string {
	if mR.GetPreviousVersionFunc != nil {
		return mR.GetPreviousVersionFunc()
	}
	return ""
}

func (mR *MockVcsRepo) GetProjectVersion() string {
	if mR.GetProjectVersionFunc != nil {
		return mR.GetProjectVersionFunc()
	}
	return ""
}
