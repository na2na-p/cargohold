//go:generate mockgen -source=$GOFILE -destination=../../tests/domain/mock_repository_allowlist.go -package=domain
package domain

import "context"

type RepositoryAllowlistRepository interface {
	IsAllowed(ctx context.Context, repository *AllowedRepository) (bool, error)
	Add(ctx context.Context, repository *AllowedRepository) error
	Remove(ctx context.Context, repository *AllowedRepository) error
	List(ctx context.Context) ([]*AllowedRepository, error)
}
