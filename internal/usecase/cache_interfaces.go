//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_cache_interfaces.go -package=usecase
package usecase

import (
	"context"
	"time"
)

type CacheKeyGenerator interface {
	MetadataKey(oid string) string
	SessionKey(sessionID string) string
	BatchUploadKey(oid string) string
}

type CacheConfig interface {
	MetadataTTL() time.Duration
}

type CacheClient interface {
	Exists(ctx context.Context, key string) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetJSON(ctx context.Context, key string, dest interface{}) error
	SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
