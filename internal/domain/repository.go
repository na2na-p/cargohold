//go:generate mockgen -source=$GOFILE -destination=../../tests/domain/mock_repository.go -package=domain
package domain

import "context"

type LFSObjectRepository interface {
	FindByOID(ctx context.Context, oid OID) (*LFSObject, error)
	Save(ctx context.Context, obj *LFSObject) error
	Update(ctx context.Context, obj *LFSObject) error
	ExistsByOID(ctx context.Context, oid OID) (bool, error)
}
