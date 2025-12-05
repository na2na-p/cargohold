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
	allowedRedirectURIs []string
}

func NewGitHubOAuthUseCase(
	oauthProvider GitHubOAuthProviderInterface,
	sessionStore SessionStoreInterface,
	stateStore OAuthStateStoreInterface,
	allowedRedirectURIs []string,
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
	if len(allowedRedirectURIs) == 0 {
		return nil, fmt.Errorf("allowedRedirectURIs is empty")
	}
	return &GitHubOAuthUseCase{
		oauthProvider:       oauthProvider,
		sessionStore:        sessionStore,
		stateStore:          stateStore,
		allowedRedirectURIs: allowedRedirectURIs,
	}, nil
}

var defaultScopes = []string{"repo"}

func (u *GitHubOAuthUseCase) StartAuthentication(
	ctx context.Context,
	repository *domain.RepositoryIdentifier,
	redirectURI string,
) (string, error) {
	if repository == nil {
		return "", fmt.Errorf("%w: repository is nil", ErrInvalidRepository)
	}

	if redirectURI == "" {
		return "", fmt.Errorf("%w: redirectURI is empty", ErrInvalidRedirectURI)
	}

	isAllowed := false
	for _, allowed := range u.allowedRedirectURIs {
		if redirectURI == allowed {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return "", fmt.Errorf("%w: redirectURI not in allowed list", ErrInvalidRedirectURI)
	}

	state := uuid.New().String()

	stateData := &OAuthStateData{
		Repository:  repository.FullName(),
		RedirectURI: redirectURI,
	}

	if err := u.stateStore.SaveState(ctx, state, stateData, OIDCStateTTL); err != nil {
		return "", fmt.Errorf("%w: %v", ErrStateSaveFailed, err)
	}

	authURL := u.oauthProvider.GetAuthorizationURL(state, defaultScopes)

	return authURL, nil
}

func (u *GitHubOAuthUseCase) HandleCallback(
	ctx context.Context,
	code string,
	state string,
) (string, error) {
	if code == "" {
		return "", fmt.Errorf("%w: missing code", ErrInvalidCode)
	}

	if state == "" {
		return "", fmt.Errorf("%w: missing state", ErrInvalidState)
	}

	stateData, err := u.stateStore.GetAndDeleteState(ctx, state)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidState, err)
	}

	repository, err := domain.NewRepositoryIdentifier(stateData.Repository)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidRepository, err)
	}

	token, err := u.oauthProvider.ExchangeCode(ctx, code)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCodeExchangeFailed, err)
	}

	githubUser, err := u.oauthProvider.GetUserInfo(ctx, token)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUserInfoFailed, err)
	}

	canAccess, err := u.oauthProvider.CanAccessRepository(ctx, token, repository)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrRepositoryAccessCheckFailed, err)
	}

	if !canAccess {
		return "", fmt.Errorf("%w: user cannot access repository %s", ErrRepositoryAccessDenied, repository.FullName())
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
		return "", fmt.Errorf("%w: %v", ErrUserInfoCreationFailed, err)
	}

	sessionID, err := u.sessionStore.CreateSession(ctx, userInfo, SessionTTL)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSessionCreationFailed, err)
	}

	return sessionID, nil
}
