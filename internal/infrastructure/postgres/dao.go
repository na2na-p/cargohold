package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PoolInterface interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Close()
}

type LFSObjectDAO struct {
	pool PoolInterface
}

type LFSObjectRow struct {
	OID        string
	Size       int64
	HashAlgo   string
	StorageKey string
	Uploaded   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewLFSObjectDAO(pool PoolInterface) *LFSObjectDAO {
	return &LFSObjectDAO{
		pool: pool,
	}
}

func (dao *LFSObjectDAO) FindByOID(ctx context.Context, oid string) (*LFSObjectRow, error) {
	query := `
		SELECT oid, size, hash_algo, storage_key, uploaded, created_at, updated_at
		FROM lfs_objects
		WHERE oid = $1
	`

	row := dao.pool.QueryRow(ctx, query, oid)

	var result LFSObjectRow
	err := row.Scan(
		&result.OID,
		&result.Size,
		&result.HashAlgo,
		&result.StorageKey,
		&result.Uploaded,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	return &result, nil
}

func (dao *LFSObjectDAO) Insert(ctx context.Context, row *LFSObjectRow) error {
	query := `
		INSERT INTO lfs_objects (oid, size, hash_algo, storage_key, uploaded, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := dao.pool.Exec(ctx, query,
		row.OID,
		row.Size,
		row.HashAlgo,
		row.StorageKey,
		row.Uploaded,
		row.CreatedAt,
		row.UpdatedAt,
	)

	return err
}

func (dao *LFSObjectDAO) Update(ctx context.Context, row *LFSObjectRow) error {
	query := `
		UPDATE lfs_objects
		SET size = $2, hash_algo = $3, storage_key = $4, uploaded = $5, updated_at = $6
		WHERE oid = $1
	`

	result, err := dao.pool.Exec(ctx, query,
		row.OID,
		row.Size,
		row.HashAlgo,
		row.StorageKey,
		row.Uploaded,
		row.UpdatedAt,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (dao *LFSObjectDAO) Exists(ctx context.Context, oid string) (bool, error) {
	query := `
		SELECT EXISTS(SELECT 1 FROM lfs_objects WHERE oid = $1)
	`

	var exists bool
	err := dao.pool.QueryRow(ctx, query, oid).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
