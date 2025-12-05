package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_domain "github.com/na2na-p/cargohold/tests/domain"
	mock_usecase "github.com/na2na-p/cargohold/tests/usecase"
	"go.uber.org/mock/gomock"
)

func TestBatchDownloadUseCase_HandleBatchDownload(t *testing.T) {
	testOID := "1234567890123456789012345678901234567890123456789012345678901234"
	testRepo, _ := domain.NewRepositoryIdentifier("owner/repo")
	otherRepo, _ := domain.NewRepositoryIdentifier("other/repo")

	type fields struct {
		downloadUseCase func(ctrl *gomock.Controller) usecase.DownloadUseCase
		policyRepo      func(ctrl *gomock.Controller) domain.AccessPolicyRepository
	}
	type args struct {
		ctx context.Context
		req usecase.BatchRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    usecase.BatchResponse
		wantErr error
	}{
		{
			name: "正常系: Download操作でオブジェクトが存在し認可成功の場合、署名付きURLが返る",
			fields: fields{
				downloadUseCase: func(ctrl *gomock.Controller) usecase.DownloadUseCase {
					mock := mock_usecase.NewMockDownloadUseCase(ctrl)
					downloadAction := usecase.NewAction("https://s3.example.com/presigned-get-url", nil, 900)
					actions := usecase.NewActions(nil, &downloadAction)
					mock.EXPECT().HandleDownloadObject(gomock.Any(), gomock.Any(), gomock.Any()).Return(
						usecase.NewResponseObject(testOID, 1024, true, &actions, nil),
					)
					return mock
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					objOID, _ := domain.NewOID(testOID)
					policyID, _ := domain.NewAccessPolicyID(1)
					policy := domain.NewAccessPolicy(policyID, objOID, testRepo, time.Now())
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(policy, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				req: usecase.NewBatchRequest(
					domain.OperationDownload,
					[]usecase.RequestObject{usecase.NewRequestObject(testOID, 1024)},
					[]string{"basic"},
					nil,
					"sha256",
					testRepo,
				),
			},
			want: func() usecase.BatchResponse {
				downloadAction := usecase.NewAction("https://s3.example.com/presigned-get-url", nil, 900)
				actions := usecase.NewActions(nil, &downloadAction)
				return usecase.NewBatchResponse(
					"basic",
					[]usecase.ResponseObject{usecase.NewResponseObject(testOID, 1024, true, &actions, nil)},
					"sha256",
				)
			}(),
			wantErr: nil,
		},
		{
			name: "異常系: Download操作で認可失敗（別リポジトリ）の場合、ErrAccessDeniedが返る",
			fields: fields{
				downloadUseCase: func(ctrl *gomock.Controller) usecase.DownloadUseCase {
					return mock_usecase.NewMockDownloadUseCase(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					objOID, _ := domain.NewOID(testOID)
					policyID, _ := domain.NewAccessPolicyID(1)
					policy := domain.NewAccessPolicy(policyID, objOID, otherRepo, time.Now())
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(policy, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				req: usecase.NewBatchRequest(
					domain.OperationDownload,
					[]usecase.RequestObject{usecase.NewRequestObject(testOID, 1024)},
					[]string{"basic"},
					nil,
					"sha256",
					testRepo,
				),
			},
			want:    usecase.BatchResponse{},
			wantErr: usecase.ErrAccessDenied,
		},
		{
			name: "異常系: Download操作でポリシー未存在の場合、ErrAccessDeniedが返る",
			fields: fields{
				downloadUseCase: func(ctrl *gomock.Controller) usecase.DownloadUseCase {
					return mock_usecase.NewMockDownloadUseCase(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				req: usecase.NewBatchRequest(
					domain.OperationDownload,
					[]usecase.RequestObject{usecase.NewRequestObject(testOID, 1024)},
					[]string{"basic"},
					nil,
					"sha256",
					testRepo,
				),
			},
			want:    usecase.BatchResponse{},
			wantErr: usecase.ErrAccessDenied,
		},
		{
			name: "異常系: Repositoryがnilの場合、ErrAccessDeniedが返る",
			fields: fields{
				downloadUseCase: func(ctrl *gomock.Controller) usecase.DownloadUseCase {
					return mock_usecase.NewMockDownloadUseCase(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
			},
			args: args{
				ctx: context.Background(),
				req: usecase.NewBatchRequest(
					domain.OperationDownload,
					[]usecase.RequestObject{usecase.NewRequestObject(testOID, 1024)},
					[]string{"basic"},
					nil,
					"sha256",
					nil,
				),
			},
			want:    usecase.BatchResponse{},
			wantErr: usecase.ErrAccessDenied,
		},
		{
			name: "異常系: Upload操作を渡した場合、ErrInvalidOperationが返る",
			fields: fields{
				downloadUseCase: func(ctrl *gomock.Controller) usecase.DownloadUseCase {
					return mock_usecase.NewMockDownloadUseCase(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
			},
			args: args{
				ctx: context.Background(),
				req: usecase.NewBatchRequest(
					domain.OperationUpload,
					[]usecase.RequestObject{usecase.NewRequestObject(testOID, 1024)},
					[]string{"basic"},
					nil,
					"sha256",
					testRepo,
				),
			},
			want:    usecase.BatchResponse{},
			wantErr: usecase.ErrInvalidOperation,
		},
		{
			name: "異常系: オブジェクトが空の場合、エラーが返る",
			fields: fields{
				downloadUseCase: func(ctrl *gomock.Controller) usecase.DownloadUseCase {
					return mock_usecase.NewMockDownloadUseCase(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
			},
			args: args{
				ctx: context.Background(),
				req: usecase.NewBatchRequest(
					domain.OperationDownload,
					[]usecase.RequestObject{},
					[]string{"basic"},
					nil,
					"sha256",
					testRepo,
				),
			},
			want:    usecase.BatchResponse{},
			wantErr: usecase.ErrNoObjects,
		},
		{
			name: "異常系: 不正なOID形式の場合、エラーが返る",
			fields: fields{
				downloadUseCase: func(ctrl *gomock.Controller) usecase.DownloadUseCase {
					return mock_usecase.NewMockDownloadUseCase(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
			},
			args: args{
				ctx: context.Background(),
				req: usecase.NewBatchRequest(
					domain.OperationDownload,
					[]usecase.RequestObject{usecase.NewRequestObject("invalid-oid", 1024)},
					[]string{"basic"},
					nil,
					"sha256",
					testRepo,
				),
			},
			want:    usecase.BatchResponse{},
			wantErr: usecase.ErrInvalidOID,
		},
		{
			name: "異常系: 不正なサイズの場合、エラーが返る",
			fields: fields{
				downloadUseCase: func(ctrl *gomock.Controller) usecase.DownloadUseCase {
					return mock_usecase.NewMockDownloadUseCase(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
			},
			args: args{
				ctx: context.Background(),
				req: usecase.NewBatchRequest(
					domain.OperationDownload,
					[]usecase.RequestObject{usecase.NewRequestObject(testOID, -1)},
					[]string{"basic"},
					nil,
					"sha256",
					testRepo,
				),
			},
			want:    usecase.BatchResponse{},
			wantErr: usecase.ErrInvalidSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			authService := domain.NewAccessAuthorizationService(tt.fields.policyRepo(ctrl))
			uc := usecase.NewBatchDownloadUseCase(
				tt.fields.downloadUseCase(ctrl),
				authService,
			)

			got, err := uc.HandleBatchDownload(tt.args.ctx, tt.args.req)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("HandleBatchDownload() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("HandleBatchDownload() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("HandleBatchDownload() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(
				usecase.BatchResponse{},
				usecase.ResponseObject{},
				usecase.Actions{},
				usecase.Action{},
				usecase.ObjectError{},
			)); diff != "" {
				t.Errorf("HandleBatchDownload() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
