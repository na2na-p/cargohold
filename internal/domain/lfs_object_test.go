package domain_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/newmo-oss/ctxtime/ctxtimetest"
	"github.com/newmo-oss/testid"
)

func TestNewLFSObject(t *testing.T) {
	type args struct {
		oidString     string
		sizeValue     int64
		hashAlgoValue string
		storageKey    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "正常系: HashAlgorithm型を使用したLFSObjectの作成",
			args: args{
				oidString:     "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				sizeValue:     1024,
				hashAlgoValue: "sha256",
				storageKey:    "test/storage/key",
			},
			wantErr: nil,
		},
		{
			name: "異常系: 空のStorageKeyでエラーが返る",
			args: args{
				oidString:     "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				sizeValue:     1024,
				hashAlgoValue: "sha256",
				storageKey:    "",
			},
			wantErr: domain.ErrInvalidStorageKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, err := domain.NewOID(tt.args.oidString)
			if err != nil {
				t.Fatalf("NewOID() failed: %v", err)
			}

			size, err := domain.NewSize(tt.args.sizeValue)
			if err != nil {
				t.Fatalf("NewSize() failed: %v", err)
			}

			hashAlgo, err := domain.NewHashAlgorithm(tt.args.hashAlgoValue)
			if err != nil {
				t.Fatalf("NewHashAlgorithm() failed: %v", err)
			}

			fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
			tid := uuid.NewString()
			ctx := testid.WithValue(context.Background(), tid)
			ctxtimetest.SetFixedNow(t, ctx, fixedTime)

			obj, err := domain.NewLFSObject(ctx, oid, size, hashAlgo, tt.args.storageKey)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want error %v, but got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("want no error, but got %v", err)
			}

			if obj == nil {
				t.Fatal("NewLFSObject() returned nil")
			}

			if diff := cmp.Diff(oid.String(), obj.OID().String()); diff != "" {
				t.Errorf("OID() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(size.Int64(), obj.Size().Int64()); diff != "" {
				t.Errorf("Size() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.hashAlgoValue, obj.HashAlgo()); diff != "" {
				t.Errorf("HashAlgo() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.storageKey, obj.GetStorageKey()); diff != "" {
				t.Errorf("GetStorageKey() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(false, obj.IsUploaded()); diff != "" {
				t.Errorf("IsUploaded() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(fixedTime, obj.CreatedAt()); diff != "" {
				t.Errorf("CreatedAt() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(fixedTime, obj.UpdatedAt()); diff != "" {
				t.Errorf("UpdatedAt() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLFSObject_MarkAsUploaded(t *testing.T) {
	type args struct {
		oidString     string
		sizeValue     int64
		hashAlgoValue string
		storageKey    string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常系: MarkAsUploadedの正常動作",
			args: args{
				oidString:     "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				sizeValue:     1024,
				hashAlgoValue: "sha256",
				storageKey:    "test/key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, err := domain.NewOID(tt.args.oidString)
			if err != nil {
				t.Fatalf("NewOID() failed: %v", err)
			}

			size, err := domain.NewSize(tt.args.sizeValue)
			if err != nil {
				t.Fatalf("NewSize() failed: %v", err)
			}

			hashAlgo, err := domain.NewHashAlgorithm(tt.args.hashAlgoValue)
			if err != nil {
				t.Fatalf("NewHashAlgorithm() failed: %v", err)
			}

			initialTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
			tid := uuid.NewString()
			ctx := testid.WithValue(context.Background(), tid)
			ctxtimetest.SetFixedNow(t, ctx, initialTime)

			obj, err := domain.NewLFSObject(ctx, oid, size, hashAlgo, tt.args.storageKey)
			if err != nil {
				t.Fatalf("NewLFSObject() failed: %v", err)
			}

			if diff := cmp.Diff(false, obj.IsUploaded()); diff != "" {
				t.Errorf("初期状態でIsUploaded() mismatch (-want +got):\n%s", diff)
			}

			updatedTime := time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)
			ctxtimetest.SetFixedNow(t, ctx, updatedTime)

			obj.MarkAsUploaded(ctx)

			if diff := cmp.Diff(true, obj.IsUploaded()); diff != "" {
				t.Errorf("MarkAsUploaded()後にIsUploaded() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(updatedTime, obj.UpdatedAt()); diff != "" {
				t.Errorf("UpdatedAt() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLFSObject_Getters(t *testing.T) {
	type args struct {
		oidValue      string
		sizeValue     int64
		hashAlgoValue string
		storageKey    string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常系: 各Getterメソッドの正常動作",
			args: args{
				oidValue:      "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				sizeValue:     2048,
				hashAlgoValue: "sha256",
				storageKey:    "storage/path/key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, err := domain.NewOID(tt.args.oidValue)
			if err != nil {
				t.Fatalf("NewOID() failed: %v", err)
			}

			size, err := domain.NewSize(tt.args.sizeValue)
			if err != nil {
				t.Fatalf("NewSize() failed: %v", err)
			}

			hashAlgo, err := domain.NewHashAlgorithm(tt.args.hashAlgoValue)
			if err != nil {
				t.Fatalf("NewHashAlgorithm() failed: %v", err)
			}

			fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
			tid := uuid.NewString()
			ctx := testid.WithValue(context.Background(), tid)
			ctxtimetest.SetFixedNow(t, ctx, fixedTime)

			obj, err := domain.NewLFSObject(ctx, oid, size, hashAlgo, tt.args.storageKey)
			if err != nil {
				t.Fatalf("NewLFSObject() failed: %v", err)
			}

			if diff := cmp.Diff(tt.args.oidValue, obj.ID().String()); diff != "" {
				t.Errorf("ID().String() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.oidValue, obj.OID().String()); diff != "" {
				t.Errorf("OID().String() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.sizeValue, obj.Size().Int64()); diff != "" {
				t.Errorf("Size().Int64() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.hashAlgoValue, obj.HashAlgo()); diff != "" {
				t.Errorf("HashAlgo() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.storageKey, obj.GetStorageKey()); diff != "" {
				t.Errorf("GetStorageKey() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLFSObject_MultipleMarkAsUploaded(t *testing.T) {
	type args struct {
		oidString     string
		sizeValue     int64
		hashAlgoValue string
		storageKey    string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常系: 複数回のMarkAsUploadedの呼び出し",
			args: args{
				oidString:     "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				sizeValue:     512,
				hashAlgoValue: "sha256",
				storageKey:    "key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, err := domain.NewOID(tt.args.oidString)
			if err != nil {
				t.Fatalf("NewOID() failed: %v", err)
			}

			size, err := domain.NewSize(tt.args.sizeValue)
			if err != nil {
				t.Fatalf("NewSize() failed: %v", err)
			}

			hashAlgo, err := domain.NewHashAlgorithm(tt.args.hashAlgoValue)
			if err != nil {
				t.Fatalf("NewHashAlgorithm() failed: %v", err)
			}

			initialTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
			tid := uuid.NewString()
			ctx := testid.WithValue(context.Background(), tid)
			ctxtimetest.SetFixedNow(t, ctx, initialTime)

			obj, err := domain.NewLFSObject(ctx, oid, size, hashAlgo, tt.args.storageKey)
			if err != nil {
				t.Fatalf("NewLFSObject() failed: %v", err)
			}

			firstUpdateTime := time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)
			ctxtimetest.SetFixedNow(t, ctx, firstUpdateTime)
			obj.MarkAsUploaded(ctx)
			firstUpdate := obj.UpdatedAt()

			secondUpdateTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
			ctxtimetest.SetFixedNow(t, ctx, secondUpdateTime)
			obj.MarkAsUploaded(ctx)
			secondUpdate := obj.UpdatedAt()

			if diff := cmp.Diff(true, obj.IsUploaded()); diff != "" {
				t.Errorf("複数回のMarkAsUploaded()後にIsUploaded() mismatch (-want +got):\n%s", diff)
			}

			if !secondUpdate.After(firstUpdate) {
				t.Errorf("2回目のMarkAsUploaded()でUpdatedAtが更新されていない: first=%v, second=%v", firstUpdate, secondUpdate)
			}
		})
	}
}

func BenchmarkNewLFSObject(b *testing.B) {
	oid, _ := domain.NewOID("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	size, _ := domain.NewSize(1024)
	hashAlgo, _ := domain.NewHashAlgorithm("sha256")
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = domain.NewLFSObject(ctx, oid, size, hashAlgo, "test/key")
	}
}

func BenchmarkLFSObject_MarkAsUploaded(b *testing.B) {
	oid, _ := domain.NewOID("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	size, _ := domain.NewSize(1024)
	hashAlgo, _ := domain.NewHashAlgorithm("sha256")
	ctx := context.Background()
	obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "test/key")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obj.MarkAsUploaded(ctx)
	}
}

func TestReconstructLFSObject(t *testing.T) {
	type args struct {
		oidString     string
		sizeValue     int64
		hashAlgoValue string
		storageKey    string
		uploaded      bool
		createdAt     time.Time
		updatedAt     time.Time
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "正常系: uploaded=falseでの復元",
			args: args{
				oidString:     "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				sizeValue:     1024,
				hashAlgoValue: "sha256",
				storageKey:    "test/storage/key",
				uploaded:      false,
				createdAt:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				updatedAt:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			},
			wantErr: nil,
		},
		{
			name: "正常系: uploaded=trueでの復元",
			args: args{
				oidString:     "b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3",
				sizeValue:     2048,
				hashAlgoValue: "sha256",
				storageKey:    "test/storage/uploaded",
				uploaded:      true,
				createdAt:     time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC),
				updatedAt:     time.Date(2024, 1, 12, 14, 0, 0, 0, time.UTC),
			},
			wantErr: nil,
		},
		{
			name: "異常系: 空のStorageKeyでエラーが返る",
			args: args{
				oidString:     "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				sizeValue:     1024,
				hashAlgoValue: "sha256",
				storageKey:    "",
				uploaded:      false,
				createdAt:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				updatedAt:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			},
			wantErr: domain.ErrInvalidStorageKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, err := domain.NewOID(tt.args.oidString)
			if err != nil {
				t.Fatalf("NewOID() failed: %v", err)
			}

			size, err := domain.NewSize(tt.args.sizeValue)
			if err != nil {
				t.Fatalf("NewSize() failed: %v", err)
			}

			obj, err := domain.ReconstructLFSObject(
				oid,
				size,
				tt.args.hashAlgoValue,
				tt.args.storageKey,
				tt.args.uploaded,
				tt.args.createdAt,
				tt.args.updatedAt,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want error %v, but got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("ReconstructLFSObject() failed: %v", err)
			}

			if obj == nil {
				t.Fatal("ReconstructLFSObject() returned nil")
			}

			if diff := cmp.Diff(oid.String(), obj.OID().String()); diff != "" {
				t.Errorf("OID() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(size.Int64(), obj.Size().Int64()); diff != "" {
				t.Errorf("Size() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.hashAlgoValue, obj.HashAlgo()); diff != "" {
				t.Errorf("HashAlgo() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.storageKey, obj.GetStorageKey()); diff != "" {
				t.Errorf("GetStorageKey() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.uploaded, obj.IsUploaded()); diff != "" {
				t.Errorf("IsUploaded() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.createdAt, obj.CreatedAt()); diff != "" {
				t.Errorf("CreatedAt() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.updatedAt, obj.UpdatedAt()); diff != "" {
				t.Errorf("UpdatedAt() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconstructLFSObject_InvalidHashAlgorithm(t *testing.T) {
	tests := []struct {
		name          string
		hashAlgoValue string
		wantErr       error
	}{
		{
			name:          "異常系: 無効なハッシュアルゴリズム",
			hashAlgoValue: "invalid",
			wantErr:       domain.ErrInvalidHashAlgorithm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, err := domain.NewOID("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
			if err != nil {
				t.Fatalf("NewOID() failed: %v", err)
			}

			size, err := domain.NewSize(1024)
			if err != nil {
				t.Fatalf("NewSize() failed: %v", err)
			}

			_, err = domain.ReconstructLFSObject(
				oid,
				size,
				tt.hashAlgoValue,
				"test/key",
				false,
				time.Now(),
				time.Now(),
			)

			if err == nil {
				t.Fatal("ReconstructLFSObject() should return error for invalid hash algorithm")
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ReconstructLFSObject() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconstructLFSObject_PreservesTimestamps(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "正常系: ReconstructLFSObjectはタイムスタンプを変更しない",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, err := domain.NewOID("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
			if err != nil {
				t.Fatalf("NewOID() failed: %v", err)
			}

			size, err := domain.NewSize(1024)
			if err != nil {
				t.Fatalf("NewSize() failed: %v", err)
			}

			pastCreatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
			pastUpdatedAt := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC)

			obj, err := domain.ReconstructLFSObject(
				oid,
				size,
				"sha256",
				"test/key",
				true,
				pastCreatedAt,
				pastUpdatedAt,
			)

			if err != nil {
				t.Fatalf("ReconstructLFSObject() failed: %v", err)
			}

			if !obj.CreatedAt().Equal(pastCreatedAt) {
				t.Errorf("CreatedAt()が変更されています: want=%v, got=%v", pastCreatedAt, obj.CreatedAt())
			}

			if !obj.UpdatedAt().Equal(pastUpdatedAt) {
				t.Errorf("UpdatedAt()が変更されています: want=%v, got=%v", pastUpdatedAt, obj.UpdatedAt())
			}
		})
	}
}
