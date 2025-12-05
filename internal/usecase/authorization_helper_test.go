package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_domain "github.com/na2na-p/cargohold/tests/domain"
	"go.uber.org/mock/gomock"
)

func TestCheckAuthorization(t *testing.T) {
	testOID := "1234567890123456789012345678901234567890123456789012345678901234"
	testRepo, _ := domain.NewRepositoryIdentifier("owner/repo")
	objOID, _ := domain.NewOID(testOID)

	type fields struct {
		authService func(ctrl *gomock.Controller) domain.AccessAuthorizationService
	}
	type args struct {
		ctx      context.Context
		op       domain.Operation
		userRepo *domain.RepositoryIdentifier
		oid      domain.OID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    domain.AuthorizationResult
		wantErr error
	}{
		{
			name: "正常系: Upload操作で認可が成功した場合、AuthorizationResultが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationUpload, testRepo, objOID).Return(
						domain.AuthorizationResult{Allowed: true, IsNewObject: true},
						nil,
					)
					return mock
				},
			},
			args: args{
				ctx:      context.Background(),
				op:       domain.OperationUpload,
				userRepo: testRepo,
				oid:      objOID,
			},
			want:    domain.AuthorizationResult{Allowed: true, IsNewObject: true},
			wantErr: nil,
		},
		{
			name: "正常系: Download操作で認可が成功した場合、AuthorizationResultが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationDownload, testRepo, objOID).Return(
						domain.AuthorizationResult{Allowed: true, IsNewObject: false},
						nil,
					)
					return mock
				},
			},
			args: args{
				ctx:      context.Background(),
				op:       domain.OperationDownload,
				userRepo: testRepo,
				oid:      objOID,
			},
			want:    domain.AuthorizationResult{Allowed: true, IsNewObject: false},
			wantErr: nil,
		},
		{
			name: "正常系: 認可が拒否された場合（Allowed=false）、結果がそのまま返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationDownload, testRepo, objOID).Return(
						domain.AuthorizationResult{Allowed: false, IsNewObject: false},
						nil,
					)
					return mock
				},
			},
			args: args{
				ctx:      context.Background(),
				op:       domain.OperationDownload,
				userRepo: testRepo,
				oid:      objOID,
			},
			want:    domain.AuthorizationResult{Allowed: false, IsNewObject: false},
			wantErr: nil,
		},
		{
			name: "異常系: ErrAuthorizationDeniedの場合、ErrAccessDeniedが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationUpload, testRepo, objOID).Return(
						domain.AuthorizationResult{},
						domain.ErrAuthorizationDenied,
					)
					return mock
				},
			},
			args: args{
				ctx:      context.Background(),
				op:       domain.OperationUpload,
				userRepo: testRepo,
				oid:      objOID,
			},
			want:    domain.AuthorizationResult{},
			wantErr: usecase.ErrAccessDenied,
		},
		{
			name: "異常系: ErrInvalidRepositoryIdentifierの場合、ErrAccessDeniedが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationDownload, gomock.Nil(), objOID).Return(
						domain.AuthorizationResult{},
						domain.ErrInvalidRepositoryIdentifier,
					)
					return mock
				},
			},
			args: args{
				ctx:      context.Background(),
				op:       domain.OperationDownload,
				userRepo: nil,
				oid:      objOID,
			},
			want:    domain.AuthorizationResult{},
			wantErr: usecase.ErrAccessDenied,
		},
		{
			name: "異常系: その他のエラーの場合、ラップされたエラーが返る",
			fields: fields{
				authService: func(ctrl *gomock.Controller) domain.AccessAuthorizationService {
					mock := mock_domain.NewMockAccessAuthorizationService(ctrl)
					mock.EXPECT().Authorize(gomock.Any(), domain.OperationUpload, testRepo, objOID).Return(
						domain.AuthorizationResult{},
						errors.New("unexpected error"),
					)
					return mock
				},
			},
			args: args{
				ctx:      context.Background(),
				op:       domain.OperationUpload,
				userRepo: testRepo,
				oid:      objOID,
			},
			want:    domain.AuthorizationResult{},
			wantErr: errors.New("認可判定に失敗しました"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			authService := tt.fields.authService(ctrl)
			got, err := usecase.CheckAuthorization(tt.args.ctx, authService, tt.args.op, tt.args.userRepo, tt.args.oid)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("CheckAuthorization() error = nil, wantErr %v", tt.wantErr)
				}
				if errors.Is(tt.wantErr, usecase.ErrAccessDenied) {
					if !errors.Is(err, usecase.ErrAccessDenied) {
						t.Errorf("CheckAuthorization() error = %v, wantErr %v", err, tt.wantErr)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("CheckAuthorization() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("CheckAuthorization() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
