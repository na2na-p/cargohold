package infrastructure

import (
	"context"
	"time"

	"github.com/na2na-p/cargohold/internal/domain"
)

type CachingLFSObjectRepository struct {
	repo         domain.LFSObjectRepository
	cacheClient  domain.CacheClient
	keyGenerator domain.CacheKeyGenerator
	cacheConfig  domain.CacheConfig
}

func NewCachingLFSObjectRepository(
	repo domain.LFSObjectRepository,
	cacheClient domain.CacheClient,
	keyGenerator domain.CacheKeyGenerator,
	cacheConfig domain.CacheConfig,
) *CachingLFSObjectRepository {
	return &CachingLFSObjectRepository{
		repo:         repo,
		cacheClient:  cacheClient,
		keyGenerator: keyGenerator,
		cacheConfig:  cacheConfig,
	}
}

func (r *CachingLFSObjectRepository) FindByOID(ctx context.Context, oid domain.OID) (*domain.LFSObject, error) {
	cacheKey := r.keyGenerator.MetadataKey(oid.String())

	var cached cachedMetadata
	err := r.cacheClient.GetJSON(ctx, cacheKey, &cached)
	if err == nil {
		cachedOID, oidErr := domain.NewOID(cached.OID)
		cachedSize, sizeErr := domain.NewSize(cached.Size)
		if oidErr == nil && sizeErr == nil {
			obj, reconstructErr := domain.ReconstructLFSObject(
				cachedOID,
				cachedSize,
				cached.HashAlgo,
				cached.StorageKey,
				cached.Uploaded,
				cached.CreatedAt,
				cached.UpdatedAt,
			)
			if reconstructErr == nil {
				return obj, nil
			}
		}
	}

	obj, err := r.repo.FindByOID(ctx, oid)
	if err != nil {
		return nil, err
	}

	r.cacheMetadata(ctx, oid, obj)

	return obj, nil
}

func (r *CachingLFSObjectRepository) Save(ctx context.Context, obj *domain.LFSObject) error {
	if err := r.repo.Save(ctx, obj); err != nil {
		return err
	}

	r.cacheMetadata(ctx, obj.OID(), obj)

	return nil
}

func (r *CachingLFSObjectRepository) Update(ctx context.Context, obj *domain.LFSObject) error {
	if err := r.repo.Update(ctx, obj); err != nil {
		return err
	}

	r.cacheMetadata(ctx, obj.OID(), obj)

	return nil
}

func (r *CachingLFSObjectRepository) ExistsByOID(ctx context.Context, oid domain.OID) (bool, error) {
	cacheKey := r.keyGenerator.MetadataKey(oid.String())

	exists, err := r.cacheClient.Exists(ctx, cacheKey)
	if err == nil && exists {
		return true, nil
	}

	return r.repo.ExistsByOID(ctx, oid)
}

func (r *CachingLFSObjectRepository) DeleteBatchUploadKey(ctx context.Context, oid string) error {
	batchKey := r.keyGenerator.BatchUploadKey(oid)
	return r.cacheClient.Delete(ctx, batchKey)
}

func (r *CachingLFSObjectRepository) cacheMetadata(ctx context.Context, oid domain.OID, obj *domain.LFSObject) {
	cacheKey := r.keyGenerator.MetadataKey(oid.String())
	cached := cachedMetadata{
		OID:        obj.OID().String(),
		Size:       obj.Size().Int64(),
		HashAlgo:   obj.HashAlgo(),
		StorageKey: obj.GetStorageKey(),
		Uploaded:   obj.IsUploaded(),
		CreatedAt:  obj.CreatedAt(),
		UpdatedAt:  obj.UpdatedAt(),
	}
	_ = r.cacheClient.SetJSON(ctx, cacheKey, cached, r.cacheConfig.MetadataTTL())
}

type cachedMetadata struct {
	OID        string    `json:"oid"`
	Size       int64     `json:"size"`
	HashAlgo   string    `json:"hash_algo"`
	StorageKey string    `json:"storage_key"`
	Uploaded   bool      `json:"uploaded"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
