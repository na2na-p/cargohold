//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_external_interfaces.go -package=usecase
package usecase

import (
	"context"
	"io"
	"time"
)

type StorageKeyGenerator interface {
	GenerateStorageKey(oid, hashAlgo string) (string, error)
}

type S3Client interface {
	GeneratePutURL(ctx context.Context, key string, ttl time.Duration) (string, error)
	GenerateGetURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type ObjectStorage interface {
	PutObject(ctx context.Context, key string, body io.Reader, contentLength int64) error
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
}

type ActionURLGenerator interface {
	GenerateUploadURL(baseURL, owner, repo, oid string) string
	GenerateDownloadURL(baseURL, owner, repo, oid string) string
}
