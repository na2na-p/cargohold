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

func TestUploadUseCase_HandleUploadObject(t *testing.T) {
	type fields struct {
		repo                func(ctrl *gomock.Controller) domain.LFSObjectRepository
		actionURLGenerator  func(ctrl *gomock.Controller) usecase.ActionURLGenerator
		storageKeyGenerator func(ctrl *gomock.Controller) usecase.StorageKeyGenerator
	}
	type args struct {
		ctx        context.Context
		baseURL    string
		owner      string
		repo       string
		oid        domain.OID
		size       domain.Size
		hashAlgo   string
		authHeader string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   usecase.ResponseObject
	}{
		{
			name: "正常系: オブジェクトがアップロード済みの場合、アクションなしで返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					ctx := context.Background()
					oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
					size, _ := domain.NewSize(1024)
					hashAlgo, _ := domain.NewHashAlgorithm("sha256")
					obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234")
					obj.MarkAsUploaded(ctx)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(obj, nil)
					return mock
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				return args{
					ctx:      context.Background(),
					baseURL:  "https://example.com",
					owner:    "test-owner",
					repo:     "test-repo",
					oid:      oid,
					size:     size,
					hashAlgo: "sha256",
				}
			}(),
			want: usecase.NewResponseObject("1234567890123456789012345678901234567890123456789012345678901234", 1024, true, nil, nil),
		},
		{
			name: "正常系: オブジェクトが未登録の場合、新規登録してアップロードURLが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
					mock.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					mock := mock_usecase.NewMockActionURLGenerator(ctrl)
					mock.EXPECT().GenerateUploadURL("https://example.com", "test-owner", "test-repo", "1234567890123456789012345678901234567890123456789012345678901234").Return("https://example.com/test-owner/test-repo/objects/1234567890123456789012345678901234567890123456789012345678901234/upload")
					return mock
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					mock := mock_usecase.NewMockStorageKeyGenerator(ctrl)
					mock.EXPECT().GenerateStorageKey(gomock.Any(), gomock.Any()).Return("objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234", nil)
					return mock
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				return args{
					ctx:      context.Background(),
					baseURL:  "https://example.com",
					owner:    "test-owner",
					repo:     "test-repo",
					oid:      oid,
					size:     size,
					hashAlgo: "sha256",
				}
			}(),
			want: func() usecase.ResponseObject {
				uploadAction := usecase.NewAction("https://example.com/test-owner/test-repo/objects/1234567890123456789012345678901234567890123456789012345678901234/upload", nil, 900)
				actions := usecase.NewActions(&uploadAction, nil)
				return usecase.NewResponseObject("1234567890123456789012345678901234567890123456789012345678901234", 1024, true, &actions, nil)
			}(),
		},
		{
			name: "正常系: オブジェクトが登録済みだがアップロード未完了の場合、アップロードURLが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					ctx := context.Background()
					oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
					size, _ := domain.NewSize(1024)
					hashAlgo, _ := domain.NewHashAlgorithm("sha256")
					obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "existing-storage-key-from-db")
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(obj, nil)
					return mock
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					mock := mock_usecase.NewMockActionURLGenerator(ctrl)
					mock.EXPECT().GenerateUploadURL("https://example.com", "test-owner", "test-repo", "1234567890123456789012345678901234567890123456789012345678901234").Return("https://example.com/test-owner/test-repo/objects/1234567890123456789012345678901234567890123456789012345678901234/upload")
					return mock
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				return args{
					ctx:      context.Background(),
					baseURL:  "https://example.com",
					owner:    "test-owner",
					repo:     "test-repo",
					oid:      oid,
					size:     size,
					hashAlgo: "sha256",
				}
			}(),
			want: func() usecase.ResponseObject {
				uploadAction := usecase.NewAction("https://example.com/test-owner/test-repo/objects/1234567890123456789012345678901234567890123456789012345678901234/upload", nil, 900)
				actions := usecase.NewActions(&uploadAction, nil)
				return usecase.NewResponseObject("1234567890123456789012345678901234567890123456789012345678901234", 1024, true, &actions, nil)
			}(),
		},
		{
			name: "異常系: メタデータ保存に失敗した場合、500エラーが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
					mock.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("save error"))
					return mock
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					mock := mock_usecase.NewMockStorageKeyGenerator(ctrl)
					mock.EXPECT().GenerateStorageKey(gomock.Any(), gomock.Any()).Return("objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234", nil)
					return mock
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				return args{
					ctx:      context.Background(),
					baseURL:  "https://example.com",
					owner:    "test-owner",
					repo:     "test-repo",
					oid:      oid,
					size:     size,
					hashAlgo: "sha256",
				}
			}(),
			want: func() usecase.ResponseObject {
				objErr := usecase.NewObjectError(500, "メタデータの保存に失敗しました")
				return usecase.NewResponseObject("1234567890123456789012345678901234567890123456789012345678901234", 1024, false, nil, &objErr)
			}(),
		},
		{
			name: "異常系: 無効なハッシュアルゴリズムの場合、400エラーが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
					return mock
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					mock := mock_usecase.NewMockStorageKeyGenerator(ctrl)
					mock.EXPECT().GenerateStorageKey(gomock.Any(), gomock.Any()).Return("objects/invalid_algo/12/34/1234567890123456789012345678901234567890123456789012345678901234", nil)
					return mock
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				return args{
					ctx:      context.Background(),
					baseURL:  "https://example.com",
					owner:    "test-owner",
					repo:     "test-repo",
					oid:      oid,
					size:     size,
					hashAlgo: "invalid_algo",
				}
			}(),
			want: func() usecase.ResponseObject {
				objErr := usecase.NewObjectError(400, "無効なハッシュアルゴリズムです")
				return usecase.NewResponseObject("1234567890123456789012345678901234567890123456789012345678901234", 1024, false, nil, &objErr)
			}(),
		},
		{
			name: "異常系: FindByOIDでNotFound以外のエラーが発生した場合、500エラーが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, errors.New("database connection error"))
					return mock
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				return args{
					ctx:      context.Background(),
					baseURL:  "https://example.com",
					owner:    "test-owner",
					repo:     "test-repo",
					oid:      oid,
					size:     size,
					hashAlgo: "sha256",
				}
			}(),
			want: func() usecase.ResponseObject {
				objErr := usecase.NewObjectError(500, "メタデータの取得に失敗しました")
				return usecase.NewResponseObject("1234567890123456789012345678901234567890123456789012345678901234", 1024, false, nil, &objErr)
			}(),
		},
		{
			name: "異常系: アップロード済みオブジェクトとサイズが一致しない場合、409エラーが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					ctx := context.Background()
					oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
					storedSize, _ := domain.NewSize(2048)
					hashAlgo, _ := domain.NewHashAlgorithm("sha256")
					obj, _ := domain.NewLFSObject(ctx, oid, storedSize, hashAlgo, "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234")
					obj.MarkAsUploaded(ctx)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(obj, nil)
					return mock
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				return args{
					ctx:      context.Background(),
					baseURL:  "https://example.com",
					owner:    "test-owner",
					repo:     "test-repo",
					oid:      oid,
					size:     size,
					hashAlgo: "sha256",
				}
			}(),
			want: func() usecase.ResponseObject {
				objErr := usecase.NewObjectError(409, "オブジェクトサイズが一致しません")
				return usecase.NewResponseObject("1234567890123456789012345678901234567890123456789012345678901234", 1024, false, nil, &objErr)
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			uc := usecase.NewUploadUseCase(
				tt.fields.repo(ctrl),
				tt.fields.actionURLGenerator(ctrl),
				tt.fields.storageKeyGenerator(ctrl),
			)

			got := uc.HandleUploadObject(tt.args.ctx, tt.args.baseURL, tt.args.owner, tt.args.repo, tt.args.oid, tt.args.size, tt.args.hashAlgo, tt.args.authHeader)

			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(
				usecase.ResponseObject{},
				usecase.Actions{},
				usecase.Action{},
				usecase.ObjectError{},
			)); diff != "" {
				t.Errorf("HandleUploadObject() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
