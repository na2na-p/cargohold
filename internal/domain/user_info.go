package domain

type UserInfo struct {
	sub        string
	email      string
	name       string
	provider   ProviderType
	repository *RepositoryIdentifier
	ref        string
}

func NewUserInfo(sub, email, name string, provider ProviderType, repository *RepositoryIdentifier, ref string) (*UserInfo, error) {
	if sub == "" {
		return nil, ErrEmptySub
	}
	return &UserInfo{
		sub:        sub,
		email:      email,
		name:       name,
		provider:   provider,
		repository: repository,
		ref:        ref,
	}, nil
}

func (u *UserInfo) Sub() string {
	return u.sub
}

func (u *UserInfo) Email() string {
	return u.email
}

func (u *UserInfo) Name() string {
	return u.name
}

func (u *UserInfo) Provider() ProviderType {
	return u.provider
}

func (u *UserInfo) Repository() *RepositoryIdentifier {
	return u.repository
}

func (u *UserInfo) Ref() string {
	return u.ref
}
