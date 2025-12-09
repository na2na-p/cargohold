package auth_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/handler/auth"
	mockauth "github.com/na2na-p/cargohold/tests/handler/auth"
	"go.uber.org/mock/gomock"
)

func TestGitHubLoginHandler(t *testing.T) {
	type args struct {
		repository string
		host       string
	}
	tests := []struct {
		name           string
		args           args
		cfg            auth.GitHubLoginHandlerConfig
		setupMock      func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface
		expectedStatus int
		expectedURL    string
	}{
		{
			name: "正常系: GitHub認証URLへリダイレクトされる",
			args: args{
				repository: "owner/repo",
				host:       "example.com",
			},
			cfg: auth.GitHubLoginHandlerConfig{TrustProxy: false},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				m := mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
				m.EXPECT().
					StartAuthentication(gomock.Any(), gomock.Any(), gomock.Any()).
					Return("https://github.com/login/oauth/authorize?client_id=test&state=abc123", nil)
				return m
			},
			expectedStatus: http.StatusFound,
			expectedURL:    "https://github.com/login/oauth/authorize?client_id=test&state=abc123",
		},
		{
			name: "正常系: AllowedHostsが設定されていて許可ホストの場合はリダイレクトされる",
			args: args{
				repository: "owner/repo",
				host:       "example.com",
			},
			cfg: auth.GitHubLoginHandlerConfig{
				TrustProxy:   false,
				AllowedHosts: []string{"example.com", "api.example.com"},
			},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				m := mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
				m.EXPECT().
					StartAuthentication(gomock.Any(), gomock.Any(), gomock.Any()).
					Return("https://github.com/login/oauth/authorize?client_id=test&state=abc123", nil)
				return m
			},
			expectedStatus: http.StatusFound,
			expectedURL:    "https://github.com/login/oauth/authorize?client_id=test&state=abc123",
		},
		{
			name: "異常系: AllowedHostsが設定されていて許可されていないホストの場合はBadRequestを返す",
			args: args{
				repository: "owner/repo",
				host:       "malicious.com",
			},
			cfg: auth.GitHubLoginHandlerConfig{
				TrustProxy:   false,
				AllowedHosts: []string{"example.com", "api.example.com"},
			},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				return mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
			},
			expectedStatus: http.StatusBadRequest,
			expectedURL:    "",
		},
		{
			name: "異常系: repositoryパラメータが空の場合はBadRequestを返す",
			args: args{
				repository: "",
				host:       "example.com",
			},
			cfg: auth.GitHubLoginHandlerConfig{TrustProxy: false},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				return mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
			},
			expectedStatus: http.StatusBadRequest,
			expectedURL:    "",
		},
		{
			name: "異常系: repositoryパラメータの形式が不正な場合はBadRequestを返す",
			args: args{
				repository: "invalid-format",
				host:       "example.com",
			},
			cfg: auth.GitHubLoginHandlerConfig{TrustProxy: false},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				return mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
			},
			expectedStatus: http.StatusBadRequest,
			expectedURL:    "",
		},
		{
			name: "異常系: owner/repo形式だがownerが空の場合はBadRequestを返す",
			args: args{
				repository: "/repo",
				host:       "example.com",
			},
			cfg: auth.GitHubLoginHandlerConfig{TrustProxy: false},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				return mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
			},
			expectedStatus: http.StatusBadRequest,
			expectedURL:    "",
		},
		{
			name: "異常系: owner/repo形式だがrepoが空の場合はBadRequestを返す",
			args: args{
				repository: "owner/",
				host:       "example.com",
			},
			cfg: auth.GitHubLoginHandlerConfig{TrustProxy: false},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				return mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
			},
			expectedStatus: http.StatusBadRequest,
			expectedURL:    "",
		},
		{
			name: "異常系: UseCaseでエラーが発生した場合はInternalServerErrorを返す",
			args: args{
				repository: "owner/repo",
				host:       "example.com",
			},
			cfg: auth.GitHubLoginHandlerConfig{TrustProxy: false},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				m := mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
				m.EXPECT().
					StartAuthentication(gomock.Any(), gomock.Any(), gomock.Any()).
					Return("", errors.New("usecase error"))
				return m
			},
			expectedStatus: http.StatusInternalServerError,
			expectedURL:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			e := echo.New()

			url := "/auth/github/login"
			if tt.args.repository != "" {
				url += "?repository=" + tt.args.repository
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.args.host != "" {
				req.Host = tt.args.host
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUC := tt.setupMock(ctrl)
			handler := auth.GitHubLoginHandler(mockUC, tt.cfg)

			err := handler(c)
			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedURL != "" {
				location := rec.Header().Get("Location")
				if location != tt.expectedURL {
					t.Errorf("expected Location %s, got %s", tt.expectedURL, location)
				}
			}
		})
	}
}

func TestResolveScheme(t *testing.T) {
	tests := []struct {
		name            string
		trustProxy      bool
		xForwardedProto string
		want            string
	}{
		{
			name:            "正常系: TrustProxy=falseの場合、X-Forwarded-Protoを無視しc.Scheme()を使用",
			trustProxy:      false,
			xForwardedProto: "https",
			want:            "http",
		},
		{
			name:            "正常系: TrustProxy=trueでX-Forwarded-Proto=httpsの場合、httpsを返す",
			trustProxy:      true,
			xForwardedProto: "https",
			want:            "https",
		},
		{
			name:            "正常系: TrustProxy=trueでX-Forwarded-Proto=httpの場合、httpを返す",
			trustProxy:      true,
			xForwardedProto: "http",
			want:            "http",
		},
		{
			name:            "正常系: TrustProxy=trueでX-Forwarded-Proto=HTTPSの場合、小文字に正規化してhttpsを返す",
			trustProxy:      true,
			xForwardedProto: "HTTPS",
			want:            "https",
		},
		{
			name:            "正常系: TrustProxy=trueでX-Forwarded-Proto=Httpの場合、小文字に正規化してhttpを返す",
			trustProxy:      true,
			xForwardedProto: "Http",
			want:            "http",
		},
		{
			name:            "異常系: TrustProxy=trueでX-Forwarded-Protoが空の場合、c.Scheme()を使用",
			trustProxy:      true,
			xForwardedProto: "",
			want:            "http",
		},
		{
			name:            "異常系: TrustProxy=trueでX-Forwarded-Protoが不正な値の場合、c.Scheme()を使用",
			trustProxy:      true,
			xForwardedProto: "ftp",
			want:            "http",
		},
		{
			name:            "異常系: TrustProxy=trueでX-Forwarded-Protoが攻撃的な値の場合、c.Scheme()を使用",
			trustProxy:      true,
			xForwardedProto: "javascript:",
			want:            "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.xForwardedProto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.xForwardedProto)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			got := auth.ResolveScheme(c, tt.trustProxy)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsHostAllowed(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		allowedHosts []string
		want         bool
	}{
		{
			name:         "正常系: 許可リストに含まれるホストの場合はtrueを返す",
			host:         "example.com",
			allowedHosts: []string{"example.com", "api.example.com"},
			want:         true,
		},
		{
			name:         "正常系: 複数の許可ホストの2番目にマッチする場合はtrueを返す",
			host:         "api.example.com",
			allowedHosts: []string{"example.com", "api.example.com"},
			want:         true,
		},
		{
			name:         "正常系: ポート付きホストが許可リストにマッチする場合はtrueを返す",
			host:         "example.com:8080",
			allowedHosts: []string{"example.com:8080"},
			want:         true,
		},
		{
			name:         "正常系: 許可リストが空の場合は全てのホストを許可しtrueを返す",
			host:         "any-host.com",
			allowedHosts: []string{},
			want:         true,
		},
		{
			name:         "正常系: 許可リストがnilの場合は全てのホストを許可しtrueを返す",
			host:         "any-host.com",
			allowedHosts: nil,
			want:         true,
		},
		{
			name:         "異常系: 許可リストに含まれないホストの場合はfalseを返す",
			host:         "malicious.com",
			allowedHosts: []string{"example.com", "api.example.com"},
			want:         false,
		},
		{
			name:         "異常系: ポート番号が異なる場合はfalseを返す",
			host:         "example.com:9090",
			allowedHosts: []string{"example.com:8080"},
			want:         false,
		},
		{
			name:         "異常系: 空のホストの場合はfalseを返す",
			host:         "",
			allowedHosts: []string{"example.com"},
			want:         false,
		},
		{
			name:         "異常系: サブドメイン攻撃の場合はfalseを返す",
			host:         "evil.example.com",
			allowedHosts: []string{"example.com"},
			want:         false,
		},
		{
			name:         "異常系: プレフィックス攻撃の場合はfalseを返す",
			host:         "example.com.evil.com",
			allowedHosts: []string{"example.com"},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.IsHostAllowed(tt.host, tt.allowedHosts)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGitHubLoginHandler_XForwardedProto(t *testing.T) {
	tests := []struct {
		name             string
		trustProxy       bool
		xForwardedProto  string
		wantSchemePrefix string
	}{
		{
			name:             "正常系: TrustProxy=trueでX-Forwarded-Proto=httpsの場合、redirectURIがhttpsで始まる",
			trustProxy:       true,
			xForwardedProto:  "https",
			wantSchemePrefix: "https://",
		},
		{
			name:             "正常系: TrustProxy=falseでX-Forwarded-Proto=httpsの場合、redirectURIがhttpで始まる",
			trustProxy:       false,
			xForwardedProto:  "https",
			wantSchemePrefix: "http://",
		},
		{
			name:             "正常系: TrustProxy=trueでX-Forwarded-Protoが空の場合、redirectURIがhttpで始まる",
			trustProxy:       true,
			xForwardedProto:  "",
			wantSchemePrefix: "http://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			var capturedRedirect string
			mockUC := mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
			mockUC.EXPECT().
				StartAuthentication(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ interface{}, _ interface{}, redirectURI string) (string, error) {
					capturedRedirect = redirectURI
					return "https://github.com/login/oauth/authorize", nil
				})

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/auth/github/login?repository=owner/repo", nil)
			if tt.xForwardedProto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.xForwardedProto)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			cfg := auth.GitHubLoginHandlerConfig{TrustProxy: tt.trustProxy}
			handler := auth.GitHubLoginHandler(mockUC, cfg)

			err := handler(c)
			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}

			if len(capturedRedirect) < len(tt.wantSchemePrefix) {
				t.Fatalf("capturedRedirect too short: %s", capturedRedirect)
			}
			gotPrefix := capturedRedirect[:len(tt.wantSchemePrefix)]
			if diff := cmp.Diff(tt.wantSchemePrefix, gotPrefix); diff != "" {
				t.Errorf("redirect URI scheme mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
