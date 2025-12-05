//go:generate mockgen -source=$GOFILE -destination=../../tests/domain/mock_access_policy_repository.go -package=domain
package domain

import "context"

type AccessPolicyRepository interface {
	FindByOID(ctx context.Context, oid OID) (*AccessPolicy, error)
	Save(ctx context.Context, policy *AccessPolicy) error
	Delete(ctx context.Context, oid OID) error
}
