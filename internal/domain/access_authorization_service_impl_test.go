package domain_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	mock_domain "github.com/na2na-p/cargohold/tests/domain"
	"go.uber.org/mock/gomock"
)

func TestNewAccessAuthorizationService(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := mock_domain.NewMockAccessPolicyRepository(ctrl)

	tests := []struct {
		name       string
		policyRepo domain.AccessPolicyRepository
		wantNil    bool
	}{
		{
			name:       "正常系: AccessPolicyRepositoryを渡すとサービスが生成される",
			policyRepo: mockRepo,
			wantNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.NewAccessAuthorizationService(tt.policyRepo)
			if (got == nil) != tt.wantNil {
				t.Errorf("NewAccessAuthorizationService() returned nil = %v, want nil = %v", got == nil, tt.wantNil)
			}
		})
	}
}

func TestAccessAuthorizationService_CanAccess(t *testing.T) {
	validOID, _ := domain.NewOID(strings.Repeat("a", 64))
	userRepo, _ := domain.NewRepositoryIdentifier("owner/repo")
	differentRepo, _ := domain.NewRepositoryIdentifier("other/repo")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	policyID, _ := domain.NewAccessPolicyID(1)

	errPolicyNotFound := errors.New("policy not found")

	type fields struct {
		policyRepo func(ctrl *gomock.Controller) domain.AccessPolicyRepository
	}
	type args struct {
		ctx      context.Context
		userRepo *domain.RepositoryIdentifier
		oid      domain.OID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr error
	}{
		{
			name: "正常系: ポリシーが存在し、リポジトリが一致する場合、trueが返る",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					policy := domain.NewAccessPolicy(policyID, validOID, userRepo, fixedTime)
					mock.EXPECT().FindByOID(gomock.Any(), validOID).Return(policy, nil)
					return mock
				},
			},
			args: args{
				ctx:      context.Background(),
				userRepo: userRepo,
				oid:      validOID,
			},
			want:    true,
			wantErr: nil,
		},
		{
			name: "正常系: ポリシーが存在し、リポジトリが一致しない場合、falseが返る",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					policy := domain.NewAccessPolicy(policyID, validOID, differentRepo, fixedTime)
					mock.EXPECT().FindByOID(gomock.Any(), validOID).Return(policy, nil)
					return mock
				},
			},
			args: args{
				ctx:      context.Background(),
				userRepo: userRepo,
				oid:      validOID,
			},
			want:    false,
			wantErr: nil,
		},
		{
			name: "異常系: ポリシーが存在しない場合、エラーが返る",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), validOID).Return(nil, errPolicyNotFound)
					return mock
				},
			},
			args: args{
				ctx:      context.Background(),
				userRepo: userRepo,
				oid:      validOID,
			},
			want:    false,
			wantErr: errPolicyNotFound,
		},
		{
			name: "異常系: ポリシーがnilで返された場合、ErrAccessPolicyNotFoundが返る",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), validOID).Return(nil, nil)
					return mock
				},
			},
			args: args{
				ctx:      context.Background(),
				userRepo: userRepo,
				oid:      validOID,
			},
			want:    false,
			wantErr: domain.ErrAccessPolicyNotFound,
		},
		{
			name: "異常系: userRepoがnilの場合、ErrInvalidRepositoryIdentifierが返る",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					return mock
				},
			},
			args: args{
				ctx:      context.Background(),
				userRepo: nil,
				oid:      validOID,
			},
			want:    false,
			wantErr: domain.ErrInvalidRepositoryIdentifier,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			service := domain.NewAccessAuthorizationService(tt.fields.policyRepo(ctrl))

			got, err := service.CanAccess(tt.args.ctx, tt.args.userRepo, tt.args.oid)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("CanAccess() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}

			if got != tt.want {
				t.Errorf("CanAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccessAuthorizationService_Authorize(t *testing.T) {
	validOID, _ := domain.NewOID(strings.Repeat("a", 64))
	userRepo, _ := domain.NewRepositoryIdentifier("owner/repo")
	differentRepo, _ := domain.NewRepositoryIdentifier("other/repo")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	policyID, _ := domain.NewAccessPolicyID(1)

	type fields struct {
		policyRepo func(ctrl *gomock.Controller) domain.AccessPolicyRepository
	}
	type args struct {
		ctx       context.Context
		operation domain.Operation
		userRepo  *domain.RepositoryIdentifier
		oid       domain.OID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    domain.AuthorizationResult
		wantErr error
	}{
		{
			name: "正常系: Download操作でポリシーが存在し、リポジトリが一致する場合、許可される",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					policy := domain.NewAccessPolicy(policyID, validOID, userRepo, fixedTime)
					mock.EXPECT().FindByOID(gomock.Any(), validOID).Return(policy, nil)
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				operation: domain.OperationDownload,
				userRepo:  userRepo,
				oid:       validOID,
			},
			want:    domain.AuthorizationResult{Allowed: true, IsNewObject: false},
			wantErr: nil,
		},
		{
			name: "異常系: Download操作でポリシーが存在せず、ErrAuthorizationDeniedが返る",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), validOID).Return(nil, nil)
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				operation: domain.OperationDownload,
				userRepo:  userRepo,
				oid:       validOID,
			},
			want:    domain.AuthorizationResult{Allowed: false, IsNewObject: false},
			wantErr: domain.ErrAuthorizationDenied,
		},
		{
			name: "異常系: Download操作でリポジトリが一致しない場合、ErrAuthorizationDeniedが返る",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					policy := domain.NewAccessPolicy(policyID, validOID, differentRepo, fixedTime)
					mock.EXPECT().FindByOID(gomock.Any(), validOID).Return(policy, nil)
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				operation: domain.OperationDownload,
				userRepo:  userRepo,
				oid:       validOID,
			},
			want:    domain.AuthorizationResult{Allowed: false, IsNewObject: false},
			wantErr: domain.ErrAuthorizationDenied,
		},
		{
			name: "正常系: Upload操作でポリシーが存在し、リポジトリが一致する場合、許可される（既存オブジェクト）",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					policy := domain.NewAccessPolicy(policyID, validOID, userRepo, fixedTime)
					mock.EXPECT().FindByOID(gomock.Any(), validOID).Return(policy, nil)
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				operation: domain.OperationUpload,
				userRepo:  userRepo,
				oid:       validOID,
			},
			want:    domain.AuthorizationResult{Allowed: true, IsNewObject: false},
			wantErr: nil,
		},
		{
			name: "正常系: Upload操作でポリシーが存在しない場合、新規オブジェクトとして許可される",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), validOID).Return(nil, nil)
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				operation: domain.OperationUpload,
				userRepo:  userRepo,
				oid:       validOID,
			},
			want:    domain.AuthorizationResult{Allowed: true, IsNewObject: true},
			wantErr: nil,
		},
		{
			name: "異常系: Upload操作でリポジトリが一致しない場合、ErrAuthorizationDeniedが返る",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					policy := domain.NewAccessPolicy(policyID, validOID, differentRepo, fixedTime)
					mock.EXPECT().FindByOID(gomock.Any(), validOID).Return(policy, nil)
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				operation: domain.OperationUpload,
				userRepo:  userRepo,
				oid:       validOID,
			},
			want:    domain.AuthorizationResult{Allowed: false, IsNewObject: false},
			wantErr: domain.ErrAuthorizationDenied,
		},
		{
			name: "異常系: userRepoがnilの場合、ErrInvalidRepositoryIdentifierが返る",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				operation: domain.OperationDownload,
				userRepo:  nil,
				oid:       validOID,
			},
			want:    domain.AuthorizationResult{Allowed: false, IsNewObject: false},
			wantErr: domain.ErrInvalidRepositoryIdentifier,
		},
		{
			name: "異常系: リポジトリ検索でエラーが発生した場合、エラーが返る",
			fields: fields{
				policyRepo: func(ctrl *gomock.Controller) domain.AccessPolicyRepository {
					mock := mock_domain.NewMockAccessPolicyRepository(ctrl)
					mock.EXPECT().FindByOID(gomock.Any(), validOID).Return(nil, errors.New("database error"))
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				operation: domain.OperationDownload,
				userRepo:  userRepo,
				oid:       validOID,
			},
			want:    domain.AuthorizationResult{Allowed: false, IsNewObject: false},
			wantErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			service := domain.NewAccessAuthorizationService(tt.fields.policyRepo(ctrl))

			got, err := service.Authorize(tt.args.ctx, tt.args.operation, tt.args.userRepo, tt.args.oid)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("Authorize() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Authorize() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
