package domain

import (
	"context"
	"time"

	"github.com/newmo-oss/ctxtime"
)

type LFSObject struct {
	oid        OID
	size       Size
	hashAlgo   HashAlgorithm
	storageKey StorageKey
	uploaded   bool
	createdAt  time.Time
	updatedAt  time.Time
}

func NewLFSObject(ctx context.Context, oid OID, size Size, hashAlgo HashAlgorithm, storageKey string) (*LFSObject, error) {
	sk, err := NewStorageKey(storageKey)
	if err != nil {
		return nil, err
	}
	now := ctxtime.Now(ctx)
	return &LFSObject{
		oid:        oid,
		size:       size,
		hashAlgo:   hashAlgo,
		storageKey: sk,
		uploaded:   false,
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

func ReconstructLFSObject(oid OID, size Size, hashAlgo string, storageKey string, uploaded bool, createdAt, updatedAt time.Time) (*LFSObject, error) {
	hashAlgorithm, err := NewHashAlgorithm(hashAlgo)
	if err != nil {
		return nil, err
	}

	sk, err := NewStorageKey(storageKey)
	if err != nil {
		return nil, err
	}

	return &LFSObject{
		oid:        oid,
		size:       size,
		hashAlgo:   hashAlgorithm,
		storageKey: sk,
		uploaded:   uploaded,
		createdAt:  createdAt,
		updatedAt:  updatedAt,
	}, nil
}

func (o *LFSObject) MarkAsUploaded(ctx context.Context) {
	o.uploaded = true
	o.updatedAt = ctxtime.Now(ctx)
}

func (o *LFSObject) IsUploaded() bool {
	return o.uploaded
}

func (o *LFSObject) GetStorageKey() string {
	return o.storageKey.String()
}

func (o *LFSObject) OID() OID {
	return o.oid
}

func (o *LFSObject) ID() OID {
	return o.oid
}

func (o *LFSObject) Size() Size {
	return o.size
}

func (o *LFSObject) HashAlgo() string {
	return o.hashAlgo.String()
}

func (o *LFSObject) CreatedAt() time.Time {
	return o.createdAt
}

func (o *LFSObject) UpdatedAt() time.Time {
	return o.updatedAt
}
