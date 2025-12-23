package domain

type OAuthState struct {
	repository  string
	redirectURI string
}

func NewOAuthState(repository, redirectURI string) *OAuthState {
	return &OAuthState{
		repository:  repository,
		redirectURI: redirectURI,
	}
}

func (o *OAuthState) Repository() string {
	return o.repository
}

func (o *OAuthState) RedirectURI() string {
	return o.redirectURI
}
