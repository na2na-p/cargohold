package usecase_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_domain "github.com/na2na-p/cargohold/tests/domain"
	mock_usecase "github.com/na2na-p/cargohold/tests/usecase"
	"go.uber.org/mock/gomock"
)

func TestProxyDownloadUseCase_Execute(t *testing.T) {
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
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantBody string
		wantSize int64
		wantErr  error
	}{
		{
			name: "正常系: オブジェクトが存在しアップロード済みの場合、ストリームとサイズが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationDownload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{Allowed: true}, nil)
					return mock
				},
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
				objectStorage: func(ctrl *gomock.Controller) usecase.ObjectStorage {
					mock := mock_usecase.NewMockObjectStorage(ctrl)
					mock.EXPECT().GetObject(gomock.Any(), "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234").Return(io.NopCloser(strings.NewReader("test file content")), nil)
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
				}
			}(),
			wantBody: "test file content",
			wantSize: 1024,
			wantErr:  nil,
		},
		{
			name: "異常系: 認可が拒否された場合、ErrAccessDeniedが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationDownload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{Allowed: false}, nil)
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
				}
			}(),
			wantBody: "",
			wantSize: 0,
			wantErr:  usecase.ErrAccessDenied,
		},
		{
			name: "異常系: 認可サービスがエラーを返した場合、ErrAccessDeniedが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationDownload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{}, domain.ErrAuthorizationDenied)
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
				}
			}(),
			wantBody: "",
			wantSize: 0,
			wantErr:  usecase.ErrAccessDenied,
		},
		{
			name: "異常系: オブジェクトが存在しない場合、ErrObjectNotFoundエラーが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationDownload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{Allowed: true}, nil)
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
				}
			}(),
			wantBody: "",
			wantSize: 0,
			wantErr:  usecase.ErrObjectNotFound,
		},
		{
			name: "異常系: リポジトリでエラーが発生した場合、エラーが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationDownload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{Allowed: true}, nil)
					return mock
				},
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, errors.New("database connection error"))
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
				}
			}(),
			wantBody: "",
			wantSize: 0,
			wantErr:  errors.New("database connection error"),
		},
		{
			name: "異常系: オブジェクトがアップロード未完了の場合、ErrNotUploadedエラーが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationDownload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{Allowed: true}, nil)
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
				}
			}(),
			wantBody: "",
			wantSize: 0,
			wantErr:  usecase.ErrNotUploaded,
		},
		{
			name: "異常系: ObjectStorage.GetObjectでエラーが発生した場合、エラーが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationDownload, gomock.Any(), gomock.Any()).Return(domain.AuthorizationResult{Allowed: true}, nil)
					return mock
				},
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
				objectStorage: func(ctrl *gomock.Controller) usecase.ObjectStorage {
					mock := mock_usecase.NewMockObjectStorage(ctrl)
					mock.EXPECT().GetObject(gomock.Any(), "objects/sha256/12/34/1234567890123456789012345678901234567890123456789012345678901234").Return(nil, errors.New("S3 error"))
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
				}
			}(),
			wantBody: "",
			wantSize: 0,
			wantErr:  errors.New("S3 error"),
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
				}
			}(),
			wantBody: "",
			wantSize: 0,
			wantErr:  usecase.ErrAccessDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			uc := usecase.NewProxyDownloadUseCase(
				tt.fields.repo(ctrl),
				tt.fields.objectStorage(ctrl),
				tt.fields.authService(ctrl),
			)

			gotStream, gotSize, err := uc.Execute(tt.args.ctx, tt.args.owner, tt.args.repo, tt.args.oid)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if errors.Is(tt.wantErr, usecase.ErrObjectNotFound) || errors.Is(tt.wantErr, usecase.ErrNotUploaded) || errors.Is(tt.wantErr, usecase.ErrAccessDenied) {
					if !errors.Is(err, tt.wantErr) {
						t.Errorf("want error %v, but got %v", tt.wantErr, err)
					}
				} else {
					if err.Error() != tt.wantErr.Error() {
						t.Errorf("want error message %q, but got %q", tt.wantErr.Error(), err.Error())
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("want no error, but got %v", err)
			}

			if diff := cmp.Diff(tt.wantSize, gotSize); diff != "" {
				t.Errorf("Execute() size mismatch (-want +got):\n%s", diff)
			}

			if gotStream != nil {
				defer func() {
					_ = gotStream.Close()
				}()
				body, readErr := io.ReadAll(gotStream)
				if readErr != nil {
					t.Fatalf("failed to read stream: %v", readErr)
				}
				if diff := cmp.Diff(tt.wantBody, string(body)); diff != "" {
					t.Errorf("Execute() body mismatch (-want +got):\n%s", diff)
				}
			} else if tt.wantBody != "" {
				t.Error("Execute() got nil stream, want non-nil")
			}
		})
	}
}
