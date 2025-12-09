package domain

type GitHubUserInfo struct {
	sub        string
	repository string
	ref        string
	actor      string
}

func NewGitHubUserInfo(sub, repository, ref, actor string) *GitHubUserInfo {
	return &GitHubUserInfo{
		sub:        sub,
		repository: repository,
		ref:        ref,
		actor:      actor,
	}
}

func (g *GitHubUserInfo) Sub() string {
	return g.sub
}

func (g *GitHubUserInfo) Repository() string {
	return g.repository
}

func (g *GitHubUserInfo) Ref() string {
	return g.ref
}

func (g *GitHubUserInfo) Actor() string {
	return g.actor
}

func (g *GitHubUserInfo) ToUserInfo() (*UserInfo, error) {
	repo, err := NewRepositoryIdentifier(g.repository)
	if err != nil {
		return nil, err
	}
	return NewUserInfo(
		g.sub,
		"",
		g.actor,
		ProviderTypeGitHub,
		repo,
		g.ref,
	)
}
