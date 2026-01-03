package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
)

type mockGitHubOIDCProvider struct{}

func (m *mockGitHubOIDCProvider) VerifyIDToken(ctx context.Context, token string) (*domain.GitHubUserInfo, error) {
	return domain.NewGitHubUserInfo(
		"github-user-123",
		"owner/repo",
		"refs/heads/main",
		"testuser",
	), nil
}

type mockSessionClient struct {
	sessions map[string]*domain.UserInfo
}

func newMockSessionClient() *mockSessionClient {
	return &mockSessionClient{
		sessions: make(map[string]*domain.UserInfo),
	}
}

func (m *mockSessionClient) GetSession(ctx context.Context, sessionID string) (*domain.UserInfo, error) {
	if userInfo, ok := m.sessions[sessionID]; ok {
		return userInfo, nil
	}
	return nil, errors.New("session not found")
}

type mockRepositoryAllowlistRepository struct{}

func (m *mockRepositoryAllowlistRepository) IsAllowed(ctx context.Context, repository *domain.AllowedRepository) (bool, error) {
	return true, nil
}

func (m *mockRepositoryAllowlistRepository) Add(ctx context.Context, repository *domain.AllowedRepository) error {
	return nil
}

func (m *mockRepositoryAllowlistRepository) Remove(ctx context.Context, repository *domain.AllowedRepository) error {
	return nil
}

func (m *mockRepositoryAllowlistRepository) List(ctx context.Context) ([]*domain.AllowedRepository, error) {
	return nil, nil
}

func mustNewUserInfo(t *testing.T, sub, email, name string, provider domain.ProviderType, repository *domain.RepositoryIdentifier, ref string) *domain.UserInfo {
	t.Helper()
	userInfo, err := domain.NewUserInfo(sub, email, name, provider, repository, ref)
	if err != nil {
		t.Fatalf("failed to create UserInfo: %v", err)
	}
	return userInfo
}

func TestAuthUseCase_AuthenticateSession(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		setupMock   func() *mockSessionClient
		expectError bool
		expectedSub string
	}{
		{
			name:      "正常系: セッションから認証できる",
			sessionID: "test-session-id",
			setupMock: func() *mockSessionClient {
				mockSession := newMockSessionClient()
				userInfo := mustNewUserInfo(t, "user-123", "test@example.com", "Test User", domain.ProviderTypeGitHub, nil, "")
				mockSession.sessions["test-session-id"] = userInfo
				return mockSession
			},
			expectError: false,
			expectedSub: "user-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSession := tt.setupMock()
			mockGitHub := &mockGitHubOIDCProvider{}
			mockRepoAllowlist := &mockRepositoryAllowlistRepository{}

			authUC := usecase.NewAuthUseCase(mockGitHub, mockRepoAllowlist, mockSession)

			ctx := context.Background()
			userInfo, err := authUC.AuthenticateSession(ctx, tt.sessionID)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError {
				if userInfo == nil {
					t.Fatalf("userInfo is nil, expected non-nil")
				}
				if userInfo.Sub() != tt.expectedSub {
					t.Errorf("expected sub %s, got %s", tt.expectedSub, userInfo.Sub())
				}
			}
		})
	}
}

func TestAuthUseCase_AuthenticateGitHubOIDC(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expectError bool
		expectedSub string
	}{
		{
			name:        "正常系: GitHub OIDCトークンから認証できる",
			token:       "test-github-token",
			expectError: false,
			expectedSub: "github-user-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGitHub := &mockGitHubOIDCProvider{}
			mockSession := newMockSessionClient()
			mockRepoAllowlist := &mockRepositoryAllowlistRepository{}

			authUC := usecase.NewAuthUseCase(mockGitHub, mockRepoAllowlist, mockSession)

			ctx := context.Background()
			userInfo, err := authUC.AuthenticateGitHubOIDC(ctx, tt.token)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError {
				if userInfo == nil {
					t.Fatalf("userInfo is nil, expected non-nil")
				}
				if userInfo.Sub() != tt.expectedSub {
					t.Errorf("expected sub %s, got %s", tt.expectedSub, userInfo.Sub())
				}
			}
		})
	}
}

var (
	_ usecase.GitHubOIDCProvider           = (*mockGitHubOIDCProvider)(nil)
	_ usecase.SessionClient                = (*mockSessionClient)(nil)
	_ domain.RepositoryAllowlistRepository = (*mockRepositoryAllowlistRepository)(nil)
)
