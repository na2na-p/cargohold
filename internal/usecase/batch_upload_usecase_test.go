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

func TestBatchUploadUseCase_HandleBatchUpload(t *testing.T) {
	testOID := "1234567890123456789012345678901234567890123456789012345678901234"
	testRepo, _ := domain.NewRepositoryIdentifier("owner/repo")
	otherRepo, _ := domain.NewRepositoryIdentifier("other/repo")

	type fields struct {
		uploadUseCase func(ctrl *gomock.Controller) usecase.UploadUseCase
		policyRepo    func(ctrl *gomock.Controller) domain.AccessPolicyRepository
	}
	type args struct {
		ctx        context.Context
		baseURL    string
		owner      string
		repo       string
		req        usecase.BatchRequest
		authHeader string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    usecase.BatchResponse
		wantErr error
	}{
		{
			name: "正常系: Upload操作で新規オブジェクト（ポリシー未存在）の場合、署名付きURLが返りAccessPolicyが作成される",
			fields: fields{
				uploadUseCase: func(ctrl *gomock.Controller) usecase.UploadUseCase {
					mock := mock_usecase.NewMockUploadUseCase(ctrl)
					uploadAction := usecase.NewAction("https://s3.example.com/presigned-put-url", nil, 900)
					actions := usecase.NewActions(&uploadAction, nil)
					mock.EXPECT().HandleUploadObject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
						usecase.NewResponseObject(testOID, 1024, true, &actions, nil),
					)
					return mock
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, nil)
					mock.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
			},
			args: args{
				ctx:     context.Background(),
				baseURL: "http://localhost:8080",
				owner:   "owner",
				repo:    "repo",
				req: usecase.NewBatchRequest(
					domain.OperationUpload,
					[]usecase.RequestObject{usecase.NewRequestObject(testOID, 1024)},
					[]string{"basic"},
					nil,
					"sha256",
					testRepo,
				),
			},
			want: func() usecase.BatchResponse {
				uploadAction := usecase.NewAction("https://s3.example.com/presigned-put-url", nil, 900)
				actions := usecase.NewActions(&uploadAction, nil)
				return usecase.NewBatchResponse(
					"basic",
					[]usecase.ResponseObject{usecase.NewResponseObject(testOID, 1024, true, &actions, nil)},
					"sha256",
				)
			}(),
			wantErr: nil,
		},
		{
			name: "正常系: Upload操作でオブジェクトがアップロード済み（認可成功）の場合、アクションなしで返る",
			fields: fields{
				uploadUseCase: func(ctrl *gomock.Controller) usecase.UploadUseCase {
					mock := mock_usecase.NewMockUploadUseCase(ctrl)
					mock.EXPECT().HandleUploadObject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
						usecase.NewResponseObject(testOID, 1024, true, nil, nil),
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
				ctx:     context.Background(),
				baseURL: "http://localhost:8080",
				owner:   "owner",
				repo:    "repo",
				req: usecase.NewBatchRequest(
					domain.OperationUpload,
					[]usecase.RequestObject{usecase.NewRequestObject(testOID, 1024)},
					[]string{"basic"},
					nil,
					"sha256",
					testRepo,
				),
			},
			want: usecase.NewBatchResponse(
				"basic",
				[]usecase.ResponseObject{usecase.NewResponseObject(testOID, 1024, true, nil, nil)},
				"sha256",
			),
			wantErr: nil,
		},
		{
			name: "異常系: Upload操作で認可失敗（別リポジトリ）の場合、ErrAccessDeniedが返る",
			fields: fields{
				uploadUseCase: func(ctrl *gomock.Controller) usecase.UploadUseCase {
					return mock_usecase.NewMockUploadUseCase(ctrl)
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
				ctx:     context.Background(),
				baseURL: "http://localhost:8080",
				owner:   "owner",
				repo:    "repo",
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
			wantErr: usecase.ErrAccessDenied,
		},
		{
			name: "異常系: Repositoryがnilの場合、ErrAccessDeniedが返る",
			fields: fields{
				uploadUseCase: func(ctrl *gomock.Controller) usecase.UploadUseCase {
					return mock_usecase.NewMockUploadUseCase(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
			},
			args: args{
				ctx:     context.Background(),
				baseURL: "http://localhost:8080",
				owner:   "owner",
				repo:    "repo",
				req: usecase.NewBatchRequest(
					domain.OperationUpload,
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
			name: "異常系: Download操作を渡した場合、ErrInvalidOperationが返る",
			fields: fields{
				uploadUseCase: func(ctrl *gomock.Controller) usecase.UploadUseCase {
					return mock_usecase.NewMockUploadUseCase(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
			},
			args: args{
				ctx:     context.Background(),
				baseURL: "http://localhost:8080",
				owner:   "owner",
				repo:    "repo",
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
			wantErr: usecase.ErrInvalidOperation,
		},
		{
			name: "異常系: オブジェクトが空の場合、エラーが返る",
			fields: fields{
				uploadUseCase: func(ctrl *gomock.Controller) usecase.UploadUseCase {
					return mock_usecase.NewMockUploadUseCase(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
			},
			args: args{
				ctx:     context.Background(),
				baseURL: "http://localhost:8080",
				owner:   "owner",
				repo:    "repo",
				req: usecase.NewBatchRequest(
					domain.OperationUpload,
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
				uploadUseCase: func(ctrl *gomock.Controller) usecase.UploadUseCase {
					return mock_usecase.NewMockUploadUseCase(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
			},
			args: args{
				ctx:     context.Background(),
				baseURL: "http://localhost:8080",
				owner:   "owner",
				repo:    "repo",
				req: usecase.NewBatchRequest(
					domain.OperationUpload,
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
				uploadUseCase: func(ctrl *gomock.Controller) usecase.UploadUseCase {
					return mock_usecase.NewMockUploadUseCase(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
			},
			args: args{
				ctx:     context.Background(),
				baseURL: "http://localhost:8080",
				owner:   "owner",
				repo:    "repo",
				req: usecase.NewBatchRequest(
					domain.OperationUpload,
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

			policyRepo := tt.fields.policyRepo(ctrl)
			authService := domain.NewAccessAuthorizationService(policyRepo)
			uc := usecase.NewBatchUploadUseCase(
				tt.fields.uploadUseCase(ctrl),
				authService,
				policyRepo,
			)

			got, err := uc.HandleBatchUpload(tt.args.ctx, tt.args.baseURL, tt.args.owner, tt.args.repo, tt.args.req, tt.args.authHeader)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("HandleBatchUpload() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("HandleBatchUpload() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("HandleBatchUpload() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(
				usecase.BatchResponse{},
				usecase.ResponseObject{},
				usecase.Actions{},
				usecase.Action{},
				usecase.ObjectError{},
			)); diff != "" {
				t.Errorf("HandleBatchUpload() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
