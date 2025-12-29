package infrastructure_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/infrastructure"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_domain "github.com/na2na-p/cargohold/tests/domain"
	mock_usecase "github.com/na2na-p/cargohold/tests/usecase"
	"go.uber.org/mock/gomock"
)

func TestCachingLFSObjectRepository_FindByOID(t *testing.T) {
	type fields struct {
		repo         func(ctrl *gomock.Controller) domain.LFSObjectRepository
		cacheClient  func(ctrl *gomock.Controller) usecase.CacheClient
		keyGenerator func(ctrl *gomock.Controller) usecase.CacheKeyGenerator
		cacheConfig  func(ctrl *gomock.Controller) usecase.CacheConfig
	}
	type args struct {
		ctx context.Context
		oid domain.OID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantOID string
		wantErr error
	}{
		{
			name: "正常系: キャッシュヒット時はキャッシュからメタデータが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				cacheClient: func(ctrl *gomock.Controller) usecase.CacheClient {
					mock := mock_usecase.NewMockCacheClient(ctrl)
					mock.EXPECT().GetJSON(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, key string, dest interface{}) error {
							cached := map[string]interface{}{
								"oid":         "1234567890123456789012345678901234567890123456789012345678901234",
								"size":        int64(1024),
								"hash_algo":   "sha256",
								"storage_key": "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234",
								"uploaded":    false,
								"created_at":  time.Now().Format(time.RFC3339Nano),
								"updated_at":  time.Now().Format(time.RFC3339Nano),
							}
							jsonData, _ := json.Marshal(cached)
							return json.Unmarshal(jsonData, dest)
						},
					)
					return mock
				},
				keyGenerator: func(ctrl *gomock.Controller) usecase.CacheKeyGenerator {
					mock := mock_usecase.NewMockCacheKeyGenerator(ctrl)
					mock.EXPECT().MetadataKey(gomock.Any()).Return("metadata:1234567890123456789012345678901234567890123456789012345678901234")
					return mock
				},
				cacheConfig: func(ctrl *gomock.Controller) usecase.CacheConfig {
					return mock_usecase.NewMockCacheConfig(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx: context.Background(),
					oid: oid,
				}
			}(),
			wantOID: "1234567890123456789012345678901234567890123456789012345678901234",
			wantErr: nil,
		},
		{
			name: "正常系: キャッシュミス時はリポジトリからメタデータが返り、キャッシュに保存される",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					ctx := context.Background()
					oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
					size, _ := domain.NewSize(1024)
					hashAlgo, _ := domain.NewHashAlgorithm("sha256")
					obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234")
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(obj, nil)
					return mock
				},
				cacheClient: func(ctrl *gomock.Controller) usecase.CacheClient {
					mock := mock_usecase.NewMockCacheClient(ctrl)
					mock.EXPECT().GetJSON(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss"))
					mock.EXPECT().SetJSON(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				keyGenerator: func(ctrl *gomock.Controller) usecase.CacheKeyGenerator {
					mock := mock_usecase.NewMockCacheKeyGenerator(ctrl)
					mock.EXPECT().MetadataKey(gomock.Any()).Return("metadata:1234567890123456789012345678901234567890123456789012345678901234").Times(2)
					return mock
				},
				cacheConfig: func(ctrl *gomock.Controller) usecase.CacheConfig {
					mock := mock_usecase.NewMockCacheConfig(ctrl)
					mock.EXPECT().MetadataTTL().Return(time.Hour)
					return mock
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx: context.Background(),
					oid: oid,
				}
			}(),
			wantOID: "1234567890123456789012345678901234567890123456789012345678901234",
			wantErr: nil,
		},
		{
			name: "異常系: キャッシュミスかつリポジトリでも見つからない場合はエラーが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, errors.New("not found"))
					return mock
				},
				cacheClient: func(ctrl *gomock.Controller) usecase.CacheClient {
					mock := mock_usecase.NewMockCacheClient(ctrl)
					mock.EXPECT().GetJSON(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss"))
					return mock
				},
				keyGenerator: func(ctrl *gomock.Controller) usecase.CacheKeyGenerator {
					mock := mock_usecase.NewMockCacheKeyGenerator(ctrl)
					mock.EXPECT().MetadataKey(gomock.Any()).Return("metadata:1234567890123456789012345678901234567890123456789012345678901234")
					return mock
				},
				cacheConfig: func(ctrl *gomock.Controller) usecase.CacheConfig {
					return mock_usecase.NewMockCacheConfig(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx: context.Background(),
					oid: oid,
				}
			}(),
			wantOID: "",
			wantErr: errors.New("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			repo := infrastructure.NewCachingLFSObjectRepository(
				tt.fields.repo(ctrl),
				tt.fields.cacheClient(ctrl),
				tt.fields.keyGenerator(ctrl),
				tt.fields.cacheConfig(ctrl),
			)

			got, err := repo.FindByOID(tt.args.ctx, tt.args.oid)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("FindByOID() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("FindByOID() unexpected error: %v", err)
			}

			if tt.wantOID != "" && got != nil {
				if diff := cmp.Diff(tt.wantOID, got.OID().String()); diff != "" {
					t.Errorf("FindByOID() OID mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCachingLFSObjectRepository_Save(t *testing.T) {
	type fields struct {
		repo         func(ctrl *gomock.Controller) domain.LFSObjectRepository
		cacheClient  func(ctrl *gomock.Controller) usecase.CacheClient
		keyGenerator func(ctrl *gomock.Controller) usecase.CacheKeyGenerator
		cacheConfig  func(ctrl *gomock.Controller) usecase.CacheConfig
	}
	type args struct {
		ctx context.Context
		obj *domain.LFSObject
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "正常系: 保存後にキャッシュに書き込まれる",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				cacheClient: func(ctrl *gomock.Controller) usecase.CacheClient {
					mock := mock_usecase.NewMockCacheClient(ctrl)
					mock.EXPECT().SetJSON(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				keyGenerator: func(ctrl *gomock.Controller) usecase.CacheKeyGenerator {
					mock := mock_usecase.NewMockCacheKeyGenerator(ctrl)
					mock.EXPECT().MetadataKey(gomock.Any()).Return("metadata:1234567890123456789012345678901234567890123456789012345678901234")
					return mock
				},
				cacheConfig: func(ctrl *gomock.Controller) usecase.CacheConfig {
					mock := mock_usecase.NewMockCacheConfig(ctrl)
					mock.EXPECT().MetadataTTL().Return(time.Hour)
					return mock
				},
			},
			args: func() args {
				ctx := context.Background()
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				hashAlgo, _ := domain.NewHashAlgorithm("sha256")
				obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx: ctx,
					obj: obj,
				}
			}(),
			wantErr: nil,
		},
		{
			name: "異常系: リポジトリ保存失敗時はエラーが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("save failed"))
					return mock
				},
				cacheClient: func(ctrl *gomock.Controller) usecase.CacheClient {
					return mock_usecase.NewMockCacheClient(ctrl)
				},
				keyGenerator: func(ctrl *gomock.Controller) usecase.CacheKeyGenerator {
					return mock_usecase.NewMockCacheKeyGenerator(ctrl)
				},
				cacheConfig: func(ctrl *gomock.Controller) usecase.CacheConfig {
					return mock_usecase.NewMockCacheConfig(ctrl)
				},
			},
			args: func() args {
				ctx := context.Background()
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				hashAlgo, _ := domain.NewHashAlgorithm("sha256")
				obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx: ctx,
					obj: obj,
				}
			}(),
			wantErr: errors.New("save failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			repo := infrastructure.NewCachingLFSObjectRepository(
				tt.fields.repo(ctrl),
				tt.fields.cacheClient(ctrl),
				tt.fields.keyGenerator(ctrl),
				tt.fields.cacheConfig(ctrl),
			)

			err := repo.Save(tt.args.ctx, tt.args.obj)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Save() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Save() unexpected error: %v", err)
			}
		})
	}
}

func TestCachingLFSObjectRepository_Update(t *testing.T) {
	type fields struct {
		repo         func(ctrl *gomock.Controller) domain.LFSObjectRepository
		cacheClient  func(ctrl *gomock.Controller) usecase.CacheClient
		keyGenerator func(ctrl *gomock.Controller) usecase.CacheKeyGenerator
		cacheConfig  func(ctrl *gomock.Controller) usecase.CacheConfig
	}
	type args struct {
		ctx context.Context
		obj *domain.LFSObject
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "正常系: 更新後にキャッシュが更新される",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				cacheClient: func(ctrl *gomock.Controller) usecase.CacheClient {
					mock := mock_usecase.NewMockCacheClient(ctrl)
					mock.EXPECT().SetJSON(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				keyGenerator: func(ctrl *gomock.Controller) usecase.CacheKeyGenerator {
					mock := mock_usecase.NewMockCacheKeyGenerator(ctrl)
					mock.EXPECT().MetadataKey(gomock.Any()).Return("metadata:1234567890123456789012345678901234567890123456789012345678901234")
					return mock
				},
				cacheConfig: func(ctrl *gomock.Controller) usecase.CacheConfig {
					mock := mock_usecase.NewMockCacheConfig(ctrl)
					mock.EXPECT().MetadataTTL().Return(time.Hour)
					return mock
				},
			},
			args: func() args {
				ctx := context.Background()
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				hashAlgo, _ := domain.NewHashAlgorithm("sha256")
				obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx: ctx,
					obj: obj,
				}
			}(),
			wantErr: nil,
		},
		{
			name: "異常系: リポジトリ更新失敗時はエラーが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("update failed"))
					return mock
				},
				cacheClient: func(ctrl *gomock.Controller) usecase.CacheClient {
					return mock_usecase.NewMockCacheClient(ctrl)
				},
				keyGenerator: func(ctrl *gomock.Controller) usecase.CacheKeyGenerator {
					return mock_usecase.NewMockCacheKeyGenerator(ctrl)
				},
				cacheConfig: func(ctrl *gomock.Controller) usecase.CacheConfig {
					return mock_usecase.NewMockCacheConfig(ctrl)
				},
			},
			args: func() args {
				ctx := context.Background()
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				hashAlgo, _ := domain.NewHashAlgorithm("sha256")
				obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx: ctx,
					obj: obj,
				}
			}(),
			wantErr: errors.New("update failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			repo := infrastructure.NewCachingLFSObjectRepository(
				tt.fields.repo(ctrl),
				tt.fields.cacheClient(ctrl),
				tt.fields.keyGenerator(ctrl),
				tt.fields.cacheConfig(ctrl),
			)

			err := repo.Update(tt.args.ctx, tt.args.obj)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Update() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Update() unexpected error: %v", err)
			}
		})
	}
}

func TestCachingLFSObjectRepository_ExistsByOID(t *testing.T) {
	type fields struct {
		repo         func(ctrl *gomock.Controller) domain.LFSObjectRepository
		cacheClient  func(ctrl *gomock.Controller) usecase.CacheClient
		keyGenerator func(ctrl *gomock.Controller) usecase.CacheKeyGenerator
		cacheConfig  func(ctrl *gomock.Controller) usecase.CacheConfig
	}
	type args struct {
		ctx context.Context
		oid domain.OID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr error
	}{
		{
			name: "正常系: キャッシュにある場合はtrueが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				cacheClient: func(ctrl *gomock.Controller) usecase.CacheClient {
					mock := mock_usecase.NewMockCacheClient(ctrl)
					mock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil)
					return mock
				},
				keyGenerator: func(ctrl *gomock.Controller) usecase.CacheKeyGenerator {
					mock := mock_usecase.NewMockCacheKeyGenerator(ctrl)
					mock.EXPECT().MetadataKey(gomock.Any()).Return("metadata:1234567890123456789012345678901234567890123456789012345678901234")
					return mock
				},
				cacheConfig: func(ctrl *gomock.Controller) usecase.CacheConfig {
					return mock_usecase.NewMockCacheConfig(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx: context.Background(),
					oid: oid,
				}
			}(),
			want:    true,
			wantErr: nil,
		},
		{
			name: "正常系: キャッシュに無い場合はDBを確認する",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().ExistsByOID(gomock.Any(), gomock.Any()).Return(true, nil)
					return mock
				},
				cacheClient: func(ctrl *gomock.Controller) usecase.CacheClient {
					mock := mock_usecase.NewMockCacheClient(ctrl)
					mock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
					return mock
				},
				keyGenerator: func(ctrl *gomock.Controller) usecase.CacheKeyGenerator {
					mock := mock_usecase.NewMockCacheKeyGenerator(ctrl)
					mock.EXPECT().MetadataKey(gomock.Any()).Return("metadata:1234567890123456789012345678901234567890123456789012345678901234")
					return mock
				},
				cacheConfig: func(ctrl *gomock.Controller) usecase.CacheConfig {
					return mock_usecase.NewMockCacheConfig(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx: context.Background(),
					oid: oid,
				}
			}(),
			want:    true,
			wantErr: nil,
		},
		{
			name: "正常系: キャッシュもDBにもない場合はfalseが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().ExistsByOID(gomock.Any(), gomock.Any()).Return(false, nil)
					return mock
				},
				cacheClient: func(ctrl *gomock.Controller) usecase.CacheClient {
					mock := mock_usecase.NewMockCacheClient(ctrl)
					mock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
					return mock
				},
				keyGenerator: func(ctrl *gomock.Controller) usecase.CacheKeyGenerator {
					mock := mock_usecase.NewMockCacheKeyGenerator(ctrl)
					mock.EXPECT().MetadataKey(gomock.Any()).Return("metadata:1234567890123456789012345678901234567890123456789012345678901234")
					return mock
				},
				cacheConfig: func(ctrl *gomock.Controller) usecase.CacheConfig {
					return mock_usecase.NewMockCacheConfig(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx: context.Background(),
					oid: oid,
				}
			}(),
			want:    false,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			repo := infrastructure.NewCachingLFSObjectRepository(
				tt.fields.repo(ctrl),
				tt.fields.cacheClient(ctrl),
				tt.fields.keyGenerator(ctrl),
				tt.fields.cacheConfig(ctrl),
			)

			got, err := repo.ExistsByOID(tt.args.ctx, tt.args.oid)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ExistsByOID() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ExistsByOID() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ExistsByOID() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCachingLFSObjectRepository_DeleteBatchUploadKey(t *testing.T) {
	type fields struct {
		repo         func(ctrl *gomock.Controller) domain.LFSObjectRepository
		cacheClient  func(ctrl *gomock.Controller) usecase.CacheClient
		keyGenerator func(ctrl *gomock.Controller) usecase.CacheKeyGenerator
		cacheConfig  func(ctrl *gomock.Controller) usecase.CacheConfig
	}
	type args struct {
		ctx context.Context
		oid string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "正常系: BatchUploadKeyが正常に削除される",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				cacheClient: func(ctrl *gomock.Controller) usecase.CacheClient {
					mock := mock_usecase.NewMockCacheClient(ctrl)
					mock.EXPECT().Delete(gomock.Any(), "batch:1234567890123456789012345678901234567890123456789012345678901234").Return(nil)
					return mock
				},
				keyGenerator: func(ctrl *gomock.Controller) usecase.CacheKeyGenerator {
					mock := mock_usecase.NewMockCacheKeyGenerator(ctrl)
					mock.EXPECT().BatchUploadKey("1234567890123456789012345678901234567890123456789012345678901234").Return("batch:1234567890123456789012345678901234567890123456789012345678901234")
					return mock
				},
				cacheConfig: func(ctrl *gomock.Controller) usecase.CacheConfig {
					return mock_usecase.NewMockCacheConfig(ctrl)
				},
			},
			args: args{
				ctx: context.Background(),
				oid: "1234567890123456789012345678901234567890123456789012345678901234",
			},
			wantErr: nil,
		},
		{
			name: "異常系: Redis削除エラー時はエラーが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				cacheClient: func(ctrl *gomock.Controller) usecase.CacheClient {
					mock := mock_usecase.NewMockCacheClient(ctrl)
					mock.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errors.New("redis delete error"))
					return mock
				},
				keyGenerator: func(ctrl *gomock.Controller) usecase.CacheKeyGenerator {
					mock := mock_usecase.NewMockCacheKeyGenerator(ctrl)
					mock.EXPECT().BatchUploadKey(gomock.Any()).Return("batch:test")
					return mock
				},
				cacheConfig: func(ctrl *gomock.Controller) usecase.CacheConfig {
					return mock_usecase.NewMockCacheConfig(ctrl)
				},
			},
			args: args{
				ctx: context.Background(),
				oid: "1234567890123456789012345678901234567890123456789012345678901234",
			},
			wantErr: errors.New("redis delete error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			repo := infrastructure.NewCachingLFSObjectRepository(
				tt.fields.repo(ctrl),
				tt.fields.cacheClient(ctrl),
				tt.fields.keyGenerator(ctrl),
				tt.fields.cacheConfig(ctrl),
			)

			err := repo.DeleteBatchUploadKey(tt.args.ctx, tt.args.oid)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("DeleteBatchUploadKey() error = nil, wantErr %v", tt.wantErr)
				}
				if diff := cmp.Diff(tt.wantErr.Error(), err.Error()); diff != "" {
					t.Errorf("DeleteBatchUploadKey() error mismatch (-want +got):\n%s", diff)
				}
				return
			}

			if err != nil {
				t.Fatalf("DeleteBatchUploadKey() unexpected error: %v", err)
			}
		})
	}
}
