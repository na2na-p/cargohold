package domain

type OAuthState struct {
	repository  string
	redirectURI string
	shell       ShellType
}

func NewOAuthState(repository, redirectURI string, shell ShellType) *OAuthState {
	return &OAuthState{
		repository:  repository,
		redirectURI: redirectURI,
		shell:       shell,
	}
}

func (o *OAuthState) Repository() string {
	return o.repository
}

func (o *OAuthState) RedirectURI() string {
	return o.redirectURI
}

func (o *OAuthState) Shell() ShellType {
	return o.shell
}
