//go:generate mockgen -source=$GOFILE -destination=../../../tests/infrastructure/oidc/mock_repository_checker.go -package=oidc
package oidc

import (
	"context"

	"github.com/na2na-p/cargohold/internal/domain"
)

type RepositoryChecker interface {
	CanAccessRepository(ctx context.Context, token *OAuthToken, repo *domain.RepositoryIdentifier) (bool, error)
}
