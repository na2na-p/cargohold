//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_repository_allowlist_cache_client.go -package=usecase
package usecase

import (
	"context"
	"time"
)

type RepositoryAllowlistCacheClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
