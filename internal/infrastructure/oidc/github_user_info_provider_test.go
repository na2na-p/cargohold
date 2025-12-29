package oidc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewGitHubUserInfoProvider(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "正常系: Providerが作成される",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewGitHubUserInfoProvider()

			if provider == nil {
				t.Fatalf("Providerがnilです")
			}
		})
	}
}

func TestGitHubUserInfoProvider_GetUserInfo(t *testing.T) {
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
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			provider := NewGitHubUserInfoProvider()
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
