package usecase

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/na2na-p/cargohold/internal/domain"
)

type GitHubOAuthUseCase struct {
	oauthProvider       GitHubOAuthProviderInterface
	sessionStore        SessionStoreInterface
	stateStore          OAuthStateStoreInterface
	allowedRedirectURIs *domain.AllowedRedirectURIs
}

func NewGitHubOAuthUseCase(
	oauthProvider GitHubOAuthProviderInterface,
	sessionStore SessionStoreInterface,
	stateStore OAuthStateStoreInterface,
	allowedRedirectURIs *domain.AllowedRedirectURIs,
) (*GitHubOAuthUseCase, error) {
	if oauthProvider == nil {
		return nil, fmt.Errorf("oauthProvider is nil")
	}
	if sessionStore == nil {
		return nil, fmt.Errorf("sessionStore is nil")
	}
	if stateStore == nil {
		return nil, fmt.Errorf("stateStore is nil")
	}
	if allowedRedirectURIs == nil {
		return nil, fmt.Errorf("allowedRedirectURIs is nil")
	}
	return &GitHubOAuthUseCase{
		oauthProvider:       oauthProvider,
		sessionStore:        sessionStore,
		stateStore:          stateStore,
		allowedRedirectURIs: allowedRedirectURIs,
	}, nil
}

func (u *GitHubOAuthUseCase) StartAuthentication(
	ctx context.Context,
	repository *domain.RepositoryIdentifier,
	redirectURI string,
	shell domain.ShellType,
) (string, error) {
	if repository == nil {
		return "", fmt.Errorf("%w: repository is nil", ErrInvalidRepository)
	}

	if redirectURI == "" {
		return "", fmt.Errorf("%w: redirectURI is empty", ErrInvalidRedirectURI)
	}

	if !u.allowedRedirectURIs.Contains(redirectURI) {
		return "", fmt.Errorf("%w: redirectURI not in allowed list", ErrInvalidRedirectURI)
	}

	state := uuid.New().String()

	stateData := domain.NewOAuthState(repository.FullName(), redirectURI, shell)

	if err := u.stateStore.SaveState(ctx, state, stateData, OIDCStateTTL); err != nil {
		return "", fmt.Errorf("%w: %v", ErrStateSaveFailed, err)
	}

	u.oauthProvider.SetRedirectURI(redirectURI)
	authURL := u.oauthProvider.GetAuthorizationURL(state)

	return authURL, nil
}

func (u *GitHubOAuthUseCase) HandleCallback(
	ctx context.Context,
	code string,
	state string,
) (string, domain.ShellType, error) {
	if code == "" {
		return "", domain.ShellType{}, fmt.Errorf("%w: missing code", ErrInvalidCode)
	}

	if state == "" {
		return "", domain.ShellType{}, fmt.Errorf("%w: missing state", ErrInvalidState)
	}

	stateData, err := u.stateStore.GetAndDeleteState(ctx, state)
	if err != nil {
		return "", domain.ShellType{}, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}

	repository, err := domain.NewRepositoryIdentifier(stateData.Repository())
	if err != nil {
		return "", domain.ShellType{}, fmt.Errorf("%w: %v", ErrInvalidRepository, err)
	}

	u.oauthProvider.SetRedirectURI(stateData.RedirectURI())
	token, err := u.oauthProvider.ExchangeCode(ctx, code)
	if err != nil {
		return "", domain.ShellType{}, fmt.Errorf("%w: %v", ErrCodeExchangeFailed, err)
	}

	githubUser, err := u.oauthProvider.GetUserInfo(ctx, token)
	if err != nil {
		return "", domain.ShellType{}, fmt.Errorf("%w: %v", ErrUserInfoFailed, err)
	}

	permissions, err := u.oauthProvider.GetRepositoryPermissions(ctx, token, repository)
	if err != nil {
		return "", domain.ShellType{}, fmt.Errorf("%w: %v", ErrRepositoryAccessCheckFailed, err)
	}

	if !permissions.CanDownload() {
		return "", domain.ShellType{}, fmt.Errorf("%w: user cannot access repository %s", ErrRepositoryAccessDenied, repository.FullName())
	}

	userInfo, err := domain.NewUserInfo(
		strconv.FormatInt(githubUser.ID, 10),
		"",
		githubUser.Name,
		domain.ProviderTypeGitHub,
		repository,
		"",
	)
	if err != nil {
		return "", domain.ShellType{}, fmt.Errorf("%w: %v", ErrUserInfoCreationFailed, err)
	}

	userInfo.SetPermissions(&permissions)

	sessionID, err := u.sessionStore.CreateSession(ctx, userInfo, SessionTTL)
	if err != nil {
		return "", domain.ShellType{}, fmt.Errorf("%w: %v", ErrSessionCreationFailed, err)
	}

	return sessionID, stateData.Shell(), nil
}
