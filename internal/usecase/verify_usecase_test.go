package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_domain "github.com/na2na-p/cargohold/tests/domain"
	mock_usecase "github.com/na2na-p/cargohold/tests/usecase"
	"go.uber.org/mock/gomock"
)

func TestVerifyUseCase_VerifyUpload(t *testing.T) {
	type fields struct {
		repo            func(ctrl *gomock.Controller) domain.LFSObjectRepository
		cacheKeyManager func(ctrl *gomock.Controller) usecase.CacheKeyManager
	}
	type args struct {
		ctx  context.Context
		oid  string
		size int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "正常系: アップロード検証が成功する",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					ctx := context.Background()
					oid, _ := domain.NewOID("a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd")
					size, _ := domain.NewSize(1024)
					hashAlgo, _ := domain.NewHashAlgorithm("sha256")
					obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "test-storage-key")

					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(obj, nil)
					mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				cacheKeyManager: func(ctrl *gomock.Controller) usecase.CacheKeyManager {
					mock := mock_usecase.NewMockCacheKeyManager(ctrl)
					mock.EXPECT().DeleteBatchUploadKey(gomock.Any(), "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd").Return(nil)
					return mock
				},
			},
			args: args{
				ctx:  context.Background(),
				oid:  "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
				size: 1024,
			},
			wantErr: nil,
		},
		{
			name: "異常系: OIDが無効な形式の場合、ErrInvalidOIDが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				cacheKeyManager: func(ctrl *gomock.Controller) usecase.CacheKeyManager {
					return mock_usecase.NewMockCacheKeyManager(ctrl)
				},
			},
			args: args{
				ctx:  context.Background(),
				oid:  "invalid-oid",
				size: 1024,
			},
			wantErr: usecase.ErrInvalidOID,
		},
		{
			name: "異常系: サイズが負の値の場合、ErrInvalidSizeが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				cacheKeyManager: func(ctrl *gomock.Controller) usecase.CacheKeyManager {
					return mock_usecase.NewMockCacheKeyManager(ctrl)
				},
			},
			args: args{
				ctx:  context.Background(),
				oid:  "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
				size: -1,
			},
			wantErr: usecase.ErrInvalidSize,
		},
		{
			name: "異常系: オブジェクトが存在しない場合、ErrObjectNotFoundが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
					return mock
				},
				cacheKeyManager: func(ctrl *gomock.Controller) usecase.CacheKeyManager {
					return mock_usecase.NewMockCacheKeyManager(ctrl)
				},
			},
			args: args{
				ctx:  context.Background(),
				oid:  "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
				size: 1024,
			},
			wantErr: usecase.ErrObjectNotFound,
		},
		{
			name: "異常系: サイズが一致しない場合、ErrSizeMismatchが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					ctx := context.Background()
					oid, _ := domain.NewOID("a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd")
					size, _ := domain.NewSize(1024)
					hashAlgo, _ := domain.NewHashAlgorithm("sha256")
					obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "test-storage-key")

					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(obj, nil)
					return mock
				},
				cacheKeyManager: func(ctrl *gomock.Controller) usecase.CacheKeyManager {
					return mock_usecase.NewMockCacheKeyManager(ctrl)
				},
			},
			args: args{
				ctx:  context.Background(),
				oid:  "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
				size: 2048,
			},
			wantErr: usecase.ErrSizeMismatch,
		},
		{
			name: "異常系: PostgreSQLの更新に失敗した場合、エラーが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					ctx := context.Background()
					oid, _ := domain.NewOID("a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd")
					size, _ := domain.NewSize(1024)
					hashAlgo, _ := domain.NewHashAlgorithm("sha256")
					obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "test-storage-key")

					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(obj, nil)
					mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("update failed"))
					return mock
				},
				cacheKeyManager: func(ctrl *gomock.Controller) usecase.CacheKeyManager {
					return mock_usecase.NewMockCacheKeyManager(ctrl)
				},
			},
			args: args{
				ctx:  context.Background(),
				oid:  "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
				size: 1024,
			},
			wantErr: errors.New("メタデータの更新に失敗しました: update failed"),
		},
		{
			name: "正常系: DeleteBatchUploadKeyに失敗しても処理は継続する",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					ctx := context.Background()
					oid, _ := domain.NewOID("a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd")
					size, _ := domain.NewSize(1024)
					hashAlgo, _ := domain.NewHashAlgorithm("sha256")
					obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "test-storage-key")

					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(obj, nil)
					mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				cacheKeyManager: func(ctrl *gomock.Controller) usecase.CacheKeyManager {
					mock := mock_usecase.NewMockCacheKeyManager(ctrl)
					mock.EXPECT().DeleteBatchUploadKey(gomock.Any(), gomock.Any()).Return(errors.New("delete error"))
					return mock
				},
			},
			args: args{
				ctx:  context.Background(),
				oid:  "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
				size: 1024,
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := tt.fields.repo(ctrl)
			cacheKeyManager := tt.fields.cacheKeyManager(ctrl)

			uc := usecase.NewVerifyUseCase(repo, cacheKeyManager)

			err := uc.VerifyUpload(tt.args.ctx, tt.args.oid, tt.args.size)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("VerifyUpload() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					if diff := cmp.Diff(tt.wantErr.Error(), err.Error()); diff != "" {
						t.Errorf("VerifyUpload() error mismatch (-want +got):\n%s", diff)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("VerifyUpload() unexpected error: %v", err)
				}
			}
		})
	}
}
