//go:generate mockgen -source=$GOFILE -destination=mock_repository_checker_test.go -package=oidc
package oidc

import (
	"context"

	"github.com/na2na-p/cargohold/internal/domain"
)

type RepositoryChecker interface {
	CanAccessRepository(ctx context.Context, token *oauthToken, repo *domain.RepositoryIdentifier) (bool, error)
	GetRepositoryPermissions(ctx context.Context, token *oauthToken, repo *domain.RepositoryIdentifier) (domain.RepositoryPermissions, error)
}
