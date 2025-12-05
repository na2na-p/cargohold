package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/na2na-p/cargohold/internal/domain"
)

type AccessPolicyRepositoryImpl struct {
	dao *AccessPolicyDAO
}

func NewAccessPolicyRepository(pool PoolInterface) domain.AccessPolicyRepository {
	return &AccessPolicyRepositoryImpl{
		dao: NewAccessPolicyDAO(pool),
	}
}

func (r *AccessPolicyRepositoryImpl) FindByOID(ctx context.Context, oid domain.OID) (*domain.AccessPolicy, error) {
	row, err := r.dao.FindByOID(ctx, oid.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccessPolicyNotFound
		}
		return nil, err
	}

	return rowToAccessPolicy(row)
}

func (r *AccessPolicyRepositoryImpl) Save(ctx context.Context, policy *domain.AccessPolicy) error {
	row := accessPolicyToRow(policy)
	return r.dao.Upsert(ctx, row)
}

func (r *AccessPolicyRepositoryImpl) Delete(ctx context.Context, oid domain.OID) error {
	err := r.dao.Delete(ctx, oid.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAccessPolicyNotFound
		}
		return err
	}
	return nil
}

func rowToAccessPolicy(row *AccessPolicyRow) (*domain.AccessPolicy, error) {
	policyID, err := domain.NewAccessPolicyID(row.ID)
	if err != nil {
		return nil, err
	}

	oid, err := domain.NewOID(row.LfsObjectOid)
	if err != nil {
		return nil, err
	}

	repo, err := domain.NewRepositoryIdentifier(row.Repository)
	if err != nil {
		return nil, err
	}

	return domain.NewAccessPolicy(policyID, oid, repo, row.CreatedAt), nil
}

func accessPolicyToRow(policy *domain.AccessPolicy) *AccessPolicyRow {
	return &AccessPolicyRow{
		ID:           policy.ID().Int64(),
		LfsObjectOid: policy.OID().String(),
		Repository:   policy.Repository().FullName(),
		CreatedAt:    policy.CreatedAt(),
	}
}
