package oidc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/infrastructure/oidc"
)

func TestNewGitHubRepositoryChecker(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "正常系: Checkerが作成される",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := oidc.NewGitHubRepositoryChecker()

			if checker == nil {
				t.Fatalf("Checkerがnilです")
			}
		})
	}
}

func TestGitHubRepositoryChecker_CanAccessRepository(t *testing.T) {
	tests := []struct {
		name           string
		token          *oidc.OAuthToken
		repo           *domain.RepositoryIdentifier
		serverResponse func(w http.ResponseWriter, r *http.Request)
		want           bool
		wantErr        bool
	}{
		{
			name: "正常系: アクセス可能なリポジトリの場合、trueが返る",
			token: &oidc.OAuthToken{
				AccessToken: "gho_valid_token",
				TokenType:   "bearer",
			},
			repo: mustNewRepositoryIdentifierForChecker(t, "owner/repo"),
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
			token: &oidc.OAuthToken{
				AccessToken: "gho_valid_token",
				TokenType:   "bearer",
			},
			repo: mustNewRepositoryIdentifierForChecker(t, "owner/private-repo"),
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
			repo:  mustNewRepositoryIdentifierForChecker(t, "owner/repo"),
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "異常系: リポジトリがnilの場合、エラーが返る",
			token: &oidc.OAuthToken{
				AccessToken: "gho_valid_token",
				TokenType:   "bearer",
			},
			repo: nil,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "異常系: AccessTokenが空の場合、エラーが返る",
			token: &oidc.OAuthToken{
				AccessToken: "",
				TokenType:   "bearer",
			},
			repo: mustNewRepositoryIdentifierForChecker(t, "owner/repo"),
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "異常系: サーバーエラーの場合、エラーが返る",
			token: &oidc.OAuthToken{
				AccessToken: "gho_valid_token",
				TokenType:   "bearer",
			},
			repo: mustNewRepositoryIdentifierForChecker(t, "owner/repo"),
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

			checker := oidc.NewGitHubRepositoryChecker()
			checker.SetAPIEndpoint(server.URL)

			ctx := context.Background()
			got, err := checker.CanAccessRepository(ctx, tt.token, tt.repo)

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

func mustNewRepositoryIdentifierForChecker(t *testing.T, fullName string) *domain.RepositoryIdentifier {
	t.Helper()
	repo, err := domain.NewRepositoryIdentifier(fullName)
	if err != nil {
		t.Fatalf("RepositoryIdentifierの作成に失敗: %v", err)
	}
	return repo
}
