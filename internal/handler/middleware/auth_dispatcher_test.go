package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/handler/middleware"
	mock_middleware "github.com/na2na-p/cargohold/tests/handler/middleware"
	"go.uber.org/mock/gomock"
)

func mustParseRepo(t *testing.T, fullName string) *domain.RepositoryIdentifier {
	t.Helper()
	repo, err := domain.NewRepositoryIdentifier(fullName)
	if err != nil {
		t.Fatalf("failed to parse repository: %v", err)
	}
	return repo
}

func mustNewUserInfo(t *testing.T, sub, email, name string, provider domain.ProviderType, repository *domain.RepositoryIdentifier, ref string) *domain.UserInfo {
	t.Helper()
	userInfo, err := domain.NewUserInfo(sub, email, name, provider, repository, ref)
	if err != nil {
		t.Fatalf("failed to create UserInfo: %v", err)
	}
	return userInfo
}

func TestAuthDispatcher(t *testing.T) {
	type fields struct {
		setupMock func(t *testing.T, ctrl *gomock.Controller) *mock_middleware.MockAuthUseCaseInterface
	}
	type args struct {
		method  string
		path    string
		owner   string
		repo    string
		headers map[string]string
		cookies []*http.Cookie
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
		wantNextCalled bool
	}{
		{
			name: "正常系: Bearer認証でリポジトリが一致する場合、nextが呼ばれる",
			fields: fields{
				setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_middleware.MockAuthUseCaseInterface {
					mock := mock_middleware.NewMockAuthUseCaseInterface(ctrl)
					mock.EXPECT().
						AuthenticateGitHubOIDC(gomock.Any(), "valid-token").
						Return(mustNewUserInfo(t,
							"repo:testowner/testrepo:ref:refs/heads/main",
							"",
							"github-actions",
							domain.ProviderTypeGitHub,
							mustParseRepo(t, "testowner/testrepo"),
							"refs/heads/main",
						), nil)
					return mock
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				headers: map[string]string{
					"Authorization": "Bearer valid-token",
				},
			},
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
		{
			name: "異常系: Bearer認証でリポジトリが不一致の場合、403が返る",
			fields: fields{
				setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_middleware.MockAuthUseCaseInterface {
					mock := mock_middleware.NewMockAuthUseCaseInterface(ctrl)
					mock.EXPECT().
						AuthenticateGitHubOIDC(gomock.Any(), "valid-token").
						Return(mustNewUserInfo(t,
							"repo:otherowner/otherrepo:ref:refs/heads/main",
							"",
							"github-actions",
							domain.ProviderTypeGitHub,
							mustParseRepo(t, "otherowner/otherrepo"),
							"refs/heads/main",
						), nil)
					return mock
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				headers: map[string]string{
					"Authorization": "Bearer valid-token",
				},
			},
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name: "異常系: Bearer認証でトークンが無効の場合、401が返る",
			fields: fields{
				setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_middleware.MockAuthUseCaseInterface {
					mock := mock_middleware.NewMockAuthUseCaseInterface(ctrl)
					mock.EXPECT().
						AuthenticateGitHubOIDC(gomock.Any(), "invalid-token").
						Return(nil, errors.New("invalid token"))
					return mock
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				headers: map[string]string{
					"Authorization": "Bearer invalid-token",
				},
			},
			wantStatusCode: http.StatusUnauthorized,
			wantNextCalled: false,
		},
		{
			name: "正常系: セッション認証でリポジトリが一致する場合、nextが呼ばれる",
			fields: fields{
				setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_middleware.MockAuthUseCaseInterface {
					mock := mock_middleware.NewMockAuthUseCaseInterface(ctrl)
					mock.EXPECT().
						AuthenticateSession(gomock.Any(), "valid-session-id").
						Return(mustNewUserInfo(t,
							"user123",
							"",
							"testuser",
							domain.ProviderTypeGitHub,
							mustParseRepo(t, "testowner/testrepo"),
							"",
						), nil)
					return mock
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				cookies: []*http.Cookie{
					{Name: "lfs_session", Value: "valid-session-id"},
				},
			},
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
		{
			name: "異常系: セッション認証でリポジトリが不一致の場合、403が返る",
			fields: fields{
				setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_middleware.MockAuthUseCaseInterface {
					mock := mock_middleware.NewMockAuthUseCaseInterface(ctrl)
					mock.EXPECT().
						AuthenticateSession(gomock.Any(), "valid-session-id").
						Return(mustNewUserInfo(t,
							"user123",
							"",
							"testuser",
							domain.ProviderTypeGitHub,
							mustParseRepo(t, "otherowner/otherrepo"),
							"",
						), nil)
					return mock
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				cookies: []*http.Cookie{
					{Name: "lfs_session", Value: "valid-session-id"},
				},
			},
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name: "異常系: セッション認証でセッションが無効の場合、401が返る",
			fields: fields{
				setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_middleware.MockAuthUseCaseInterface {
					mock := mock_middleware.NewMockAuthUseCaseInterface(ctrl)
					mock.EXPECT().
						AuthenticateSession(gomock.Any(), "invalid-session-id").
						Return(nil, errors.New("session not found"))
					return mock
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				cookies: []*http.Cookie{
					{Name: "lfs_session", Value: "invalid-session-id"},
				},
			},
			wantStatusCode: http.StatusUnauthorized,
			wantNextCalled: false,
		},
		{
			name: "異常系: 認証情報がない場合、401が返る",
			fields: fields{
				setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_middleware.MockAuthUseCaseInterface {
					mock := mock_middleware.NewMockAuthUseCaseInterface(ctrl)
					return mock
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
			},
			wantStatusCode: http.StatusUnauthorized,
			wantNextCalled: false,
		},
		{
			name: "正常系: Bearer認証でリポジトリ名の大文字小文字が異なっても一致として扱う",
			fields: fields{
				setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_middleware.MockAuthUseCaseInterface {
					mock := mock_middleware.NewMockAuthUseCaseInterface(ctrl)
					mock.EXPECT().
						AuthenticateGitHubOIDC(gomock.Any(), "valid-token").
						Return(mustNewUserInfo(t,
							"repo:TestOwner/TestRepo:ref:refs/heads/main",
							"",
							"github-actions",
							domain.ProviderTypeGitHub,
							mustParseRepo(t, "TestOwner/TestRepo"),
							"refs/heads/main",
						), nil)
					return mock
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				headers: map[string]string{
					"Authorization": "Bearer valid-token",
				},
			},
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
		{
			name: "異常系: URLパスからリポジトリ抽出に失敗した場合、400が返る",
			fields: fields{
				setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_middleware.MockAuthUseCaseInterface {
					mock := mock_middleware.NewMockAuthUseCaseInterface(ctrl)
					mock.EXPECT().
						AuthenticateGitHubOIDC(gomock.Any(), "valid-token").
						Return(mustNewUserInfo(t,
							"repo:testowner/testrepo:ref:refs/heads/main",
							"",
							"github-actions",
							domain.ProviderTypeGitHub,
							mustParseRepo(t, "testowner/testrepo"),
							"refs/heads/main",
						), nil)
					return mock
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/info/lfs/objects/batch",
				owner:  "",
				repo:   "",
				headers: map[string]string{
					"Authorization": "Bearer valid-token",
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantNextCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			e := echo.New()
			req := httptest.NewRequest(tt.args.method, tt.args.path, nil)
			for k, v := range tt.args.headers {
				req.Header.Set(k, v)
			}
			for _, cookie := range tt.args.cookies {
				req.AddCookie(cookie)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("owner", "repo")
			c.SetParamValues(tt.args.owner, tt.args.repo)

			nextCalled := false
			nextHandler := func(c echo.Context) error {
				nextCalled = true
				return c.NoContent(http.StatusOK)
			}

			mockUseCase := tt.fields.setupMock(t, ctrl)
			middlewareFunc := middleware.AuthDispatcher(mockUseCase)
			h := middlewareFunc(nextHandler)
			_ = h(c)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("AuthDispatcher() status code = %v, want %v", rec.Code, tt.wantStatusCode)
			}

			if nextCalled != tt.wantNextCalled {
				t.Errorf("AuthDispatcher() nextCalled = %v, want %v", nextCalled, tt.wantNextCalled)
			}
		})
	}
}
