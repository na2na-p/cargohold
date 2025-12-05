package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/na2na-p/cargohold/internal/domain"
)

type LFSObjectRepositoryImpl struct {
	dao *LFSObjectDAO
}

func NewLFSObjectRepository(pool PoolInterface) domain.LFSObjectRepository {
	return &LFSObjectRepositoryImpl{
		dao: NewLFSObjectDAO(pool),
	}
}

func (r *LFSObjectRepositoryImpl) FindByOID(ctx context.Context, oid domain.OID) (*domain.LFSObject, error) {
	row, err := r.dao.FindByOID(ctx, oid.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return rowToDomain(row)
}

func (r *LFSObjectRepositoryImpl) Save(ctx context.Context, obj *domain.LFSObject) error {
	row := domainToRow(obj)
	return r.dao.Insert(ctx, row)
}

func (r *LFSObjectRepositoryImpl) Update(ctx context.Context, obj *domain.LFSObject) error {
	row := domainToRow(obj)
	err := r.dao.Update(ctx, row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *LFSObjectRepositoryImpl) ExistsByOID(ctx context.Context, oid domain.OID) (bool, error) {
	return r.dao.Exists(ctx, oid.String())
}

func rowToDomain(row *LFSObjectRow) (*domain.LFSObject, error) {
	oid, err := domain.NewOID(row.OID)
	if err != nil {
		return nil, err
	}

	size, err := domain.NewSize(row.Size)
	if err != nil {
		return nil, err
	}

	return domain.ReconstructLFSObject(
		oid,
		size,
		row.HashAlgo,
		row.StorageKey,
		row.Uploaded,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func domainToRow(obj *domain.LFSObject) *LFSObjectRow {
	return &LFSObjectRow{
		OID:        obj.OID().String(),
		Size:       obj.Size().Int64(),
		HashAlgo:   obj.HashAlgo(),
		StorageKey: obj.GetStorageKey(),
		Uploaded:   obj.IsUploaded(),
		CreatedAt:  obj.CreatedAt(),
		UpdatedAt:  obj.UpdatedAt(),
	}
}
