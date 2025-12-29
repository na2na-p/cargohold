package oidc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
	"go.uber.org/mock/gomock"
)

func TestNewGitHubOAuthProviderAdapter(t *testing.T) {
	tests := []struct {
		name     string
		provider GitHubOAuthProviderInternal
		wantNil  bool
	}{
		{
			name:     "正常系: プロバイダーを渡すとアダプターが作成される",
			provider: &mockInternalProvider{},
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewGitHubOAuthProviderAdapter(tt.provider)
			if tt.wantNil && adapter != nil {
				t.Errorf("nilが期待されましたが、アダプターが返りました")
			}
			if !tt.wantNil && adapter == nil {
				t.Errorf("アダプターが期待されましたが、nilが返りました")
			}
		})
	}
}

func TestNewGitHubOAuthProviderAdapter_PanicOnNilProvider(t *testing.T) {
	tests := []struct {
		name          string
		provider      GitHubOAuthProviderInternal
		wantPanic     bool
		panicContains string
	}{
		{
			name:          "異常系: nilを渡すとパニックする",
			provider:      nil,
			wantPanic:     true,
			panicContains: "NewGitHubOAuthProviderAdapter: provider is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.wantPanic {
					if r == nil {
						t.Errorf("パニックが期待されましたが、発生しませんでした")
						return
					}
					panicMsg, ok := r.(string)
					if !ok {
						t.Errorf("パニックメッセージが文字列ではありません: %v", r)
						return
					}
					if panicMsg != tt.panicContains {
						t.Errorf("パニックメッセージが一致しません: want=%q, got=%q", tt.panicContains, panicMsg)
					}
				} else {
					if r != nil {
						t.Errorf("パニックは期待されていませんでしたが、発生しました: %v", r)
					}
				}
			}()

			NewGitHubOAuthProviderAdapter(tt.provider)
		})
	}
}

func TestGitHubOAuthProviderAdapter_GetAuthorizationURL(t *testing.T) {
	type fields struct {
		setupMock func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal
	}
	type args struct {
		state  string
		scopes []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "正常系: 内部プロバイダーにそのまま委譲される",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal {
					mock := NewMockGitHubOAuthProviderInternal(ctrl)
					mock.EXPECT().GetAuthorizationURL("test-state", []string{"read:user", "repo"}).Return("https://github.com/login/oauth/authorize?state=test-state")
					return mock
				},
			},
			args: args{
				state:  "test-state",
				scopes: []string{"read:user", "repo"},
			},
			want: "https://github.com/login/oauth/authorize?state=test-state",
		},
		{
			name: "正常系: スコープが空の場合も正しく委譲される",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal {
					mock := NewMockGitHubOAuthProviderInternal(ctrl)
					mock.EXPECT().GetAuthorizationURL("state-empty-scope", []string{}).Return("https://github.com/login/oauth/authorize?state=state-empty-scope")
					return mock
				},
			},
			args: args{
				state:  "state-empty-scope",
				scopes: []string{},
			},
			want: "https://github.com/login/oauth/authorize?state=state-empty-scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := tt.fields.setupMock(ctrl)
			adapter := NewGitHubOAuthProviderAdapter(mockProvider)

			got := adapter.GetAuthorizationURL(tt.args.state, tt.args.scopes)

			if got != tt.want {
				t.Errorf("GetAuthorizationURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitHubOAuthProviderAdapter_ExchangeCode(t *testing.T) {
	type fields struct {
		setupMock func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal
	}
	type args struct {
		ctx  context.Context
		code string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *usecase.OAuthTokenResult
		wantErr error
	}{
		{
			name: "正常系: 内部プロバイダーの結果がOAuthTokenResultに変換される",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal {
					mock := NewMockGitHubOAuthProviderInternal(ctrl)
					mock.EXPECT().ExchangeCode(gomock.Any(), "valid-code").Return(&oauthToken{
						AccessToken: "gho_test_token",
						TokenType:   "bearer",
						Scope:       "read:user,repo",
					}, nil)
					return mock
				},
			},
			args: args{
				ctx:  context.Background(),
				code: "valid-code",
			},
			want: &usecase.OAuthTokenResult{
				AccessToken: "gho_test_token",
				TokenType:   "bearer",
				Scope:       "read:user,repo",
			},
			wantErr: nil,
		},
		{
			name: "異常系: 内部プロバイダーがエラーを返す場合、エラーがそのまま返される",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal {
					mock := NewMockGitHubOAuthProviderInternal(ctrl)
					mock.EXPECT().ExchangeCode(gomock.Any(), "invalid-code").Return(nil, errors.New("invalid code"))
					return mock
				},
			},
			args: args{
				ctx:  context.Background(),
				code: "invalid-code",
			},
			want:    nil,
			wantErr: errors.New("invalid code"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := tt.fields.setupMock(ctrl)
			adapter := NewGitHubOAuthProviderAdapter(mockProvider)

			got, err := adapter.ExchangeCode(tt.args.ctx, tt.args.code)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("エラーが期待されましたが、nilが返りました")
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("エラーメッセージが一致しません: want=%v, got=%v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("エラーは期待されていませんでしたが、%v が返りました", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ExchangeCode() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGitHubOAuthProviderAdapter_GetUserInfo(t *testing.T) {
	type fields struct {
		setupMock func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal
	}
	type args struct {
		ctx   context.Context
		token *usecase.OAuthTokenResult
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *usecase.GitHubUserResult
		wantErr error
	}{
		{
			name: "正常系: OAuthTokenResultがOAuthTokenに変換されて呼び出され、GitHubUserがGitHubUserResultに変換される",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal {
					mock := NewMockGitHubOAuthProviderInternal(ctrl)
					expectedToken := &oauthToken{
						AccessToken: "gho_test_token",
						TokenType:   "bearer",
						Scope:       "read:user",
					}
					mock.EXPECT().GetUserInfo(gomock.Any(), expectedToken).Return(&gitHubUser{
						ID:    12345,
						Login: "testuser",
						Name:  "Test User",
					}, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				token: &usecase.OAuthTokenResult{
					AccessToken: "gho_test_token",
					TokenType:   "bearer",
					Scope:       "read:user",
				},
			},
			want: &usecase.GitHubUserResult{
				ID:    12345,
				Login: "testuser",
				Name:  "Test User",
			},
			wantErr: nil,
		},
		{
			name: "異常系: 内部プロバイダーがエラーを返す場合、エラーがそのまま返される",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal {
					mock := NewMockGitHubOAuthProviderInternal(ctrl)
					mock.EXPECT().GetUserInfo(gomock.Any(), gomock.Any()).Return(nil, errors.New("unauthorized"))
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				token: &usecase.OAuthTokenResult{
					AccessToken: "invalid_token",
					TokenType:   "bearer",
				},
			},
			want:    nil,
			wantErr: errors.New("unauthorized"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := tt.fields.setupMock(ctrl)
			adapter := NewGitHubOAuthProviderAdapter(mockProvider)

			got, err := adapter.GetUserInfo(tt.args.ctx, tt.args.token)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("エラーが期待されましたが、nilが返りました")
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("エラーメッセージが一致しません: want=%v, got=%v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("エラーは期待されていませんでしたが、%v が返りました", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("GetUserInfo() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGitHubOAuthProviderAdapter_CanAccessRepository(t *testing.T) {
	type fields struct {
		setupMock func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal
	}
	type args struct {
		ctx   context.Context
		token *usecase.OAuthTokenResult
		repo  *domain.RepositoryIdentifier
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr error
	}{
		{
			name: "正常系: アクセス可能な場合trueが返される",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal {
					mock := NewMockGitHubOAuthProviderInternal(ctrl)
					expectedToken := &oauthToken{
						AccessToken: "gho_test_token",
						TokenType:   "bearer",
						Scope:       "repo",
					}
					mock.EXPECT().CanAccessRepository(gomock.Any(), expectedToken, gomock.Any()).Return(true, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				token: &usecase.OAuthTokenResult{
					AccessToken: "gho_test_token",
					TokenType:   "bearer",
					Scope:       "repo",
				},
				repo: mustCreateRepositoryIdentifier(t, "owner/repo"),
			},
			want:    true,
			wantErr: nil,
		},
		{
			name: "正常系: アクセス不可能な場合falseが返される",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal {
					mock := NewMockGitHubOAuthProviderInternal(ctrl)
					mock.EXPECT().CanAccessRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				token: &usecase.OAuthTokenResult{
					AccessToken: "gho_test_token",
					TokenType:   "bearer",
				},
				repo: mustCreateRepositoryIdentifier(t, "owner/private-repo"),
			},
			want:    false,
			wantErr: nil,
		},
		{
			name: "異常系: 内部プロバイダーがエラーを返す場合、エラーがそのまま返される",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *MockGitHubOAuthProviderInternal {
					mock := NewMockGitHubOAuthProviderInternal(ctrl)
					mock.EXPECT().CanAccessRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, errors.New("server error"))
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				token: &usecase.OAuthTokenResult{
					AccessToken: "gho_test_token",
					TokenType:   "bearer",
				},
				repo: mustCreateRepositoryIdentifier(t, "owner/repo"),
			},
			want:    false,
			wantErr: errors.New("server error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := tt.fields.setupMock(ctrl)
			adapter := NewGitHubOAuthProviderAdapter(mockProvider)

			got, err := adapter.CanAccessRepository(tt.args.ctx, tt.args.token, tt.args.repo)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("エラーが期待されましたが、nilが返りました")
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("エラーメッセージが一致しません: want=%v, got=%v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("エラーは期待されていませんでしたが、%v が返りました", err)
			}

			if got != tt.want {
				t.Errorf("CanAccessRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitHubOAuthProviderAdapter_ImplementsInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := NewMockGitHubOAuthProviderInternal(ctrl)
	adapter := NewGitHubOAuthProviderAdapter(mockProvider)

	var _ usecase.GitHubOAuthProviderInterface = adapter
}

type mockInternalProvider struct{}

func (m *mockInternalProvider) GetAuthorizationURL(state string, scopes []string) string {
	return ""
}

func (m *mockInternalProvider) ExchangeCode(ctx context.Context, code string) (*oauthToken, error) {
	return nil, nil
}

func (m *mockInternalProvider) GetUserInfo(ctx context.Context, token *oauthToken) (*gitHubUser, error) {
	return nil, nil
}

func (m *mockInternalProvider) CanAccessRepository(ctx context.Context, token *oauthToken, repo *domain.RepositoryIdentifier) (bool, error) {
	return false, nil
}

func (m *mockInternalProvider) SetRedirectURI(redirectURI string) {
}

func mustCreateRepositoryIdentifier(t *testing.T, fullName string) *domain.RepositoryIdentifier {
	t.Helper()
	repo, err := domain.NewRepositoryIdentifier(fullName)
	if err != nil {
		t.Fatalf("RepositoryIdentifierの作成に失敗: %v", err)
	}
	return repo
}
