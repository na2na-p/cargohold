//go:generate mockgen -source=$GOFILE -destination=mock_github_oauth_provider_internal_test.go -package=oidc
package oidc

import (
	"context"

	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
)

type GitHubOAuthProviderInternal interface {
	SetRedirectURI(redirectURI string)
	GetAuthorizationURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*oauthToken, error)
	GetUserInfo(ctx context.Context, token *oauthToken) (*gitHubUser, error)
	CanAccessRepository(ctx context.Context, token *oauthToken, repo *domain.RepositoryIdentifier) (bool, error)
	GetRepositoryPermissions(ctx context.Context, token *oauthToken, repo *domain.RepositoryIdentifier) (domain.RepositoryPermissions, error)
}

type GitHubOAuthProviderAdapter struct {
	provider GitHubOAuthProviderInternal
}

func NewGitHubOAuthProviderAdapter(provider GitHubOAuthProviderInternal) *GitHubOAuthProviderAdapter {
	if provider == nil {
		panic("NewGitHubOAuthProviderAdapter: provider is nil")
	}
	return &GitHubOAuthProviderAdapter{
		provider: provider,
	}
}

func (a *GitHubOAuthProviderAdapter) SetRedirectURI(redirectURI string) {
	a.provider.SetRedirectURI(redirectURI)
}

func (a *GitHubOAuthProviderAdapter) GetAuthorizationURL(state string) string {
	return a.provider.GetAuthorizationURL(state)
}

func (a *GitHubOAuthProviderAdapter) ExchangeCode(ctx context.Context, code string) (*usecase.OAuthTokenResult, error) {
	token, err := a.provider.ExchangeCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return &usecase.OAuthTokenResult{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Scope:       token.Scope,
	}, nil
}

func (a *GitHubOAuthProviderAdapter) GetUserInfo(ctx context.Context, token *usecase.OAuthTokenResult) (*usecase.GitHubUserResult, error) {
	internalToken := &oauthToken{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Scope:       token.Scope,
	}
	user, err := a.provider.GetUserInfo(ctx, internalToken)
	if err != nil {
		return nil, err
	}
	return &usecase.GitHubUserResult{
		ID:    user.ID,
		Login: user.Login,
		Name:  user.Name,
	}, nil
}

func (a *GitHubOAuthProviderAdapter) CanAccessRepository(ctx context.Context, token *usecase.OAuthTokenResult, repo *domain.RepositoryIdentifier) (bool, error) {
	internalToken := &oauthToken{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Scope:       token.Scope,
	}
	return a.provider.CanAccessRepository(ctx, internalToken, repo)
}

func (a *GitHubOAuthProviderAdapter) GetRepositoryPermissions(ctx context.Context, token *usecase.OAuthTokenResult, repo *domain.RepositoryIdentifier) (domain.RepositoryPermissions, error) {
	internalToken := &oauthToken{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Scope:       token.Scope,
	}
	return a.provider.GetRepositoryPermissions(ctx, internalToken, repo)
}
