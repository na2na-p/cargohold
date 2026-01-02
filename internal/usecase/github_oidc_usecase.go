//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_github_oidc_usecase.go -package=usecase
package usecase

import (
	"context"
	"fmt"

	"github.com/na2na-p/cargohold/internal/domain"
)

type GitHubOIDCProvider interface {
	VerifyIDToken(ctx context.Context, token string) (*domain.GitHubUserInfo, error)
}

type GitHubOIDCUseCase struct {
	githubProvider    GitHubOIDCProvider
	repoAllowlistRepo domain.RepositoryAllowlistRepository
}

func NewGitHubOIDCUseCase(
	githubProvider GitHubOIDCProvider,
	repoAllowlistRepo domain.RepositoryAllowlistRepository,
) *GitHubOIDCUseCase {
	return &GitHubOIDCUseCase{
		githubProvider:    githubProvider,
		repoAllowlistRepo: repoAllowlistRepo,
	}
}

func (uc *GitHubOIDCUseCase) Authenticate(ctx context.Context, token string) (*domain.UserInfo, error) {
	githubUserInfo, err := uc.githubProvider.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if githubUserInfo == nil {
		return nil, fmt.Errorf("github provider returned nil user info")
	}

	allowedRepo, err := domain.NewAllowedRepositoryFromString(githubUserInfo.Repository())
	if err != nil {
		return nil, fmt.Errorf("リポジトリ形式が不正です: %w", err)
	}

	allowed, err := uc.repoAllowlistRepo.IsAllowed(ctx, allowedRepo)
	if err != nil {
		return nil, fmt.Errorf("リポジトリ許可チェックに失敗しました: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("%w: repository=%s", ErrInvalidRepository, githubUserInfo.Repository())
	}

	userInfo, err := githubUserInfo.ToUserInfo()
	if err != nil {
		return nil, err
	}

	fullPerms := domain.NewRepositoryPermissions(true, true, true, true, true)
	userInfo.SetPermissions(&fullPerms)

	return userInfo, nil
}
