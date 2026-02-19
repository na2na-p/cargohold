package auth_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/na2na-p/cargohold/internal/handler/auth"
	"github.com/na2na-p/cargohold/internal/handler/middleware"
	"github.com/na2na-p/cargohold/internal/usecase"
	mockauth "github.com/na2na-p/cargohold/tests/handler/auth"
	"go.uber.org/mock/gomock"
)

func TestGitHubCallbackHandler(t *testing.T) {
	type args struct {
		code  string
		state string
	}
	tests := []struct {
		name                     string
		args                     args
		host                     string
		setupMock                func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface
		expectedStatus           int
		expectCookie             bool
		expectedCookieName       string
		expectedRedirectLocation string
		wantAppError             bool
	}{
		{
			name: "正常系: コールバック処理が成功しセッションCookieが設定される",
			args: args{
				code:  "valid-auth-code",
				state: "valid-state",
			},
			host: "example.com",
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				m := mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
				m.EXPECT().
					HandleCallback(gomock.Any(), "valid-auth-code", "valid-state").
					Return("session-id-12345", nil)
				return m
			},
			expectedStatus:           http.StatusFound,
			expectCookie:             true,
			expectedCookieName:       "lfs_session",
			expectedRedirectLocation: "/auth/session?session_id=session-id-12345&host=example.com",
			wantAppError:             false,
		},
		{
			name: "異常系: codeパラメータが空の場合はBadRequestを返す",
			args: args{
				code:  "",
				state: "valid-state",
			},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				return mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
			},
			expectedStatus: http.StatusBadRequest,
			expectCookie:   false,
			wantAppError:   true,
		},
		{
			name: "異常系: stateパラメータが空の場合はBadRequestを返す",
			args: args{
				code:  "valid-auth-code",
				state: "",
			},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				return mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
			},
			expectedStatus: http.StatusBadRequest,
			expectCookie:   false,
			wantAppError:   true,
		},
		{
			name: "異常系: codeとstateの両方が空の場合はBadRequestを返す",
			args: args{
				code:  "",
				state: "",
			},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				return mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
			},
			expectedStatus: http.StatusBadRequest,
			expectCookie:   false,
			wantAppError:   true,
		},
		{
			name: "異常系: state検証が失敗した場合はUnauthorizedを返す",
			args: args{
				code:  "valid-auth-code",
				state: "invalid-state",
			},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				m := mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
				m.EXPECT().
					HandleCallback(gomock.Any(), "valid-auth-code", "invalid-state").
					Return("", fmt.Errorf("%w: state not found", usecase.ErrInvalidState))
				return m
			},
			expectedStatus: http.StatusUnauthorized,
			expectCookie:   false,
			wantAppError:   true,
		},
		{
			name: "異常系: リポジトリアクセス権がない場合はForbiddenを返す",
			args: args{
				code:  "valid-auth-code",
				state: "valid-state",
			},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				m := mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
				m.EXPECT().
					HandleCallback(gomock.Any(), "valid-auth-code", "valid-state").
					Return("", fmt.Errorf("%w: access denied", usecase.ErrRepositoryAccessDenied))
				return m
			},
			expectedStatus: http.StatusForbidden,
			expectCookie:   false,
			wantAppError:   true,
		},
		{
			name: "異常系: コード交換が失敗した場合はUnauthorizedを返す",
			args: args{
				code:  "invalid-code",
				state: "valid-state",
			},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				m := mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
				m.EXPECT().
					HandleCallback(gomock.Any(), "invalid-code", "valid-state").
					Return("", fmt.Errorf("%w: exchange failed", usecase.ErrCodeExchangeFailed))
				return m
			},
			expectedStatus: http.StatusUnauthorized,
			expectCookie:   false,
			wantAppError:   true,
		},
		{
			name: "異常系: その他のエラーの場合はInternalServerErrorを返す",
			args: args{
				code:  "valid-auth-code",
				state: "valid-state",
			},
			setupMock: func(ctrl *gomock.Controller) *mockauth.MockGitHubOAuthUseCaseInterface {
				m := mockauth.NewMockGitHubOAuthUseCaseInterface(ctrl)
				m.EXPECT().
					HandleCallback(gomock.Any(), "valid-auth-code", "valid-state").
					Return("", errors.New("unexpected error"))
				return m
			},
			expectedStatus: http.StatusInternalServerError,
			expectCookie:   false,
			wantAppError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			e := echo.New()

			url := "/auth/github/callback"
			queryParams := ""
			if tt.args.code != "" {
				queryParams += "code=" + tt.args.code
			}
			if tt.args.state != "" {
				if queryParams != "" {
					queryParams += "&"
				}
				queryParams += "state=" + tt.args.state
			}
			if queryParams != "" {
				url += "?" + queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.host != "" {
				req.Host = tt.host
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUC := tt.setupMock(ctrl)
			handler := auth.GitHubCallbackHandler(mockUC)

			err := handler(c)

			if tt.wantAppError {
				if err == nil {
					t.Fatal("expected AppError, got nil")
				}
				appErr, ok := err.(*middleware.AppError)
				if !ok {
					t.Fatalf("expected *middleware.AppError, got %T", err)
				}
				if appErr.StatusCode != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, appErr.StatusCode)
				}
			} else {
				if err != nil {
					t.Fatalf("handler returned error: %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
				}

				if tt.expectedRedirectLocation != "" {
					location := rec.Header().Get("Location")
					expectedParsed, _ := neturl.Parse(tt.expectedRedirectLocation)
					actualParsed, _ := neturl.Parse(location)

					if actualParsed.Path != expectedParsed.Path {
						t.Errorf("expected redirect path %s, got %s", expectedParsed.Path, actualParsed.Path)
					}

					expectedParams := expectedParsed.Query()
					actualParams := actualParsed.Query()

					if actualParams.Get("session_id") != expectedParams.Get("session_id") {
						t.Errorf("expected session_id param %s, got %s", expectedParams.Get("session_id"), actualParams.Get("session_id"))
					}

					if !strings.Contains(actualParams.Get("host"), expectedParams.Get("host")) {
						t.Errorf("expected host param to contain %s, got %s", expectedParams.Get("host"), actualParams.Get("host"))
					}
				}
			}

			if tt.expectCookie {
				cookies := rec.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == tt.expectedCookieName {
						found = true
						if !cookie.HttpOnly {
							t.Error("Cookie should be HttpOnly")
						}
						if !cookie.Secure {
							t.Error("Cookie should be Secure")
						}
						if cookie.SameSite != http.SameSiteLaxMode {
							t.Error("Cookie should have SameSite=Lax")
						}
						if cookie.MaxAge != 86400 {
							t.Errorf("expected MaxAge 86400, got %d", cookie.MaxAge)
						}
						if cookie.Path != "/" {
							t.Errorf("expected Path '/', got '%s'", cookie.Path)
						}
					}
				}
				if !found {
					t.Errorf("Cookie '%s' not found", tt.expectedCookieName)
				}
			}
		})
	}
}
