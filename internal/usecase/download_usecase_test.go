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

func TestDownloadUseCase_HandleDownloadObject(t *testing.T) {
	type fields struct {
		repo               func(ctrl *gomock.Controller) domain.LFSObjectRepository
		actionURLGenerator func(ctrl *gomock.Controller) usecase.ActionURLGenerator
	}
	type args struct {
		ctx        context.Context
		baseURL    string
		owner      string
		repo       string
		oid        domain.OID
		size       domain.Size
		authHeader string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   usecase.ResponseObject
	}{
		{
			name: "正常系: オブジェクトが存在しアップロード済みの場合、ダウンロードURLが返る",
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
					mock := mock_usecase.NewMockActionURLGenerator(ctrl)
					mock.EXPECT().GenerateDownloadURL("https://example.com", "owner", "repo", "1234567890123456789012345678901234567890123456789012345678901234").Return("https://example.com/owner/repo/objects/1234567890123456789012345678901234567890123456789012345678901234/download")
					return mock
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				return args{
					ctx:     context.Background(),
					baseURL: "https://example.com",
					owner:   "owner",
					repo:    "repo",
					oid:     oid,
					size:    size,
				}
			}(),
			want: func() usecase.ResponseObject {
				downloadAction := usecase.NewAction("https://example.com/owner/repo/objects/1234567890123456789012345678901234567890123456789012345678901234/download", nil, 900)
				actions := usecase.NewActions(nil, &downloadAction)
				return usecase.NewResponseObject("1234567890123456789012345678901234567890123456789012345678901234", 1024, true, &actions, nil)
			}(),
		},
		{
			name: "異常系: オブジェクトが存在しない場合、404エラーが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
					return mock
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				return args{
					ctx:     context.Background(),
					baseURL: "https://example.com",
					owner:   "owner",
					repo:    "repo",
					oid:     oid,
					size:    size,
				}
			}(),
			want: func() usecase.ResponseObject {
				objErr := usecase.NewObjectError(404, "オブジェクトが存在しません")
				return usecase.NewResponseObject("1234567890123456789012345678901234567890123456789012345678901234", 1024, false, nil, &objErr)
			}(),
		},
		{
			name: "異常系: メタデータ取得で内部エラーが発生した場合、500エラーが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, errors.New("database connection error"))
					return mock
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				return args{
					ctx:     context.Background(),
					baseURL: "https://example.com",
					owner:   "owner",
					repo:    "repo",
					oid:     oid,
					size:    size,
				}
			}(),
			want: func() usecase.ResponseObject {
				objErr := usecase.NewObjectError(500, "メタデータの取得に失敗しました")
				return usecase.NewResponseObject("1234567890123456789012345678901234567890123456789012345678901234", 1024, false, nil, &objErr)
			}(),
		},
		{
			name: "異常系: オブジェクトが存在するがアップロード未完了の場合、404エラーが返る",
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
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
			},
			args: func() args {
				oid, _ := domain.NewOID("1234567890123456789012345678901234567890123456789012345678901234")
				size, _ := domain.NewSize(1024)
				return args{
					ctx:     context.Background(),
					baseURL: "https://example.com",
					owner:   "owner",
					repo:    "repo",
					oid:     oid,
					size:    size,
				}
			}(),
			want: func() usecase.ResponseObject {
				objErr := usecase.NewObjectError(404, "オブジェクトがまだアップロードされていません")
				return usecase.NewResponseObject("1234567890123456789012345678901234567890123456789012345678901234", 1024, false, nil, &objErr)
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			uc := usecase.NewDownloadUseCase(
				tt.fields.repo(ctrl),
				tt.fields.actionURLGenerator(ctrl),
			)

			got := uc.HandleDownloadObject(tt.args.ctx, tt.args.baseURL, tt.args.owner, tt.args.repo, tt.args.oid, tt.args.size, tt.args.authHeader)

			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(
				usecase.ResponseObject{},
				usecase.Actions{},
				usecase.Action{},
				usecase.ObjectError{},
			)); diff != "" {
				t.Errorf("HandleDownloadObject() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
