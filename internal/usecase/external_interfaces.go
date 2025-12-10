//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_external_interfaces.go -package=usecase
package usecase

import (
	"context"
	"io"
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

type StorageKeyGenerator interface {
	GenerateStorageKey(oid, hashAlgo string) (string, error)
}

type S3Client interface {
	GeneratePutURL(ctx context.Context, key string, ttl time.Duration) (string, error)
	GenerateGetURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type ObjectStorage interface {
	PutObject(ctx context.Context, key string, body io.Reader) error
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
}

type ActionURLGenerator interface {
	GenerateUploadURL(baseURL, owner, repo, oid string) string
	GenerateDownloadURL(baseURL, owner, repo, oid string) string
}
