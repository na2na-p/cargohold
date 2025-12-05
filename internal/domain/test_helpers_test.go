package domain_test

import (
	"context"
	"testing"

	"github.com/na2na-p/cargohold/internal/domain"
)

func NewTestLFSObject(t *testing.T, ctx context.Context) *domain.LFSObject {
	t.Helper()

	oid, err := domain.NewOID("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	if err != nil {
		t.Fatalf("NewOID() failed: %v", err)
	}

	size, err := domain.NewSize(1024)
	if err != nil {
		t.Fatalf("NewSize() failed: %v", err)
	}

	hashAlgo, err := domain.NewHashAlgorithm("sha256")
	if err != nil {
		t.Fatalf("NewHashAlgorithm() failed: %v", err)
	}

	obj, err := domain.NewLFSObject(ctx, oid, size, hashAlgo, "test/storage/key")
	if err != nil {
		t.Fatalf("NewLFSObject() failed: %v", err)
	}

	return obj
}

func NewTestLFSObjectWithParams(t *testing.T, ctx context.Context, oidString string, sizeValue int64, hashAlgoValue, storageKey string) *domain.LFSObject {
	t.Helper()

	oid, err := domain.NewOID(oidString)
	if err != nil {
		t.Fatalf("NewOID() failed: %v", err)
	}

	size, err := domain.NewSize(sizeValue)
	if err != nil {
		t.Fatalf("NewSize() failed: %v", err)
	}

	hashAlgo, err := domain.NewHashAlgorithm(hashAlgoValue)
	if err != nil {
		t.Fatalf("NewHashAlgorithm() failed: %v", err)
	}

	obj, err := domain.NewLFSObject(ctx, oid, size, hashAlgo, storageKey)
	if err != nil {
		t.Fatalf("NewLFSObject() failed: %v", err)
	}

	return obj
}

func NewTestOID(t *testing.T) domain.OID {
	t.Helper()

	oid, err := domain.NewOID("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	if err != nil {
		t.Fatalf("NewOID() failed: %v", err)
	}

	return oid
}

func NewTestOIDWithValue(t *testing.T, value string) domain.OID {
	t.Helper()

	oid, err := domain.NewOID(value)
	if err != nil {
		t.Fatalf("NewOID() failed: %v", err)
	}

	return oid
}

func NewTestSize(t *testing.T) domain.Size {
	t.Helper()

	size, err := domain.NewSize(1024)
	if err != nil {
		t.Fatalf("NewSize() failed: %v", err)
	}

	return size
}

func NewTestSizeWithValue(t *testing.T, value int64) domain.Size {
	t.Helper()

	size, err := domain.NewSize(value)
	if err != nil {
		t.Fatalf("NewSize() failed: %v", err)
	}

	return size
}
