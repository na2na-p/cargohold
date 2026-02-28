package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_usecase "github.com/na2na-p/cargohold/tests/usecase"
	"go.uber.org/mock/gomock"
)

func mustNewAllowedRedirectURIs(t *testing.T, uris []string) *domain.AllowedRedirectURIs {
	t.Helper()
	allowedURIs, err := domain.NewAllowedRedirectURIs(uris)
	if err != nil {
		t.Fatalf("failed to create AllowedRedirectURIs: %v", err)
	}
	return allowedURIs
}

func TestGitHubOAuthUseCase_StartAuthentication(t *testing.T) {
	type args struct {
		repository  *domain.RepositoryIdentifier
		redirectURI string
	}
	tests := []struct {
		name                string
		args                args
		allowedRedirectURIs *domain.AllowedRedirectURIs
		setupMocks          func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface)
		wantErr             error
		wantURL             bool
	}{
		{
			name: "正常系: 認証URLを生成し返却する",
			args: args{
				repository: func() *domain.RepositoryIdentifier {
					r, _ := domain.NewRepositoryIdentifier("owner/repo")
					return r
				}(),
				redirectURI: "https://example.com/callback",
			},
			allowedRedirectURIs: func() *domain.AllowedRedirectURIs {
				a, _ := domain.NewAllowedRedirectURIs([]string{"https://example.com/callback"})
				return a
			}(),
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)

				stateStore.EXPECT().SaveState(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

				oauthProvider.EXPECT().SetRedirectURI(gomock.Any())
				oauthProvider.EXPECT().GetAuthorizationURL(gomock.Any()).Return("https://github.com/login/oauth/authorize?state=xxx")

				return oauthProvider, sessionStore, stateStore
			},
			wantErr: nil,
			wantURL: true,
		},
		{
			name: "異常系: repositoryがnilの場合エラー",
			args: args{
				repository:  nil,
				redirectURI: "https://example.com/callback",
			},
			allowedRedirectURIs: func() *domain.AllowedRedirectURIs {
				a, _ := domain.NewAllowedRedirectURIs([]string{"https://example.com/callback"})
				return a
			}(),
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)
				return oauthProvider, sessionStore, stateStore
			},
			wantErr: usecase.ErrInvalidRepository,
			wantURL: false,
		},
		{
			name: "異常系: redirectURIが空の場合エラー",
			args: args{
				repository: func() *domain.RepositoryIdentifier {
					r, _ := domain.NewRepositoryIdentifier("owner/repo")
					return r
				}(),
				redirectURI: "",
			},
			allowedRedirectURIs: func() *domain.AllowedRedirectURIs {
				a, _ := domain.NewAllowedRedirectURIs([]string{"https://example.com/callback"})
				return a
			}(),
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)
				return oauthProvider, sessionStore, stateStore
			},
			wantErr: usecase.ErrInvalidRedirectURI,
			wantURL: false,
		},
		{
			name: "異常系: redirectURIがホワイトリストにない場合エラー",
			args: args{
				repository: func() *domain.RepositoryIdentifier {
					r, _ := domain.NewRepositoryIdentifier("owner/repo")
					return r
				}(),
				redirectURI: "https://malicious.com/callback",
			},
			allowedRedirectURIs: func() *domain.AllowedRedirectURIs {
				a, _ := domain.NewAllowedRedirectURIs([]string{"https://example.com/callback"})
				return a
			}(),
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)
				return oauthProvider, sessionStore, stateStore
			},
			wantErr: usecase.ErrInvalidRedirectURI,
			wantURL: false,
		},
		{
			name: "正常系: 複数のホワイトリストURIから一致",
			args: args{
				repository: func() *domain.RepositoryIdentifier {
					r, _ := domain.NewRepositoryIdentifier("owner/repo")
					return r
				}(),
				redirectURI: "https://example2.com/callback",
			},
			allowedRedirectURIs: func() *domain.AllowedRedirectURIs {
				a, _ := domain.NewAllowedRedirectURIs([]string{"https://example.com/callback", "https://example2.com/callback"})
				return a
			}(),
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)

				stateStore.EXPECT().SaveState(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

				oauthProvider.EXPECT().SetRedirectURI(gomock.Any())
				oauthProvider.EXPECT().GetAuthorizationURL(gomock.Any()).Return("https://github.com/login/oauth/authorize?state=xxx")

				return oauthProvider, sessionStore, stateStore
			},
			wantErr: nil,
			wantURL: true,
		},
		{
			name: "異常系: state保存に失敗した場合エラー",
			args: args{
				repository: func() *domain.RepositoryIdentifier {
					r, _ := domain.NewRepositoryIdentifier("owner/repo")
					return r
				}(),
				redirectURI: "https://example.com/callback",
			},
			allowedRedirectURIs: func() *domain.AllowedRedirectURIs {
				a, _ := domain.NewAllowedRedirectURIs([]string{"https://example.com/callback"})
				return a
			}(),
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)

				stateStore.EXPECT().SaveState(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("Redis保存エラー"))

				return oauthProvider, sessionStore, stateStore
			},
			wantErr: usecase.ErrStateSaveFailed,
			wantURL: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			oauthProvider, sessionStore, stateStore := tt.setupMocks(ctrl)
			uc, err := usecase.NewGitHubOAuthUseCase(oauthProvider, sessionStore, stateStore, tt.allowedRedirectURIs)
			if err != nil {
				t.Fatalf("NewGitHubOAuthUseCase() unexpected error: %v", err)
			}

			authURL, err := uc.StartAuthentication(ctx, tt.args.repository, tt.args.redirectURI, "")

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("StartAuthentication() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("StartAuthentication() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("StartAuthentication() unexpected error: %v", err)
				}
				if tt.wantURL && authURL == "" {
					t.Errorf("StartAuthentication() authURL is empty")
				}
			}
		})
	}
}

func TestGitHubOAuthUseCase_HandleCallback(t *testing.T) {
	type args struct {
		code  string
		state string
	}
	tests := []struct {
		name          string
		args          args
		setupMocks    func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface)
		wantErr       error
		wantSessionID bool
	}{
		{
			name: "正常系: コールバックを処理し、セッションIDを返す",
			args: args{
				code:  "valid-code",
				state: "valid-state",
			},
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)

				repo, _ := domain.NewRepositoryIdentifier("owner/repo")

				stateStore.EXPECT().GetAndDeleteState(gomock.Any(), "valid-state").Return(domain.NewOAuthState("owner/repo", "https://example.com/callback", ""), nil)

				oauthProvider.EXPECT().SetRedirectURI("https://example.com/callback")

				token := &usecase.OAuthTokenResult{
					AccessToken: "access-token",
					TokenType:   "Bearer",
					Scope:       "repo",
				}
				oauthProvider.EXPECT().ExchangeCode(gomock.Any(), "valid-code").Return(token, nil)

				oauthProvider.EXPECT().GetUserInfo(gomock.Any(), token).Return(&usecase.GitHubUserResult{
					ID:    12345,
					Login: "testuser",
					Name:  "Test User",
				}, nil)

				perms := domain.NewRepositoryPermissions(false, true, true, false, false)
				oauthProvider.EXPECT().GetRepositoryPermissions(gomock.Any(), token, repo).Return(perms, nil)

				sessionStore.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any()).Return("session-id-123", nil)

				return oauthProvider, sessionStore, stateStore
			},
			wantErr:       nil,
			wantSessionID: true,
		},
		{
			name: "異常系: codeが空の場合エラー",
			args: args{
				code:  "",
				state: "valid-state",
			},
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)
				return oauthProvider, sessionStore, stateStore
			},
			wantErr:       usecase.ErrInvalidCode,
			wantSessionID: false,
		},
		{
			name: "異常系: stateが空の場合エラー（storeを呼ばない）",
			args: args{
				code:  "valid-code",
				state: "",
			},
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)
				return oauthProvider, sessionStore, stateStore
			},
			wantErr:       usecase.ErrInvalidState,
			wantSessionID: false,
		},
		{
			name: "異常系: state検証に失敗した場合エラー",
			args: args{
				code:  "valid-code",
				state: "invalid-state",
			},
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)

				stateStore.EXPECT().GetAndDeleteState(gomock.Any(), "invalid-state").Return(nil, errors.New("state not found"))

				return oauthProvider, sessionStore, stateStore
			},
			wantErr:       usecase.ErrInvalidState,
			wantSessionID: false,
		},
		{
			name: "異常系: コード交換に失敗した場合エラー",
			args: args{
				code:  "invalid-code",
				state: "valid-state",
			},
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)

				stateStore.EXPECT().GetAndDeleteState(gomock.Any(), "valid-state").Return(domain.NewOAuthState("owner/repo", "https://example.com/callback", ""), nil)

				oauthProvider.EXPECT().SetRedirectURI("https://example.com/callback")

				oauthProvider.EXPECT().ExchangeCode(gomock.Any(), "invalid-code").Return(nil, errors.New("invalid code"))

				return oauthProvider, sessionStore, stateStore
			},
			wantErr:       usecase.ErrCodeExchangeFailed,
			wantSessionID: false,
		},
		{
			name: "異常系: ユーザー情報取得に失敗した場合エラー",
			args: args{
				code:  "valid-code",
				state: "valid-state",
			},
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)

				stateStore.EXPECT().GetAndDeleteState(gomock.Any(), "valid-state").Return(domain.NewOAuthState("owner/repo", "https://example.com/callback", ""), nil)

				oauthProvider.EXPECT().SetRedirectURI("https://example.com/callback")

				token := &usecase.OAuthTokenResult{
					AccessToken: "access-token",
					TokenType:   "Bearer",
				}
				oauthProvider.EXPECT().ExchangeCode(gomock.Any(), "valid-code").Return(token, nil)

				oauthProvider.EXPECT().GetUserInfo(gomock.Any(), token).Return(nil, errors.New("user info error"))

				return oauthProvider, sessionStore, stateStore
			},
			wantErr:       usecase.ErrUserInfoFailed,
			wantSessionID: false,
		},
		{
			name: "異常系: リポジトリアクセス権がない場合エラー",
			args: args{
				code:  "valid-code",
				state: "valid-state",
			},
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)

				repo, _ := domain.NewRepositoryIdentifier("owner/repo")

				stateStore.EXPECT().GetAndDeleteState(gomock.Any(), "valid-state").Return(domain.NewOAuthState("owner/repo", "https://example.com/callback", ""), nil)

				oauthProvider.EXPECT().SetRedirectURI("https://example.com/callback")

				token := &usecase.OAuthTokenResult{
					AccessToken: "access-token",
					TokenType:   "Bearer",
				}
				oauthProvider.EXPECT().ExchangeCode(gomock.Any(), "valid-code").Return(token, nil)

				oauthProvider.EXPECT().GetUserInfo(gomock.Any(), token).Return(&usecase.GitHubUserResult{
					ID:    12345,
					Login: "testuser",
					Name:  "Test User",
				}, nil)

				perms := domain.NewRepositoryPermissions(false, false, false, false, false)
				oauthProvider.EXPECT().GetRepositoryPermissions(gomock.Any(), token, repo).Return(perms, nil)

				return oauthProvider, sessionStore, stateStore
			},
			wantErr:       usecase.ErrRepositoryAccessDenied,
			wantSessionID: false,
		},
		{
			name: "異常系: リポジトリアクセス権検証でエラー発生",
			args: args{
				code:  "valid-code",
				state: "valid-state",
			},
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)

				repo, _ := domain.NewRepositoryIdentifier("owner/repo")

				stateStore.EXPECT().GetAndDeleteState(gomock.Any(), "valid-state").Return(domain.NewOAuthState("owner/repo", "https://example.com/callback", ""), nil)

				oauthProvider.EXPECT().SetRedirectURI("https://example.com/callback")

				token := &usecase.OAuthTokenResult{
					AccessToken: "access-token",
					TokenType:   "Bearer",
				}
				oauthProvider.EXPECT().ExchangeCode(gomock.Any(), "valid-code").Return(token, nil)

				oauthProvider.EXPECT().GetUserInfo(gomock.Any(), token).Return(&usecase.GitHubUserResult{
					ID:    12345,
					Login: "testuser",
					Name:  "Test User",
				}, nil)

				oauthProvider.EXPECT().GetRepositoryPermissions(gomock.Any(), token, repo).Return(domain.RepositoryPermissions{}, errors.New("API error"))

				return oauthProvider, sessionStore, stateStore
			},
			wantErr:       usecase.ErrRepositoryAccessCheckFailed,
			wantSessionID: false,
		},
		{
			name: "異常系: セッション作成に失敗した場合エラー",
			args: args{
				code:  "valid-code",
				state: "valid-state",
			},
			setupMocks: func(ctrl *gomock.Controller) (*mock_usecase.MockGitHubOAuthProviderInterface, *mock_usecase.MockSessionStoreInterface, *mock_usecase.MockOAuthStateStoreInterface) {
				oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
				sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
				stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)

				repo, _ := domain.NewRepositoryIdentifier("owner/repo")

				stateStore.EXPECT().GetAndDeleteState(gomock.Any(), "valid-state").Return(domain.NewOAuthState("owner/repo", "https://example.com/callback", ""), nil)

				oauthProvider.EXPECT().SetRedirectURI("https://example.com/callback")

				token := &usecase.OAuthTokenResult{
					AccessToken: "access-token",
					TokenType:   "Bearer",
				}
				oauthProvider.EXPECT().ExchangeCode(gomock.Any(), "valid-code").Return(token, nil)

				oauthProvider.EXPECT().GetUserInfo(gomock.Any(), token).Return(&usecase.GitHubUserResult{
					ID:    12345,
					Login: "testuser",
					Name:  "Test User",
				}, nil)

				perms := domain.NewRepositoryPermissions(false, true, true, false, false)
				oauthProvider.EXPECT().GetRepositoryPermissions(gomock.Any(), token, repo).Return(perms, nil)

				sessionStore.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any()).Return("", errors.New("Redis error"))

				return oauthProvider, sessionStore, stateStore
			},
			wantErr:       usecase.ErrSessionCreationFailed,
			wantSessionID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			oauthProvider, sessionStore, stateStore := tt.setupMocks(ctrl)
			allowedURIs := mustNewAllowedRedirectURIs(t, []string{"https://example.com/callback"})
			uc, err := usecase.NewGitHubOAuthUseCase(oauthProvider, sessionStore, stateStore, allowedURIs)
			if err != nil {
				t.Fatalf("NewGitHubOAuthUseCase() unexpected error: %v", err)
			}

			sessionID, _, err := uc.HandleCallback(ctx, tt.args.code, tt.args.state)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("HandleCallback() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("HandleCallback() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("HandleCallback() unexpected error: %v", err)
				}
				if tt.wantSessionID && sessionID == "" {
					t.Errorf("HandleCallback() sessionID is empty")
				}
			}
		})
	}
}

func TestGitHubOAuthUseCase_HandleCallback_UserInfoMapping(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	oauthProvider := mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl)
	sessionStore := mock_usecase.NewMockSessionStoreInterface(ctrl)
	stateStore := mock_usecase.NewMockOAuthStateStoreInterface(ctrl)

	repo, _ := domain.NewRepositoryIdentifier("owner/repo")

	stateStore.EXPECT().GetAndDeleteState(gomock.Any(), "valid-state").Return(domain.NewOAuthState("owner/repo", "https://example.com/callback", ""), nil)

	oauthProvider.EXPECT().SetRedirectURI("https://example.com/callback")

	token := &usecase.OAuthTokenResult{
		AccessToken: "access-token",
		TokenType:   "Bearer",
	}
	oauthProvider.EXPECT().ExchangeCode(gomock.Any(), "valid-code").Return(token, nil)

	oauthProvider.EXPECT().GetUserInfo(gomock.Any(), token).Return(&usecase.GitHubUserResult{
		ID:    12345,
		Login: "testuser",
		Name:  "Test User",
	}, nil)

	perms := domain.NewRepositoryPermissions(false, true, true, false, false)
	oauthProvider.EXPECT().GetRepositoryPermissions(gomock.Any(), token, repo).Return(perms, nil)

	var capturedUserInfo *domain.UserInfo
	sessionStore.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, userInfo *domain.UserInfo, _ interface{}) (string, error) {
			capturedUserInfo = userInfo
			return "session-id", nil
		},
	)

	allowedURIs := mustNewAllowedRedirectURIs(t, []string{"https://example.com/callback"})
	uc, err := usecase.NewGitHubOAuthUseCase(oauthProvider, sessionStore, stateStore, allowedURIs)
	if err != nil {
		t.Fatalf("NewGitHubOAuthUseCase() unexpected error: %v", err)
	}
	_, _, err = uc.HandleCallback(ctx, "valid-code", "valid-state")
	if err != nil {
		t.Fatalf("HandleCallback() unexpected error: %v", err)
	}

	ownerRepo, _ := domain.NewRepositoryIdentifier("owner/repo")
	wantUserInfo, _ := domain.NewUserInfo(
		"12345",
		"",
		"Test User",
		domain.ProviderTypeGitHub,
		ownerRepo,
		"",
	)
	expectedPerms := domain.NewRepositoryPermissions(false, true, true, false, false)
	wantUserInfo.SetPermissions(&expectedPerms)

	if diff := cmp.Diff(wantUserInfo, capturedUserInfo, cmp.AllowUnexported(domain.UserInfo{}, domain.ProviderType{}, domain.RepositoryIdentifier{}, domain.RepositoryPermissions{})); diff != "" {
		t.Errorf("UserInfo mismatch (-want +got):\n%s", diff)
	}
}

func TestNewGitHubOAuthUseCase(t *testing.T) {
	type fields struct {
		oauthProvider       usecase.GitHubOAuthProviderInterface
		sessionStore        usecase.SessionStoreInterface
		stateStore          usecase.OAuthStateStoreInterface
		allowedRedirectURIs *domain.AllowedRedirectURIs
	}
	tests := []struct {
		name        string
		setupFields func(ctrl *gomock.Controller) fields
		wantErr     bool
		errContains string
	}{
		{
			name: "正常系: 全ての依存性が設定されている場合成功",
			setupFields: func(ctrl *gomock.Controller) fields {
				allowedURIs, _ := domain.NewAllowedRedirectURIs([]string{"https://example.com/callback"})
				return fields{
					oauthProvider:       mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl),
					sessionStore:        mock_usecase.NewMockSessionStoreInterface(ctrl),
					stateStore:          mock_usecase.NewMockOAuthStateStoreInterface(ctrl),
					allowedRedirectURIs: allowedURIs,
				}
			},
			wantErr: false,
		},
		{
			name: "異常系: oauthProviderがnilの場合エラー",
			setupFields: func(ctrl *gomock.Controller) fields {
				allowedURIs, _ := domain.NewAllowedRedirectURIs([]string{"https://example.com/callback"})
				return fields{
					oauthProvider:       nil,
					sessionStore:        mock_usecase.NewMockSessionStoreInterface(ctrl),
					stateStore:          mock_usecase.NewMockOAuthStateStoreInterface(ctrl),
					allowedRedirectURIs: allowedURIs,
				}
			},
			wantErr:     true,
			errContains: "oauthProvider is nil",
		},
		{
			name: "異常系: sessionStoreがnilの場合エラー",
			setupFields: func(ctrl *gomock.Controller) fields {
				allowedURIs, _ := domain.NewAllowedRedirectURIs([]string{"https://example.com/callback"})
				return fields{
					oauthProvider:       mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl),
					sessionStore:        nil,
					stateStore:          mock_usecase.NewMockOAuthStateStoreInterface(ctrl),
					allowedRedirectURIs: allowedURIs,
				}
			},
			wantErr:     true,
			errContains: "sessionStore is nil",
		},
		{
			name: "異常系: stateStoreがnilの場合エラー",
			setupFields: func(ctrl *gomock.Controller) fields {
				allowedURIs, _ := domain.NewAllowedRedirectURIs([]string{"https://example.com/callback"})
				return fields{
					oauthProvider:       mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl),
					sessionStore:        mock_usecase.NewMockSessionStoreInterface(ctrl),
					stateStore:          nil,
					allowedRedirectURIs: allowedURIs,
				}
			},
			wantErr:     true,
			errContains: "stateStore is nil",
		},
		{
			name: "異常系: allowedRedirectURIsがnilの場合エラー",
			setupFields: func(ctrl *gomock.Controller) fields {
				return fields{
					oauthProvider:       mock_usecase.NewMockGitHubOAuthProviderInterface(ctrl),
					sessionStore:        mock_usecase.NewMockSessionStoreInterface(ctrl),
					stateStore:          mock_usecase.NewMockOAuthStateStoreInterface(ctrl),
					allowedRedirectURIs: nil,
				}
			},
			wantErr:     true,
			errContains: "allowedRedirectURIs is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := tt.setupFields(ctrl)
			uc, err := usecase.NewGitHubOAuthUseCase(f.oauthProvider, f.sessionStore, f.stateStore, f.allowedRedirectURIs)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewGitHubOAuthUseCase() error = nil, wantErr %v", tt.wantErr)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewGitHubOAuthUseCase() error = %v, want error containing %q", err, tt.errContains)
				}
				if uc != nil {
					t.Errorf("NewGitHubOAuthUseCase() returned non-nil UseCase on error")
				}
			} else {
				if err != nil {
					t.Fatalf("NewGitHubOAuthUseCase() unexpected error: %v", err)
				}
				if uc == nil {
					t.Errorf("NewGitHubOAuthUseCase() returned nil UseCase")
				}
			}
		})
	}
}
