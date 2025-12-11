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

func TestBatchUseCase_HandleBatchRequest(t *testing.T) {
	testOID := "1234567890123456789012345678901234567890123456789012345678901234"
	testRepo, _ := domain.NewRepositoryIdentifier("owner/repo")
	otherRepo, _ := domain.NewRepositoryIdentifier("other/repo")

	type fields struct {
		repo                func(ctrl *gomock.Controller) domain.LFSObjectRepository
		actionURLGenerator  func(ctrl *gomock.Controller) usecase.ActionURLGenerator
		policyRepo          func(ctrl *gomock.Controller) domain.AccessPolicyRepository
		storageKeyGenerator func(ctrl *gomock.Controller) usecase.StorageKeyGenerator
	}
	type args struct {
		ctx     context.Context
		baseURL string
		owner   string
		repo    string
		req     usecase.BatchRequest
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
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					ctx := context.Background()
					objOID, _ := domain.NewOID(testOID)
					size, _ := domain.NewSize(1024)
					hashAlgo, _ := domain.NewHashAlgorithm("sha256")
					obj, _ := domain.NewLFSObject(ctx, objOID, size, hashAlgo, "objects/sha256/12/34/"+testOID)
					obj.MarkAsUploaded(ctx)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(obj, nil)
					return mock
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					mock := mock_usecase.NewMockActionURLGenerator(ctrl)
					mock.EXPECT().GenerateDownloadURL(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("http://localhost:8080/owner/repo/objects/download/" + testOID)
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
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
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
			want: func() usecase.BatchResponse {
				downloadAction := usecase.NewAction("http://localhost:8080/owner/repo/objects/download/"+testOID, nil, 900)
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
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					objOID, _ := domain.NewOID(testOID)
					policyID, _ := domain.NewAccessPolicyID(1)
					policy := domain.NewAccessPolicy(policyID, objOID, otherRepo, time.Now())
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(policy, nil)
					return mock
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
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
			wantErr: usecase.ErrAccessDenied,
		},
		{
			name: "異常系: Download操作でポリシー未存在の場合、ErrAccessDeniedが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, nil)
					return mock
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
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
			wantErr: usecase.ErrAccessDenied,
		},
		{
			name: "正常系: Upload操作で新規オブジェクト（ポリシー未存在）の場合、署名付きURLが返りAccessPolicyが作成される",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
					mock.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					mock := mock_usecase.NewMockActionURLGenerator(ctrl)
					mock.EXPECT().GenerateUploadURL(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("http://localhost:8080/owner/repo/objects/upload/" + testOID)
					return mock
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(nil, nil)
					mock.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					mock := mock_usecase.NewMockStorageKeyGenerator(ctrl)
					mock.EXPECT().GenerateStorageKey(gomock.Any(), gomock.Any()).Return("objects/sha256/12/34/"+testOID, nil)
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
				uploadAction := usecase.NewAction("http://localhost:8080/owner/repo/objects/upload/"+testOID, nil, 900)
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
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					mock := mock_domain.NewMockLFSObjectRepository(ctrl)
					ctx := context.Background()
					objOID, _ := domain.NewOID(testOID)
					size, _ := domain.NewSize(1024)
					hashAlgo, _ := domain.NewHashAlgorithm("sha256")
					obj, _ := domain.NewLFSObject(ctx, objOID, size, hashAlgo, "objects/sha256/12/34/"+testOID)
					obj.MarkAsUploaded(ctx)
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(obj, nil)
					return mock
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					objOID, _ := domain.NewOID(testOID)
					policyID, _ := domain.NewAccessPolicyID(1)
					policy := domain.NewAccessPolicy(policyID, objOID, testRepo, time.Now())
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(policy, nil)
					return mock
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
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
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					objOID, _ := domain.NewOID(testOID)
					policyID, _ := domain.NewAccessPolicyID(1)
					policy := domain.NewAccessPolicy(policyID, objOID, otherRepo, time.Now())
					mock.EXPECT().FindByOID(gomock.Any(), gomock.Any()).Return(policy, nil)
					return mock
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
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
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
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
					nil,
				),
			},
			want:    usecase.BatchResponse{},
			wantErr: usecase.ErrAccessDenied,
		},
		{
			name: "異常系: 不正なオペレーション種別の場合、エラーが返る",
			fields: fields{
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
				},
			},
			args: args{
				ctx:     context.Background(),
				baseURL: "http://localhost:8080",
				owner:   "owner",
				repo:    "repo",
				req: usecase.NewBatchRequest(
					domain.Operation{},
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
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
				},
			},
			args: args{
				ctx:     context.Background(),
				baseURL: "http://localhost:8080",
				owner:   "owner",
				repo:    "repo",
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
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
				},
			},
			args: args{
				ctx:     context.Background(),
				baseURL: "http://localhost:8080",
				owner:   "owner",
				repo:    "repo",
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
				repo: func(ctrl *gomock.Controller) domain.LFSObjectRepository {
					return mock_domain.NewMockLFSObjectRepository(ctrl)
				},
				actionURLGenerator: func(ctrl *gomock.Controller) usecase.ActionURLGenerator {
					return mock_usecase.NewMockActionURLGenerator(ctrl)
				},
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					return mock_domain.NewMockAccessPolicyRepository(ctrl)
				},
				storageKeyGenerator: func(ctrl *gomock.Controller) usecase.StorageKeyGenerator {
					return mock_usecase.NewMockStorageKeyGenerator(ctrl)
				},
			},
			args: args{
				ctx:     context.Background(),
				baseURL: "http://localhost:8080",
				owner:   "owner",
				repo:    "repo",
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

			uc := usecase.NewBatchUseCase(
				tt.fields.repo(ctrl),
				tt.fields.actionURLGenerator(ctrl),
				tt.fields.policyRepo(ctrl),
				tt.fields.storageKeyGenerator(ctrl),
			)

			got, err := uc.HandleBatchRequest(tt.args.ctx, tt.args.baseURL, tt.args.owner, tt.args.repo, tt.args.req)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("HandleBatchRequest() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("HandleBatchRequest() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("HandleBatchRequest() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(
				usecase.BatchResponse{},
				usecase.ResponseObject{},
				usecase.Actions{},
				usecase.Action{},
				usecase.ObjectError{},
			)); diff != "" {
				t.Errorf("HandleBatchRequest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
