//go:build e2e

package e2e

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestGitHubOAuthLogin_Redirect(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		repository string
	}
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
	}{
		{
			name: "正常系: GitHub OAuth URLにリダイレクトされる",
			args: args{
				repository: "na2na-p/na2na-platform",
			},
			wantStatusCode: http.StatusFound,
		},
		{
			name: "正常系: 別のリポジトリでもGitHub OAuth URLにリダイレクトされる",
			args: args{
				repository: "na2na-p/test-repo",
			},
			wantStatusCode: http.StatusFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
				Timeout: 30 * time.Second,
			}

			loginURL := GetOAuthLoginEndpoint(tt.args.repository)
			resp, err := client.Get(loginURL)
			if err != nil {
				t.Fatalf("リクエストの送信に失敗: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode = %d, want %d", resp.StatusCode, tt.wantStatusCode)
				return
			}

			location := resp.Header.Get("Location")
			if location == "" {
				t.Error("Locationヘッダーが設定されていません")
				return
			}

			redirectURL, err := url.Parse(location)
			if err != nil {
				t.Fatalf("リダイレクトURLのパースに失敗: %v", err)
			}

			if !strings.Contains(redirectURL.Host, "github.com") {
				t.Errorf("リダイレクト先がGitHubではありません: %s", redirectURL.Host)
			}

			if !strings.Contains(redirectURL.Path, "/login/oauth/authorize") {
				t.Errorf("リダイレクトパスがOAuth認証パスではありません: %s", redirectURL.Path)
			}

			query := redirectURL.Query()
			if query.Get("client_id") == "" {
				t.Error("client_idパラメータがありません")
			}
			if query.Get("redirect_uri") == "" {
				t.Error("redirect_uriパラメータがありません")
			}
			if query.Get("state") == "" {
				t.Error("stateパラメータがありません")
			}
		})
	}
}

func TestGitHubOAuthLogin_Redirect_ParameterValidation(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	tests := []struct {
		name           string
		url            string
		wantStatusCode int
	}{
		{
			name:           "異常系: repositoryパラメータが欠落している場合は400エラー",
			url:            GetBaseEndpoint() + "/auth/github/login",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "異常系: repositoryパラメータが空の場合は400エラー",
			url:            GetBaseEndpoint() + "/auth/github/login?repository=",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "異常系: repositoryパラメータの形式が不正な場合は400エラー",
			url:            GetBaseEndpoint() + "/auth/github/login?repository=invalid-format",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "異常系: repositoryパラメータでownerが空の場合は400エラー",
			url:            GetBaseEndpoint() + "/auth/github/login?repository=/repo",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "異常系: repositoryパラメータでrepoが空の場合は400エラー",
			url:            GetBaseEndpoint() + "/auth/github/login?repository=owner/",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
				Timeout: 30 * time.Second,
			}

			resp, err := client.Get(tt.url)
			if err != nil {
				t.Fatalf("リクエストの送信に失敗: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if diff := cmp.Diff(tt.wantStatusCode, resp.StatusCode); diff != "" {
				t.Errorf("StatusCode mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGitHubOAuthLogin_HostHeaderValidation(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	tests := []struct {
		name           string
		hostHeader     string
		wantStatusCode int
	}{
		{
			name:           "異常系: 許可されていないホストからのリクエストは400エラー",
			hostHeader:     "malicious.example.com",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
				Timeout: 30 * time.Second,
			}

			loginURL := GetOAuthLoginEndpoint("na2na-p/na2na-platform")
			req, err := http.NewRequest(http.MethodGet, loginURL, nil)
			if err != nil {
				t.Fatalf("リクエストの作成に失敗: %v", err)
			}

			req.Host = tt.hostHeader

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("リクエストの送信に失敗: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if diff := cmp.Diff(tt.wantStatusCode, resp.StatusCode); diff != "" {
				t.Errorf("StatusCode mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
