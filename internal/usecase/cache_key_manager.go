//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_cache_key_manager.go -package=usecase
package usecase

import "context"

type CacheKeyManager interface {
	DeleteBatchUploadKey(ctx context.Context, oid string) error
}
