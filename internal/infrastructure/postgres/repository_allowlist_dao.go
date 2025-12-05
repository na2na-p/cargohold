package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// RepositoryAllowlistDAO はrepository_allowlistテーブルへのデータアクセスを提供する
type RepositoryAllowlistDAO struct {
	pool PoolInterface
}

// RepositoryAllowlistRow はrepository_allowlistテーブルの1行を表す
type RepositoryAllowlistRow struct {
	ID         int64
	Repository string
	CreatedAt  time.Time
}

// NewRepositoryAllowlistDAO は新しいRepositoryAllowlistDAOを作成する
func NewRepositoryAllowlistDAO(pool PoolInterface) *RepositoryAllowlistDAO {
	return &RepositoryAllowlistDAO{
		pool: pool,
	}
}

// Exists は指定されたリポジトリが許可リストに存在するかを確認する
func (dao *RepositoryAllowlistDAO) Exists(ctx context.Context, repository string) (bool, error) {
	query := `
		SELECT EXISTS(SELECT 1 FROM repository_allowlist WHERE repository = $1)
	`

	var exists bool
	err := dao.pool.QueryRow(ctx, query, repository).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// Insert は新しいリポジトリを許可リストに追加する
func (dao *RepositoryAllowlistDAO) Insert(ctx context.Context, repository string) error {
	query := `
		INSERT INTO repository_allowlist (repository)
		VALUES ($1)
		ON CONFLICT (repository) DO NOTHING
	`

	_, err := dao.pool.Exec(ctx, query, repository)
	return err
}

// Delete は指定されたリポジトリを許可リストから削除する
func (dao *RepositoryAllowlistDAO) Delete(ctx context.Context, repository string) error {
	query := `
		DELETE FROM repository_allowlist WHERE repository = $1
	`

	result, err := dao.pool.Exec(ctx, query, repository)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// FindAll は許可リポジトリの一覧を取得する
func (dao *RepositoryAllowlistDAO) FindAll(ctx context.Context) ([]string, error) {
	query := `
		SELECT repository FROM repository_allowlist ORDER BY repository
	`

	rows, err := dao.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repositories []string
	for rows.Next() {
		var repo string
		if err := rows.Scan(&repo); err != nil {
			return nil, err
		}
		repositories = append(repositories, repo)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return repositories, nil
}
