package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/mock/gomock"

	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_domain "github.com/na2na-p/cargohold/tests/domain"
	mock_usecase "github.com/na2na-p/cargohold/tests/usecase"
)

func TestProxyUploadUseCase_Execute(t *testing.T) {
	type fields struct {
		repo          func(ctrl *gomock.Controller) domain.LFSObjectRepository
		objectStorage func(ctrl *gomock.Controller) usecase.ObjectStorage
		authService   func(ctrl *gomock.Controller) domain.AccessAuthorizationService
	}
	type args struct {
		ctx   context.Context
		owner string
		repo  string
		oid   domain.OID
		body  io.Reader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "正常系: LFSObjectが見つかりアップロードが成功する",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationUpload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{Allowed: true}, nil)
					return mock
				},
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					ctx := context.Background()
					oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
					size, _ := domain.NewSize(1024)
					hashAlgo, _ := domain.NewHashAlgorithm("sha256")
					obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234")
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(obj, nil)
					mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				objectStorage: func(ctrl *gomock.Controller) usecase.ObjectStorage {
					mock := mock_usecase.NewMockObjectStorage(ctrl)
					mock.EXPECT().PutObject(gomock.Any(), "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234", gomock.Any()).Return(nil)
					return mock
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx:   context.Background(),
					owner: "testowner",
					repo:  "testrepo",
					oid:   oid,
					body:  bytes.NewReader([]byte("test data")),
				}
			}(),
			wantErr: nil,
		},
		{
			name: "異常系: 認可が拒否された場合、ErrAccessDeniedが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationUpload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{Allowed: false}, nil)
					return mock
				},
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				objectStorage: func(ctrl *gomock.Controller) usecase.ObjectStorage {
					return mock_usecase.NewMockObjectStorage(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx:   context.Background(),
					owner: "testowner",
					repo:  "testrepo",
					oid:   oid,
					body:  bytes.NewReader([]byte("test data")),
				}
			}(),
			wantErr: usecase.ErrAccessDenied,
		},
		{
			name: "異常系: 認可サービスがエラーを返した場合、ErrAccessDeniedが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationUpload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{}, domain.ErrAuthorizationDenied)
					return mock
				},
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				objectStorage: func(ctrl *gomock.Controller) usecase.ObjectStorage {
					return mock_usecase.NewMockObjectStorage(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx:   context.Background(),
					owner: "testowner",
					repo:  "testrepo",
					oid:   oid,
					body:  bytes.NewReader([]byte("test data")),
				}
			}(),
			wantErr: usecase.ErrAccessDenied,
		},
		{
			name: "異常系: LFSObjectが見つからない場合、ErrObjectNotFoundが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationUpload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{Allowed: true}, nil)
					return mock
				},
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
					return mock
				},
				objectStorage: func(ctrl *gomock.Controller) usecase.ObjectStorage {
					return mock_usecase.NewMockObjectStorage(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx:   context.Background(),
					owner: "testowner",
					repo:  "testrepo",
					oid:   oid,
					body:  bytes.NewReader([]byte("test data")),
				}
			}(),
			wantErr: usecase.ErrObjectNotFound,
		},
		{
			name: "異常系: リポジトリ検索でエラーが発生した場合、そのエラーが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationUpload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{Allowed: true}, nil)
					return mock
				},
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, errors.New("database error"))
					return mock
				},
				objectStorage: func(ctrl *gomock.Controller) usecase.ObjectStorage {
					return mock_usecase.NewMockObjectStorage(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx:   context.Background(),
					owner: "testowner",
					repo:  "testrepo",
					oid:   oid,
					body:  bytes.NewReader([]byte("test data")),
				}
			}(),
			wantErr: errors.New("database error"),
		},
		{
			name: "異常系: ObjectStorage.PutObjectでエラーが発生した場合、そのエラーが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationUpload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{Allowed: true}, nil)
					return mock
				},
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
				objectStorage: func(ctrl *gomock.Controller) usecase.ObjectStorage {
					mock := mock_usecase.NewMockObjectStorage(ctrl)
					mock.EXPECT().PutObject(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("storage error"))
					return mock
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx:   context.Background(),
					owner: "testowner",
					repo:  "testrepo",
					oid:   oid,
					body:  bytes.NewReader([]byte("test data")),
				}
			}(),
			wantErr: errors.New("storage error"),
		},
		{
			name: "異常系: リポジトリ更新でエラーが発生した場合、そのエラーが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationUpload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{Allowed: true}, nil)
					return mock
				},
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					ctx := context.Background()
					oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
					size, _ := domain.NewSize(1024)
					hashAlgo, _ := domain.NewHashAlgorithm("sha256")
					obj, _ := domain.NewLFSObject(ctx, oid, size, hashAlgo, "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234")
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(obj, nil)
					mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("update error"))
					return mock
				},
				objectStorage: func(ctrl *gomock.Controller) usecase.ObjectStorage {
					mock := mock_usecase.NewMockObjectStorage(ctrl)
					mock.EXPECT().PutObject(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx:   context.Background(),
					owner: "testowner",
					repo:  "testrepo",
					oid:   oid,
					body:  bytes.NewReader([]byte("test data")),
				}
			}(),
			wantErr: errors.New("update error"),
		},
		{
			name: "異常系: 不正なリポジトリ識別子の場合、ErrAccessDeniedが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					return mock_domain.NewMockAccessAuthorizationService(ctrl)
				},
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				objectStorage: func(ctrl *gomock.Controller) usecase.ObjectStorage {
					return mock_usecase.NewMockObjectStorage(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				return args{
					ctx:   context.Background(),
					owner: "",
					repo:  "",
					oid:   oid,
					body:  bytes.NewReader([]byte("test data")),
				}
			}(),
			wantErr: usecase.ErrAccessDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			uc := usecase.NewProxyUploadUseCase(
				tt.fields.repo(ctrl),
				tt.fields.objectStorage(ctrl),
				tt.fields.authService(ctrl),
			)

			err := uc.Execute(tt.args.ctx, tt.args.owner, tt.args.repo, tt.args.oid, tt.args.body)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if errors.Is(tt.wantErr, usecase.ErrAccessDenied) || errors.Is(tt.wantErr, usecase.ErrObjectNotFound) {
					if !errors.Is(err, tt.wantErr) {
						t.Errorf("want error %v, but got %v", tt.wantErr, err)
					}
				} else {
					if diff := cmp.Diff(tt.wantErr.Error(), err.Error()); diff != "" {
						t.Errorf("error mismatch (-want +got):\n%s", diff)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}
		})
	}
}
