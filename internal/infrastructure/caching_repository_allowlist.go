package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
	"github.com/na2na-p/cargohold/internal/usecase"
)

const (
	repositoryCacheTTL = 5 * time.Minute
)

type CachingRepositoryAllowlist struct {
	pgRepo      domain.RepositoryAllowlistRepository
	cacheClient usecase.RepositoryAllowlistCacheClient
	cacheTTL    time.Duration
}

func NewCachingRepositoryAllowlist(
	pgRepo domain.RepositoryAllowlistRepository,
	cacheClient usecase.RepositoryAllowlistCacheClient,
) *CachingRepositoryAllowlist {
	return &CachingRepositoryAllowlist{
		pgRepo:      pgRepo,
		cacheClient: cacheClient,
		cacheTTL:    repositoryCacheTTL,
	}
}

func NewCachingRepositoryAllowlistWithTTL(
	pgRepo domain.RepositoryAllowlistRepository,
	cacheClient usecase.RepositoryAllowlistCacheClient,
	ttl time.Duration,
) *CachingRepositoryAllowlist {
	return &CachingRepositoryAllowlist{
		pgRepo:      pgRepo,
		cacheClient: cacheClient,
		cacheTTL:    ttl,
	}
}

func (r *CachingRepositoryAllowlist) IsAllowed(ctx context.Context, repository *domain.AllowedRepository) (bool, error) {
	cacheKey := r.getCacheKey(repository)

	allowed, err := r.checkCache(ctx, cacheKey)
	if err == nil {
		return allowed, nil
	}

	allowed, err = r.pgRepo.IsAllowed(ctx, repository)
	if err != nil {
		return false, fmt.Errorf("リポジトリ許可チェックに失敗しました: %w", err)
	}

	r.setCache(ctx, cacheKey, allowed)

	return allowed, nil
}

func (r *CachingRepositoryAllowlist) Add(ctx context.Context, repository *domain.AllowedRepository) error {
	if err := r.pgRepo.Add(ctx, repository); err != nil {
		return fmt.Errorf("リポジトリの追加に失敗しました: %w", err)
	}

	cacheKey := r.getCacheKey(repository)
	r.setCache(ctx, cacheKey, true)

	return nil
}

func (r *CachingRepositoryAllowlist) Remove(ctx context.Context, repository *domain.AllowedRepository) error {
	if err := r.pgRepo.Remove(ctx, repository); err != nil {
		return fmt.Errorf("リポジトリの削除に失敗しました: %w", err)
	}

	cacheKey := r.getCacheKey(repository)
	_ = r.cacheClient.Delete(ctx, cacheKey)

	return nil
}

func (r *CachingRepositoryAllowlist) List(ctx context.Context) ([]*domain.AllowedRepository, error) {
	repositories, err := r.pgRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("リポジトリ一覧の取得に失敗しました: %w", err)
	}
	return repositories, nil
}

func (r *CachingRepositoryAllowlist) getCacheKey(repository *domain.AllowedRepository) string {
	return redis.OIDCGitHubRepoKey(repository.String())
}

func (r *CachingRepositoryAllowlist) checkCache(ctx context.Context, cacheKey string) (bool, error) {
	val, err := r.cacheClient.Get(ctx, cacheKey)
	if err != nil {
		return false, err
	}
	return val == "true", nil
}

func (r *CachingRepositoryAllowlist) setCache(ctx context.Context, cacheKey string, allowed bool) {
	value := "false"
	if allowed {
		value = "true"
	}
	_ = r.cacheClient.Set(ctx, cacheKey, value, r.cacheTTL)
}
