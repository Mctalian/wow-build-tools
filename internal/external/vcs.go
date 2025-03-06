package external

type Vcs interface {
	Checkout() error
	lookForCurseSlug() error
}

type VcsType int

const (
	Unknown VcsType = iota
	Git
	Svn
	Hg
)

func ToVcsType(t string) VcsType {
	switch t {
	case "git":
		return Git
	case "svn":
		return Svn
	case "hg":
		return Hg
	default:
		return Unknown
	}
}

func (t VcsType) ToString() string {
	switch t {
	case Git:
		return "git"
	case Svn:
		return "svn"
	case Hg:
		return "hg"
	default:
		return "unknown"
	}
}

type BaseVcs struct {
	Vcs
}
