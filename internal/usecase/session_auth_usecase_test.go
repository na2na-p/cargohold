package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_usecase "github.com/na2na-p/cargohold/tests/usecase"
	"go.uber.org/mock/gomock"
)

func mustNewUserInfoInSessionTest(t *testing.T, sub, email, name string, provider domain.ProviderType, repository *domain.RepositoryIdentifier, ref string) *domain.UserInfo {
	t.Helper()
	userInfo, err := domain.NewUserInfo(sub, email, name, provider, repository, ref)
	if err != nil {
		t.Fatalf("failed to create UserInfo: %v", err)
	}
	return userInfo
}

func TestSessionAuthUseCase_Authenticate(t *testing.T) {
	ownerRepo, _ := domain.NewRepositoryIdentifier("owner/repo")

	type args struct {
		sessionID string
	}
	type mockFields struct {
		sessionClient *mock_usecase.MockSessionClient
	}
	tests := []struct {
		name       string
		setupMocks func(ctrl *gomock.Controller) mockFields
		args       args
		want       *domain.UserInfo
		wantErr    error
	}{
		{
			name: "正常系: セッションが存在する場合、ユーザー情報を返す",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				sessionClient := mock_usecase.NewMockSessionClient(ctrl)
				expectedUserInfo := mustNewUserInfoInSessionTest(t,
					"test-sub",
					"test@example.com",
					"Test User",
					domain.ProviderTypeGitHub,
					nil,
					"",
				)
				sessionClient.EXPECT().GetSession(gomock.Any(), "test-session-id").Return(expectedUserInfo, nil)
				return mockFields{sessionClient: sessionClient}
			},
			args: args{
				sessionID: "test-session-id",
			},
			want:    mustNewUserInfoInSessionTest(t, "test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, nil, ""),
			wantErr: nil,
		},
		{
			name: "正常系: GitHub Actionsセッションが存在する場合、リポジトリ情報を含むユーザー情報を返す",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				sessionClient := mock_usecase.NewMockSessionClient(ctrl)
				expectedUserInfo := mustNewUserInfoInSessionTest(t,
					"repo:owner/repo:ref:refs/heads/main",
					"",
					"github-actor",
					domain.ProviderTypeGitHub,
					ownerRepo,
					"refs/heads/main",
				)
				sessionClient.EXPECT().GetSession(gomock.Any(), "github-session-id").Return(expectedUserInfo, nil)
				return mockFields{sessionClient: sessionClient}
			},
			args: args{
				sessionID: "github-session-id",
			},
			want: mustNewUserInfoInSessionTest(t,
				"repo:owner/repo:ref:refs/heads/main",
				"",
				"github-actor",
				domain.ProviderTypeGitHub,
				ownerRepo,
				"refs/heads/main",
			),
			wantErr: nil,
		},
		{
			name: "正常系: GitHub Actions PRセッションが存在する場合",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				sessionClient := mock_usecase.NewMockSessionClient(ctrl)
				expectedUserInfo := mustNewUserInfoInSessionTest(t,
					"repo:owner/repo:ref:refs/pull/42/merge",
					"",
					"pr-author",
					domain.ProviderTypeGitHub,
					ownerRepo,
					"refs/pull/42/merge",
				)
				sessionClient.EXPECT().GetSession(gomock.Any(), "github-pr-session-id").Return(expectedUserInfo, nil)
				return mockFields{sessionClient: sessionClient}
			},
			args: args{
				sessionID: "github-pr-session-id",
			},
			want: mustNewUserInfoInSessionTest(t,
				"repo:owner/repo:ref:refs/pull/42/merge",
				"",
				"pr-author",
				domain.ProviderTypeGitHub,
				ownerRepo,
				"refs/pull/42/merge",
			),
			wantErr: nil,
		},
		{
			name: "異常系: セッションが存在しない場合、ErrSessionNotFound",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				sessionClient := mock_usecase.NewMockSessionClient(ctrl)
				sessionClient.EXPECT().GetSession(gomock.Any(), "invalid-session-id").Return(nil, errors.New("session not found"))
				return mockFields{sessionClient: sessionClient}
			},
			args: args{
				sessionID: "invalid-session-id",
			},
			want:    nil,
			wantErr: usecase.ErrSessionNotFound,
		},
		{
			name: "異常系: SessionClient エラーの場合、ErrSessionNotFound",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				sessionClient := mock_usecase.NewMockSessionClient(ctrl)
				sessionClient.EXPECT().GetSession(gomock.Any(), "error-session-id").Return(nil, errors.New("redis connection error"))
				return mockFields{sessionClient: sessionClient}
			},
			args: args{
				sessionID: "error-session-id",
			},
			want:    nil,
			wantErr: usecase.ErrSessionNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mocks := tt.setupMocks(ctrl)
			uc := usecase.NewSessionAuthUseCase(mocks.sessionClient)

			got, err := uc.Authenticate(ctx, tt.args.sessionID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Authenticate() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("Authenticate() unexpected error: %v", err)
				}
				if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(domain.UserInfo{}, domain.ProviderType{}, domain.RepositoryIdentifier{})); diff != "" {
					t.Errorf("Authenticate() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
