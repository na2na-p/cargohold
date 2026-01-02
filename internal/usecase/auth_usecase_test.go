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

func mustNewUserInfo(t *testing.T, sub, email, name string, provider domain.ProviderType, repository *domain.RepositoryIdentifier, ref string) *domain.UserInfo {
	t.Helper()
	userInfo, err := domain.NewUserInfo(sub, email, name, provider, repository, ref)
	if err != nil {
		t.Fatalf("failed to create UserInfo: %v", err)
	}
	return userInfo
}

func TestAuthUseCase_AuthenticateSession(t *testing.T) {
	ownerRepo, _ := domain.NewRepositoryIdentifier("owner/repo")

	type args struct {
		sessionID string
	}
	type mockFields struct {
		redisClient  *mock_usecase.MockCacheClient
		keyGenerator *mock_usecase.MockCacheKeyGenerator
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
				redisClient := mock_usecase.NewMockCacheClient(ctrl)
				keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
				keyGenerator.EXPECT().SessionKey(gomock.Any()).Return("lfs:session:test-session-id")
				redisClient.EXPECT().GetJSON(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, key string, dest interface{}) error {
						sessionData := &usecase.SessionData{
							Sub:      "test-sub",
							Email:    "test@example.com",
							Name:     "Test User",
							Provider: "github",
						}
						if m, ok := dest.(*usecase.SessionData); ok {
							*m = *sessionData
						}
						return nil
					},
				)
				return mockFields{redisClient: redisClient, keyGenerator: keyGenerator}
			},
			args: args{
				sessionID: "test-session-id",
			},
			want:    mustNewUserInfo(t, "test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, nil, ""),
			wantErr: nil,
		},
		{
			name: "正常系: GitHub Actionsセッションが存在する場合、リポジトリ情報を含むユーザー情報を返す",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				redisClient := mock_usecase.NewMockCacheClient(ctrl)
				keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
				keyGenerator.EXPECT().SessionKey(gomock.Any()).Return("lfs:session:github-session-id")
				redisClient.EXPECT().GetJSON(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, key string, dest interface{}) error {
						sessionData := &usecase.SessionData{
							Sub:        "repo:owner/repo:ref:refs/heads/main",
							Email:      "",
							Name:       "github-actor",
							Provider:   "github",
							Repository: "owner/repo",
							Ref:        "refs/heads/main",
						}
						if m, ok := dest.(*usecase.SessionData); ok {
							*m = *sessionData
						}
						return nil
					},
				)
				return mockFields{redisClient: redisClient, keyGenerator: keyGenerator}
			},
			args: args{
				sessionID: "github-session-id",
			},
			want: mustNewUserInfo(t,
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
			name: "異常系: セッションが存在しない場合、ErrSessionNotFound",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				redisClient := mock_usecase.NewMockCacheClient(ctrl)
				keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
				keyGenerator.EXPECT().SessionKey(gomock.Any()).Return("lfs:session:invalid-session-id")
				redisClient.EXPECT().GetJSON(gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.ErrCacheMiss)
				return mockFields{redisClient: redisClient, keyGenerator: keyGenerator}
			},
			args: args{
				sessionID: "invalid-session-id",
			},
			want:    nil,
			wantErr: usecase.ErrSessionNotFound,
		},
		{
			name: "異常系: セッションデータにsubが含まれていない場合、ErrInvalidSessionData",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				redisClient := mock_usecase.NewMockCacheClient(ctrl)
				keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
				keyGenerator.EXPECT().SessionKey(gomock.Any()).Return("lfs:session:malformed-session-id")
				redisClient.EXPECT().GetJSON(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, key string, dest interface{}) error {
						sessionData := &usecase.SessionData{
							Sub:   "",
							Email: "test@example.com",
							Name:  "Test User",
						}
						if m, ok := dest.(*usecase.SessionData); ok {
							*m = *sessionData
						}
						return nil
					},
				)
				return mockFields{redisClient: redisClient, keyGenerator: keyGenerator}
			},
			args: args{
				sessionID: "malformed-session-id",
			},
			want:    nil,
			wantErr: usecase.ErrInvalidSessionData,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mocks := tt.setupMocks(ctrl)
			uc := usecase.NewAuthUseCase(nil, nil, mocks.redisClient, mocks.keyGenerator)

			got, err := uc.AuthenticateSession(ctx, tt.args.sessionID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("AuthenticateSession() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("AuthenticateSession() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("AuthenticateSession() unexpected error: %v", err)
				}
				if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(domain.UserInfo{}, domain.ProviderType{}, domain.RepositoryIdentifier{})); diff != "" {
					t.Errorf("AuthenticateSession() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestAuthUseCase_AuthenticateGitHubOIDC(t *testing.T) {
	ownerRepo, _ := domain.NewRepositoryIdentifier("owner/repo")

	type args struct {
		token string
	}
	type fields struct {
		setupGitHubProvider func(ctrl *gomock.Controller) *mock_usecase.MockGitHubOIDCProvider
		setupRepoAllowlist  func(ctrl *gomock.Controller) *mock_domain.MockRepositoryAllowlistRepository
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       *domain.UserInfo
		wantErrMsg string
	}{
		{
			name: "正常系: GitHub JWTトークンを検証し、ユーザー情報を返す",
			fields: fields{
				setupGitHubProvider: func(ctrl *gomock.Controller) *mock_usecase.MockGitHubOIDCProvider {
					githubProvider := mock_usecase.NewMockGitHubOIDCProvider(ctrl)
					githubProvider.EXPECT().VerifyIDToken(gomock.Any(), "valid-github-token").Return(
						domain.NewGitHubUserInfo(
							"repo:owner/repo:ref:refs/heads/main",
							"owner/repo",
							"refs/heads/main",
							"github-actor",
						), nil)
					return githubProvider
				},
				setupRepoAllowlist: func(ctrl *gomock.Controller) *mock_domain.MockRepositoryAllowlistRepository {
					repoAllowlist := mock_domain.NewMockRepositoryAllowlistRepository(ctrl)
					repoAllowlist.EXPECT().IsAllowed(gomock.Any(), gomock.Any()).Return(true, nil)
					return repoAllowlist
				},
			},
			args: args{
				token: "valid-github-token",
			},
			want: mustNewUserInfoWithFullPermissions(t,
				"repo:owner/repo:ref:refs/heads/main",
				"",
				"github-actor",
				domain.ProviderTypeGitHub,
				ownerRepo,
				"refs/heads/main",
			),
			wantErrMsg: "",
		},
		{
			name: "異常系: トークン検証失敗",
			fields: fields{
				setupGitHubProvider: func(ctrl *gomock.Controller) *mock_usecase.MockGitHubOIDCProvider {
					githubProvider := mock_usecase.NewMockGitHubOIDCProvider(ctrl)
					githubProvider.EXPECT().VerifyIDToken(gomock.Any(), "invalid-github-token").Return(nil, errors.New("トークン検証エラー"))
					return githubProvider
				},
				setupRepoAllowlist: func(ctrl *gomock.Controller) *mock_domain.MockRepositoryAllowlistRepository {
					return mock_domain.NewMockRepositoryAllowlistRepository(ctrl)
				},
			},
			args: args{
				token: "invalid-github-token",
			},
			want:       nil,
			wantErrMsg: "トークン検証エラー",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			githubProvider := tt.fields.setupGitHubProvider(ctrl)
			repoAllowlist := tt.fields.setupRepoAllowlist(ctrl)
			uc := usecase.NewAuthUseCase(githubProvider, repoAllowlist, nil, nil)

			got, err := uc.AuthenticateGitHubOIDC(ctx, tt.args.token)

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("AuthenticateGitHubOIDC() error = nil, wantErrMsg %q", tt.wantErrMsg)
				}
				if err.Error() != tt.wantErrMsg {
					t.Errorf("AuthenticateGitHubOIDC() error = %q, wantErrMsg %q", err.Error(), tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("AuthenticateGitHubOIDC() unexpected error: %v", err)
				}
				if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(domain.UserInfo{}, domain.ProviderType{}, domain.RepositoryIdentifier{}, domain.RepositoryPermissions{})); diff != "" {
					t.Errorf("AuthenticateGitHubOIDC() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestAuthUseCase_AuthenticateGitHubOIDC_NotConfigured(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr error
	}{
		{
			name:    "異常系: GitHub OIDCが設定されていない場合、ErrGitHubOIDCNotConfigured",
			args:    "any-token",
			wantErr: usecase.ErrGitHubOIDCNotConfigured,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			redisClient := mock_usecase.NewMockCacheClient(ctrl)
			keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
			uc := usecase.NewAuthUseCase(nil, nil, redisClient, keyGenerator)

			got, err := uc.AuthenticateGitHubOIDC(ctx, tt.args)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("AuthenticateGitHubOIDC() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("AuthenticateGitHubOIDC() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
			if got != nil {
				t.Errorf("AuthenticateGitHubOIDC() got = %v, want nil", got)
			}
		})
	}
}
