package repo

import (
	"github.com/McTalian/wow-build-tools/internal/tokens"
)

type SvnRepo struct {
	BaseVcsRepo
	repo *Repo
}

func (sR *SvnRepo) IsIgnored(path string, isDir bool) bool {
	return false
}

func (sR *SvnRepo) GetInjectionValues(stm *tokens.SimpleTokenMap) error {
	return nil
}

func NewSvnRepo(r *Repo) (*SvnRepo, error) {
	sR := SvnRepo{repo: r}

	return &sR, nil
}
