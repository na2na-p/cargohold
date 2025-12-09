//go:generate mockgen -source=$GOFILE -destination=../../../tests/infrastructure/oidc/mock_cache_client.go -package=oidc
package oidc

import (
	"context"
	"time"
)

type CacheClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	GetJSON(ctx context.Context, key string, dest interface{}) error
	SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}
