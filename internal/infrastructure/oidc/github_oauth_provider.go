package oidc

import (
	"context"

	"github.com/na2na-p/cargohold/internal/domain"
)

const (
	GitHubOAuthAuthorizeURL = "https://github.com/login/oauth/authorize"
	GitHubOAuthTokenURL     = "https://github.com/login/oauth/access_token"
	GitHubAPIUserURL        = "https://api.github.com/user"
	GitHubAPIBaseURL        = "https://api.github.com"
)

type OAuthToken struct {
	AccessToken string
	TokenType   string
	Scope       string
}

type GitHubUser struct {
	ID    int64
	Login string
	Name  string
}

type GitHubOAuthProvider struct {
	tokenExchanger    *gitHubTokenExchanger
	userInfoProvider  *gitHubUserInfoProvider
	repositoryChecker *gitHubRepositoryChecker
}

func NewGitHubOAuthProvider(clientID, clientSecret, redirectURI string) (*GitHubOAuthProvider, error) {
	tokenExchanger, err := NewGitHubTokenExchanger(clientID, clientSecret, redirectURI)
	if err != nil {
		return nil, err
	}

	return &GitHubOAuthProvider{
		tokenExchanger:    tokenExchanger,
		userInfoProvider:  NewGitHubUserInfoProvider(),
		repositoryChecker: NewGitHubRepositoryChecker(),
	}, nil
}

func (p *GitHubOAuthProvider) SetTokenEndpoint(endpoint string) {
	p.tokenExchanger.SetTokenEndpoint(endpoint)
}

func (p *GitHubOAuthProvider) SetUserInfoEndpoint(endpoint string) {
	p.userInfoProvider.SetUserInfoEndpoint(endpoint)
}

func (p *GitHubOAuthProvider) SetAPIEndpoint(endpoint string) {
	p.repositoryChecker.SetAPIEndpoint(endpoint)
}

func (p *GitHubOAuthProvider) SetRedirectURI(redirectURI string) {
	p.tokenExchanger.SetRedirectURI(redirectURI)
}

func (p *GitHubOAuthProvider) GetAuthorizationURL(state string, scopes []string) string {
	return p.tokenExchanger.GetAuthorizationURL(state, scopes)
}

func (p *GitHubOAuthProvider) ExchangeCode(ctx context.Context, code string) (*OAuthToken, error) {
	return p.tokenExchanger.ExchangeCode(ctx, code)
}

func (p *GitHubOAuthProvider) GetUserInfo(ctx context.Context, token *OAuthToken) (*GitHubUser, error) {
	return p.userInfoProvider.GetUserInfo(ctx, token)
}

func (p *GitHubOAuthProvider) CanAccessRepository(ctx context.Context, token *OAuthToken, repo *domain.RepositoryIdentifier) (bool, error) {
	return p.repositoryChecker.CanAccessRepository(ctx, token, repo)
}
