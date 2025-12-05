package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// AccessPolicyDAO はlfs_object_access_policiesテーブルへのデータアクセスを提供する
type AccessPolicyDAO struct {
	pool PoolInterface
}

// AccessPolicyRow はlfs_object_access_policiesテーブルの1行を表す
type AccessPolicyRow struct {
	ID           int64
	LfsObjectOid string
	Repository   string
	CreatedAt    time.Time
}

// NewAccessPolicyDAO は新しいAccessPolicyDAOを作成する
func NewAccessPolicyDAO(pool PoolInterface) *AccessPolicyDAO {
	return &AccessPolicyDAO{
		pool: pool,
	}
}

// FindByOID は指定されたLFS Object OIDに対応するレコードを取得する
func (dao *AccessPolicyDAO) FindByOID(ctx context.Context, oid string) (*AccessPolicyRow, error) {
	query := `
		SELECT id, lfs_object_oid, repository, created_at
		FROM lfs_object_access_policies
		WHERE lfs_object_oid = $1
	`

	row := dao.pool.QueryRow(ctx, query, oid)

	var result AccessPolicyRow
	err := row.Scan(
		&result.ID,
		&result.LfsObjectOid,
		&result.Repository,
		&result.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	return &result, nil
}

// Upsert は新しいレコードを挿入するか、既存のレコードを更新する（UPSERT処理）
func (dao *AccessPolicyDAO) Upsert(ctx context.Context, row *AccessPolicyRow) error {
	query := `
		INSERT INTO lfs_object_access_policies (lfs_object_oid, repository, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (lfs_object_oid)
		DO UPDATE SET repository = EXCLUDED.repository
	`

	_, err := dao.pool.Exec(ctx, query,
		row.LfsObjectOid,
		row.Repository,
		row.CreatedAt,
	)

	return err
}

// Delete は指定されたLFS Object OIDに対応するレコードを削除する
func (dao *AccessPolicyDAO) Delete(ctx context.Context, oid string) error {
	query := `
		DELETE FROM lfs_object_access_policies
		WHERE lfs_object_oid = $1
	`

	result, err := dao.pool.Exec(ctx, query, oid)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
