package domain

type OAuthState struct {
	repository  string
	redirectURI string
	shell       string
}

func NewOAuthState(repository, redirectURI, shell string) *OAuthState {
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

func (o *OAuthState) Shell() string {
	return o.shell
}
