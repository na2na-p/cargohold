package oidc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewGitHubTokenExchanger(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		redirectURI  string
		wantErr      bool
	}{
		{
			name:         "正常系: 全パラメータが正しい場合、Exchangerが作成される",
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
			name:         "正常系: redirectURIが空の場合も動的設定可能でExchangerが作成される",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			redirectURI:  "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exchanger, err := NewGitHubTokenExchanger(tt.clientID, tt.clientSecret, tt.redirectURI)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("エラーが期待されましたが、nilが返りました")
				}
				return
			}

			if err != nil {
				t.Fatalf("エラーは期待されていませんでしたが、%v が返りました", err)
			}

			if exchanger == nil {
				t.Fatalf("Exchangerがnilです")
			}
		})
	}
}

func TestGitHubTokenExchanger_GetAuthorizationURL(t *testing.T) {
	tests := []struct {
		name           string
		state          string
		scopes         []string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:   "正常系: 基本的なスコープでURLが生成される",
			state:  "test-state-123",
			scopes: []string{"read:user", "repo"},
			wantContains: []string{
				"https://github.com/login/oauth/authorize",
				"client_id=test-client-id",
				"redirect_uri=",
				"state=test-state-123",
				"scope=read%3Auser+repo",
			},
			wantNotContain: []string{},
		},
		{
			name:   "正常系: スコープが空の場合もURLが生成される",
			state:  "test-state-456",
			scopes: []string{},
			wantContains: []string{
				"https://github.com/login/oauth/authorize",
				"client_id=test-client-id",
				"state=test-state-456",
			},
			wantNotContain: []string{
				"scope=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exchanger, err := NewGitHubTokenExchanger("test-client-id", "test-client-secret", "http://localhost:8080/callback")
			if err != nil {
				t.Fatalf("Exchanger作成に失敗: %v", err)
			}

			authURL := exchanger.GetAuthorizationURL(tt.state, tt.scopes)

			for _, want := range tt.wantContains {
				if !containsString(authURL, want) {
					t.Errorf("URL に %q が含まれていません: %s", want, authURL)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if containsString(authURL, notWant) {
					t.Errorf("URL に %q が含まれるべきではありません: %s", notWant, authURL)
				}
			}
		})
	}
}

func TestGitHubTokenExchanger_ExchangeCode(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		want           *oauthToken
		wantErr        bool
		wantErrMsg     string
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
		{
			name: "異常系: レスポンスが大きすぎる場合、エラーが返る",
			code: "test-code",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				largeData := make([]byte, 2<<20)
				for i := range largeData {
					largeData[i] = 'a'
				}
				_, _ = w.Write(largeData)
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: "レスポンスが大きすぎます",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			exchanger, err := NewGitHubTokenExchanger("test-client-id", "test-client-secret", "http://localhost:8080/callback")
			if err != nil {
				t.Fatalf("Exchanger作成に失敗: %v", err)
			}

			exchanger.SetTokenEndpoint(server.URL)

			ctx := context.Background()
			got, err := exchanger.ExchangeCode(ctx, tt.code)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("エラーが期待されましたが、nilが返りました")
				}
				if tt.wantErrMsg != "" && !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("エラーメッセージに %q が含まれていません: %v", tt.wantErrMsg, err)
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
