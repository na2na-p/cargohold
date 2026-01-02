package domain

type RepositoryPermissions struct {
	admin    bool
	push     bool
	pull     bool
	maintain bool
	triage   bool
}

func NewRepositoryPermissions(admin, push, pull, maintain, triage bool) RepositoryPermissions {
	return RepositoryPermissions{
		admin:    admin,
		push:     push,
		pull:     pull,
		maintain: maintain,
		triage:   triage,
	}
}

func (p RepositoryPermissions) CanUpload() bool {
	return p.push || p.admin || p.maintain
}

func (p RepositoryPermissions) CanDownload() bool {
	return p.pull || p.push || p.admin || p.maintain || p.triage
}

func (p RepositoryPermissions) Admin() bool {
	return p.admin
}

func (p RepositoryPermissions) Push() bool {
	return p.push
}

func (p RepositoryPermissions) Pull() bool {
	return p.pull
}

func (p RepositoryPermissions) Maintain() bool {
	return p.maintain
}

func (p RepositoryPermissions) Triage() bool {
	return p.triage
}
