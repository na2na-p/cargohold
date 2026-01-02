package oidc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewGitHubOAuthProvider(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		redirectURI  string
		wantErr      bool
	}{
		{
			name:         "正常系: 全パラメータが正しい場合、プロバイダーが作成される",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			redirectURI:  "http://localhost:8080/callback",
			wantErr:      false,
		},
		{
			name:         "異常系: clientIDが空の場合、エラーが返る",
			clientID:     "",
			clientSecret: "test-client-secret",
			redirectURI:  "http://localhost:8080/callback",
			wantErr:      true,
		},
		{
			name:         "異常系: clientSecretが空の場合、エラーが返る",
			clientID:     "test-client-id",
			clientSecret: "",
			redirectURI:  "http://localhost:8080/callback",
			wantErr:      true,
		},
		{
			name:         "正常系: redirectURIが空の場合も動的設定可能でプロバイダーが作成される",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			redirectURI:  "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGitHubOAuthProvider(tt.clientID, tt.clientSecret, tt.redirectURI)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("エラーが期待されましたが、nilが返りました")
				}
				return
			}

			if err != nil {
				t.Fatalf("エラーは期待されていませんでしたが、%v が返りました", err)
			}

			if provider == nil {
				t.Fatalf("プロバイダーがnilです")
			}
		})
	}
}

func TestGitHubOAuthProvider_GetAuthorizationURL(t *testing.T) {
	tests := []struct {
		name         string
		state        string
		wantContains []string
	}{
		{
			name:  "正常系: stateを指定してURLが生成される",
			state: "test-state-123",
			wantContains: []string{
				"https://github.com/login/oauth/authorize",
				"client_id=test-client-id",
				"redirect_uri=",
				"state=test-state-123",
			},
		},
		{
			name:  "正常系: 別のstateでもURLが生成される",
			state: "test-state-456",
			wantContains: []string{
				"https://github.com/login/oauth/authorize",
				"client_id=test-client-id",
				"state=test-state-456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGitHubOAuthProvider("test-client-id", "test-client-secret", "http://localhost:8080/callback")
			if err != nil {
				t.Fatalf("プロバイダー作成に失敗: %v", err)
			}

			authURL := provider.GetAuthorizationURL(tt.state)

			for _, want := range tt.wantContains {
				if !containsString(authURL, want) {
					t.Errorf("URL に %q が含まれていません: %s", want, authURL)
				}
			}
		})
	}
}

func TestGitHubOAuthProvider_ExchangeCode(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		want           *oauthToken
		wantErr        bool
	}{
		{
			name: "正常系: 有効なコードでトークンが取得できる",
			code: "valid-code",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("期待されるメソッド: POST, 実際: %s", r.Method)
				}
				if r.Header.Get("Accept") != "application/json" {
					t.Errorf("期待されるAcceptヘッダー: application/json, 実際: %s", r.Header.Get("Accept"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"access_token": "gho_test_token_123",
					"token_type":   "bearer",
					"scope":        "read:user,repo",
				})
			},
			want: &oauthToken{
				AccessToken: "gho_test_token_123",
				TokenType:   "bearer",
				Scope:       "read:user,repo",
			},
			wantErr: false,
		},
		{
			name: "異常系: 無効なコードでエラーが返る",
			code: "invalid-code",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":             "bad_verification_code",
					"error_description": "The code passed is incorrect or expired.",
				})
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "異常系: サーバーエラーの場合、エラーが返る",
			code: "test-code",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			provider, err := NewGitHubOAuthProvider("test-client-id", "test-client-secret", "http://localhost:8080/callback")
			if err != nil {
				t.Fatalf("プロバイダー作成に失敗: %v", err)
			}

			provider.SetTokenEndpoint(server.URL)

			ctx := context.Background()
			got, err := provider.ExchangeCode(ctx, tt.code)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("エラーが期待されましたが、nilが返りました")
				}
				return
			}

			if err != nil {
				t.Fatalf("エラーは期待されていませんでしたが、%v が返りました", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("トークンが一致しません (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGitHubOAuthProvider_GetUserInfo(t *testing.T) {
	tests := []struct {
		name           string
		token          *oauthToken
		serverResponse func(w http.ResponseWriter, r *http.Request)
		want           *gitHubUser
		wantErr        bool
	}{
		{
			name: "正常系: 有効なトークンでユーザー情報が取得できる",
			token: &oauthToken{
				AccessToken: "gho_valid_token",
				TokenType:   "bearer",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("期待されるメソッド: GET, 実際: %s", r.Method)
				}
				authHeader := r.Header.Get("Authorization")
				if authHeader != "Bearer gho_valid_token" {
					t.Errorf("期待されるAuthorizationヘッダー: Bearer gho_valid_token, 実際: %s", authHeader)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":    int64(12345),
					"login": "testuser",
					"name":  "Test User",
				})
			},
			want: &gitHubUser{
				ID:    12345,
				Login: "testuser",
				Name:  "Test User",
			},
			wantErr: false,
		},
		{
			name: "異常系: 無効なトークンでエラーが返る",
			token: &oauthToken{
				AccessToken: "invalid_token",
				TokenType:   "bearer",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"message": "Bad credentials",
				})
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:  "異常系: トークンがnilの場合、エラーが返る",
			token: nil,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// このレスポンスは呼ばれないはず
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "異常系: AccessTokenが空の場合、エラーが返る",
			token: &oauthToken{
				AccessToken: "",
				TokenType:   "bearer",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// このレスポンスは呼ばれないはず
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			provider, err := NewGitHubOAuthProvider("test-client-id", "test-client-secret", "http://localhost:8080/callback")
			if err != nil {
				t.Fatalf("プロバイダー作成に失敗: %v", err)
			}

			provider.SetUserInfoEndpoint(server.URL)

			ctx := context.Background()
			got, err := provider.GetUserInfo(ctx, tt.token)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("エラーが期待されましたが、nilが返りました")
				}
				return
			}

			if err != nil {
				t.Fatalf("エラーは期待されていませんでしたが、%v が返りました", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ユーザー情報が一致しません (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGitHubOAuthProvider_CanAccessRepository(t *testing.T) {
	tests := []struct {
		name           string
		token          *oauthToken
		repo           *domain.RepositoryIdentifier
		serverResponse func(w http.ResponseWriter, r *http.Request)
		want           bool
		wantErr        bool
	}{
		{
			name: "正常系: アクセス可能なリポジトリの場合、trueが返る",
			token: &oauthToken{
				AccessToken: "gho_valid_token",
				TokenType:   "bearer",
			},
			repo: mustNewRepositoryIdentifier(t, "owner/repo"),
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/repos/owner/repo" {
					t.Errorf("期待されるパス: /repos/owner/repo, 実際: %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":        1,
					"full_name": "owner/repo",
					"private":   false,
				})
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "正常系: アクセス不可能なリポジトリの場合、falseが返る",
			token: &oauthToken{
				AccessToken: "gho_valid_token",
				TokenType:   "bearer",
			},
			repo: mustNewRepositoryIdentifier(t, "owner/private-repo"),
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"message": "Not Found",
				})
			},
			want:    false,
			wantErr: false,
		},
		{
			name:  "異常系: トークンがnilの場合、エラーが返る",
			token: nil,
			repo:  mustNewRepositoryIdentifier(t, "owner/repo"),
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// このレスポンスは呼ばれないはず
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "異常系: リポジトリがnilの場合、エラーが返る",
			token: &oauthToken{
				AccessToken: "gho_valid_token",
				TokenType:   "bearer",
			},
			repo: nil,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// このレスポンスは呼ばれないはず
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "異常系: AccessTokenが空の場合、エラーが返る",
			token: &oauthToken{
				AccessToken: "",
				TokenType:   "bearer",
			},
			repo: mustNewRepositoryIdentifier(t, "owner/repo"),
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// このレスポンスは呼ばれないはず
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "異常系: サーバーエラーの場合、エラーが返る",
			token: &oauthToken{
				AccessToken: "gho_valid_token",
				TokenType:   "bearer",
			},
			repo: mustNewRepositoryIdentifier(t, "owner/repo"),
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			provider, err := NewGitHubOAuthProvider("test-client-id", "test-client-secret", "http://localhost:8080/callback")
			if err != nil {
				t.Fatalf("プロバイダー作成に失敗: %v", err)
			}

			provider.SetAPIEndpoint(server.URL)

			ctx := context.Background()
			got, err := provider.CanAccessRepository(ctx, tt.token, tt.repo)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("エラーが期待されましたが、nilが返りました")
				}
				return
			}

			if err != nil {
				t.Fatalf("エラーは期待されていませんでしたが、%v が返りました", err)
			}

			if got != tt.want {
				t.Errorf("結果が一致しません: want=%v, got=%v", tt.want, got)
			}
		})
	}
}

// containsString はsがsubstrを含むかチェックするヘルパー関数
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mustNewRepositoryIdentifier はテスト用のRepositoryIdentifierを作成するヘルパー関数
func mustNewRepositoryIdentifier(t *testing.T, fullName string) *domain.RepositoryIdentifier {
	t.Helper()
	repo, err := domain.NewRepositoryIdentifier(fullName)
	if err != nil {
		t.Fatalf("RepositoryIdentifierの作成に失敗: %v", err)
	}
	return repo
}

// TestOAuthToken_Fields はOAuthTokenの各フィールドが正しく設定されているかをテストする
func TestOAuthToken_Fields(t *testing.T) {
	token := &oauthToken{
		AccessToken: "test-access-token",
		TokenType:   "bearer",
		Scope:       "read:user",
	}

	if token.AccessToken != "test-access-token" {
		t.Errorf("AccessToken が一致しません: want=test-access-token, got=%s", token.AccessToken)
	}
	if token.TokenType != "bearer" {
		t.Errorf("TokenType が一致しません: want=bearer, got=%s", token.TokenType)
	}
	if token.Scope != "read:user" {
		t.Errorf("Scope が一致しません: want=read:user, got=%s", token.Scope)
	}
}

// TestGitHubUser_Fields はGitHubUserの各フィールドが正しく設定されているかをテストする
func TestGitHubUser_Fields(t *testing.T) {
	user := &gitHubUser{
		ID:    12345,
		Login: "testuser",
		Name:  "Test User",
	}

	if user.ID != 12345 {
		t.Errorf("ID が一致しません: want=12345, got=%d", user.ID)
	}
	if user.Login != "testuser" {
		t.Errorf("Login が一致しません: want=testuser, got=%s", user.Login)
	}
	if user.Name != "Test User" {
		t.Errorf("Name が一致しません: want=Test User, got=%s", user.Name)
	}
}
