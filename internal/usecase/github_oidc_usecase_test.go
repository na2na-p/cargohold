package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
	mockdomain "github.com/na2na-p/cargohold/tests/domain"
	mock_usecase "github.com/na2na-p/cargohold/tests/usecase"
	"go.uber.org/mock/gomock"
)

func TestGitHubOIDCUseCase_Authenticate(t *testing.T) {
	ownerRepo, _ := domain.NewRepositoryIdentifier("owner/repo")

	type fields struct {
		setupGitHubProvider func(ctrl *gomock.Controller) *mock_usecase.MockGitHubOIDCProvider
		setupRepoAllowlist  func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository
	}
	type args struct {
		token string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domain.UserInfo
		wantErr error
	}{
		{
			name: "正常系: GitHub JWTトークンを検証し、ユーザー情報を返す（全権限付与）",
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
				setupRepoAllowlist: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
					repoAllowlist := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
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
			wantErr: nil,
		},
		{
			name: "正常系: GitHub Actions PRイベントのトークンを検証（全権限付与）",
			fields: fields{
				setupGitHubProvider: func(ctrl *gomock.Controller) *mock_usecase.MockGitHubOIDCProvider {
					githubProvider := mock_usecase.NewMockGitHubOIDCProvider(ctrl)
					githubProvider.EXPECT().VerifyIDToken(gomock.Any(), "pr-event-token").Return(
						domain.NewGitHubUserInfo(
							"repo:owner/repo:ref:refs/pull/123/merge",
							"owner/repo",
							"refs/pull/123/merge",
							"pr-author",
						), nil)
					return githubProvider
				},
				setupRepoAllowlist: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
					repoAllowlist := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					repoAllowlist.EXPECT().IsAllowed(gomock.Any(), gomock.Any()).Return(true, nil)
					return repoAllowlist
				},
			},
			args: args{
				token: "pr-event-token",
			},
			want: mustNewUserInfoWithFullPermissions(t,
				"repo:owner/repo:ref:refs/pull/123/merge",
				"",
				"pr-author",
				domain.ProviderTypeGitHub,
				ownerRepo,
				"refs/pull/123/merge",
			),
			wantErr: nil,
		},
		{
			name: "正常系: GitHub Actions タグイベントのトークンを検証（全権限付与）",
			fields: fields{
				setupGitHubProvider: func(ctrl *gomock.Controller) *mock_usecase.MockGitHubOIDCProvider {
					githubProvider := mock_usecase.NewMockGitHubOIDCProvider(ctrl)
					githubProvider.EXPECT().VerifyIDToken(gomock.Any(), "tag-event-token").Return(
						domain.NewGitHubUserInfo(
							"repo:owner/repo:ref:refs/tags/v1.0.0",
							"owner/repo",
							"refs/tags/v1.0.0",
							"release-author",
						), nil)
					return githubProvider
				},
				setupRepoAllowlist: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
					repoAllowlist := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					repoAllowlist.EXPECT().IsAllowed(gomock.Any(), gomock.Any()).Return(true, nil)
					return repoAllowlist
				},
			},
			args: args{
				token: "tag-event-token",
			},
			want: mustNewUserInfoWithFullPermissions(t,
				"repo:owner/repo:ref:refs/tags/v1.0.0",
				"",
				"release-author",
				domain.ProviderTypeGitHub,
				ownerRepo,
				"refs/tags/v1.0.0",
			),
			wantErr: nil,
		},
		{
			name: "異常系: トークン検証失敗",
			fields: fields{
				setupGitHubProvider: func(ctrl *gomock.Controller) *mock_usecase.MockGitHubOIDCProvider {
					githubProvider := mock_usecase.NewMockGitHubOIDCProvider(ctrl)
					githubProvider.EXPECT().VerifyIDToken(gomock.Any(), "invalid-github-token").Return(nil, errors.New("トークン検証エラー"))
					return githubProvider
				},
				setupRepoAllowlist: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
					return mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
				},
			},
			args: args{
				token: "invalid-github-token",
			},
			want:    nil,
			wantErr: errors.New("トークン検証エラー"),
		},
		{
			name: "異常系: 期限切れトークン",
			fields: fields{
				setupGitHubProvider: func(ctrl *gomock.Controller) *mock_usecase.MockGitHubOIDCProvider {
					githubProvider := mock_usecase.NewMockGitHubOIDCProvider(ctrl)
					githubProvider.EXPECT().VerifyIDToken(gomock.Any(), "expired-token").Return(nil, errors.New("トークンの有効期限が切れています"))
					return githubProvider
				},
				setupRepoAllowlist: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
					return mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
				},
			},
			args: args{
				token: "expired-token",
			},
			want:    nil,
			wantErr: errors.New("トークンの有効期限が切れています"),
		},
		{
			name: "異常系: 不正な署名のトークン",
			fields: fields{
				setupGitHubProvider: func(ctrl *gomock.Controller) *mock_usecase.MockGitHubOIDCProvider {
					githubProvider := mock_usecase.NewMockGitHubOIDCProvider(ctrl)
					githubProvider.EXPECT().VerifyIDToken(gomock.Any(), "bad-signature-token").Return(nil, errors.New("トークンの署名が無効です"))
					return githubProvider
				},
				setupRepoAllowlist: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
					return mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
				},
			},
			args: args{
				token: "bad-signature-token",
			},
			want:    nil,
			wantErr: errors.New("トークンの署名が無効です"),
		},
		{
			name: "異常系: プロバイダーがnilを返す",
			fields: fields{
				setupGitHubProvider: func(ctrl *gomock.Controller) *mock_usecase.MockGitHubOIDCProvider {
					githubProvider := mock_usecase.NewMockGitHubOIDCProvider(ctrl)
					githubProvider.EXPECT().VerifyIDToken(gomock.Any(), "nil-result-token").Return(nil, nil)
					return githubProvider
				},
				setupRepoAllowlist: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
					return mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
				},
			},
			args: args{
				token: "nil-result-token",
			},
			want:    nil,
			wantErr: errors.New("github provider returned nil user info"),
		},
		{
			name: "異常系: 許可されていないリポジトリ",
			fields: fields{
				setupGitHubProvider: func(ctrl *gomock.Controller) *mock_usecase.MockGitHubOIDCProvider {
					githubProvider := mock_usecase.NewMockGitHubOIDCProvider(ctrl)
					githubProvider.EXPECT().VerifyIDToken(gomock.Any(), "unauthorized-repo-token").Return(
						domain.NewGitHubUserInfo(
							"repo:unauthorized/repo:ref:refs/heads/main",
							"unauthorized/repo",
							"refs/heads/main",
							"github-actor",
						), nil)
					return githubProvider
				},
				setupRepoAllowlist: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
					repoAllowlist := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					repoAllowlist.EXPECT().IsAllowed(gomock.Any(), gomock.Any()).Return(false, nil)
					return repoAllowlist
				},
			},
			args: args{
				token: "unauthorized-repo-token",
			},
			want:    nil,
			wantErr: usecase.ErrInvalidRepository,
		},
		{
			name: "異常系: リポジトリ許可チェックでエラー",
			fields: fields{
				setupGitHubProvider: func(ctrl *gomock.Controller) *mock_usecase.MockGitHubOIDCProvider {
					githubProvider := mock_usecase.NewMockGitHubOIDCProvider(ctrl)
					githubProvider.EXPECT().VerifyIDToken(gomock.Any(), "check-error-token").Return(
						domain.NewGitHubUserInfo(
							"repo:owner/repo:ref:refs/heads/main",
							"owner/repo",
							"refs/heads/main",
							"github-actor",
						), nil)
					return githubProvider
				},
				setupRepoAllowlist: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
					repoAllowlist := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					repoAllowlist.EXPECT().IsAllowed(gomock.Any(), gomock.Any()).Return(false, errors.New("DB接続エラー"))
					return repoAllowlist
				},
			},
			args: args{
				token: "check-error-token",
			},
			want:    nil,
			wantErr: errors.New("リポジトリ許可チェックに失敗しました: DB接続エラー"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			githubProvider := tt.fields.setupGitHubProvider(ctrl)
			repoAllowlist := tt.fields.setupRepoAllowlist(ctrl)
			uc := usecase.NewGitHubOIDCUseCase(githubProvider, repoAllowlist)

			got, err := uc.Authenticate(ctx, tt.args.token)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Authenticate() error = nil, wantErr %v", tt.wantErr)
				}
				if errors.Is(tt.wantErr, usecase.ErrInvalidRepository) {
					if !errors.Is(err, usecase.ErrInvalidRepository) {
						t.Errorf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
					}
				} else {
					if err.Error() != tt.wantErr.Error() {
						t.Errorf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("Authenticate() unexpected error: %v", err)
				}
				if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(domain.UserInfo{}, domain.ProviderType{}, domain.RepositoryIdentifier{}, domain.RepositoryPermissions{})); diff != "" {
					t.Errorf("Authenticate() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func mustNewUserInfoWithFullPermissions(t *testing.T, sub, email, name string, provider domain.ProviderType, repository *domain.RepositoryIdentifier, ref string) *domain.UserInfo {
	t.Helper()
	userInfo, err := domain.NewUserInfo(sub, email, name, provider, repository, ref)
	if err != nil {
		t.Fatalf("mustNewUserInfoWithFullPermissions: %v", err)
	}
	fullPerms := domain.NewRepositoryPermissions(true, true, true, true, true)
	userInfo.SetPermissions(&fullPerms)
	return userInfo
}
