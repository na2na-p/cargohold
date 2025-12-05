package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/na2na-p/cargohold/internal/domain"
	infra "github.com/na2na-p/cargohold/internal/infrastructure"
)

// RepositoryAllowlistRepositoryImpl はRepositoryAllowlistRepositoryのPostgreSQL実装
type RepositoryAllowlistRepositoryImpl struct {
	dao *RepositoryAllowlistDAO
}

// NewRepositoryAllowlistRepository は新しいRepositoryAllowlistRepositoryを作成する
func NewRepositoryAllowlistRepository(pool PoolInterface) domain.RepositoryAllowlistRepository {
	return &RepositoryAllowlistRepositoryImpl{
		dao: NewRepositoryAllowlistDAO(pool),
	}
}

// IsAllowed は指定されたリポジトリが許可されているかをチェックする
func (r *RepositoryAllowlistRepositoryImpl) IsAllowed(ctx context.Context, repository *domain.AllowedRepository) (bool, error) {
	if repository == nil {
		return false, fmt.Errorf("repository is nil")
	}
	return r.dao.Exists(ctx, repository.String())
}

// Add は許可リポジトリリストにリポジトリを追加する
func (r *RepositoryAllowlistRepositoryImpl) Add(ctx context.Context, repository *domain.AllowedRepository) error {
	if repository == nil {
		return fmt.Errorf("repository is nil")
	}
	return r.dao.Insert(ctx, repository.String())
}

// Remove は許可リポジトリリストからリポジトリを削除する
func (r *RepositoryAllowlistRepositoryImpl) Remove(ctx context.Context, repository *domain.AllowedRepository) error {
	if repository == nil {
		return fmt.Errorf("repository is nil")
	}
	err := r.dao.Delete(ctx, repository.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return infra.ErrNotFound
		}
		return err
	}
	return nil
}

// List は許可リポジトリの一覧を取得する
func (r *RepositoryAllowlistRepositoryImpl) List(ctx context.Context) ([]*domain.AllowedRepository, error) {
	repoStrings, err := r.dao.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	repos := make([]*domain.AllowedRepository, 0, len(repoStrings))
	for _, repoStr := range repoStrings {
		repo, err := domain.NewAllowedRepositoryFromString(repoStr)
		if err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}

	return repos, nil
}
