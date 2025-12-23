//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_auth_usecase.go -package=usecase
package usecase

import (
	"context"

	"github.com/na2na-p/cargohold/internal/domain"
)

type AuthUseCase struct {
	githubOIDCUseCase  *GitHubOIDCUseCase
	sessionAuthUseCase *SessionAuthUseCase
}

func NewAuthUseCase(
	githubProvider GitHubOIDCProvider,
	repoAllowlistRepo domain.RepositoryAllowlistRepository,
	redisClient domain.CacheClient,
	keyGenerator domain.CacheKeyGenerator,
) *AuthUseCase {
	var githubOIDCUseCase *GitHubOIDCUseCase
	if githubProvider != nil && repoAllowlistRepo != nil {
		githubOIDCUseCase = NewGitHubOIDCUseCase(githubProvider, repoAllowlistRepo)
	}
	return &AuthUseCase{
		githubOIDCUseCase:  githubOIDCUseCase,
		sessionAuthUseCase: NewSessionAuthUseCase(redisClient, keyGenerator),
	}
}

func (uc *AuthUseCase) AuthenticateSession(ctx context.Context, sessionID string) (*domain.UserInfo, error) {
	return uc.sessionAuthUseCase.Authenticate(ctx, sessionID)
}

func (uc *AuthUseCase) AuthenticateGitHubOIDC(ctx context.Context, token string) (*domain.UserInfo, error) {
	if uc.githubOIDCUseCase == nil {
		return nil, ErrGitHubOIDCNotConfigured
	}
	return uc.githubOIDCUseCase.Authenticate(ctx, token)
}
