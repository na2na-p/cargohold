package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
)

type mockGitHubOIDCProvider struct{}

type mockCacheKeyGenerator struct{}

func (m *mockCacheKeyGenerator) MetadataKey(oid string) string {
	return "lfs:meta:" + oid
}

func (m *mockCacheKeyGenerator) SessionKey(sessionID string) string {
	return "lfs:session:" + sessionID
}

func (m *mockCacheKeyGenerator) BatchUploadKey(oid string) string {
	return "lfs:batch:upload:" + oid
}

func (m *mockGitHubOIDCProvider) VerifyIDToken(ctx context.Context, token string) (*domain.GitHubUserInfo, error) {
	return domain.NewGitHubUserInfo(
		"github-user-123",
		"owner/repo",
		"refs/heads/main",
		"testuser",
	), nil
}

type mockCacheClient struct {
	cache map[string]interface{}
}

func newMockCacheClient() *mockCacheClient {
	return &mockCacheClient{
		cache: make(map[string]interface{}),
	}
}

func (m *mockCacheClient) Get(ctx context.Context, key string) (string, error) {
	if val, ok := m.cache[key].(string); ok {
		return val, nil
	}
	return "", usecase.ErrCacheMiss
}

func (m *mockCacheClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.cache[key] = value
	return nil
}

func (m *mockCacheClient) Exists(ctx context.Context, key string) (bool, error) {
	_, exists := m.cache[key]
	return exists, nil
}

func (m *mockCacheClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, exists := m.cache[key]
	if !exists {
		return usecase.ErrCacheMiss
	}

	if sessionDataDest, ok := dest.(*usecase.SessionData); ok {
		if sessionData, ok := val.(*usecase.SessionData); ok {
			*sessionDataDest = *sessionData
			return nil
		}
		if mapData, ok := val.(map[string]interface{}); ok {
			if sub, ok := mapData["sub"].(string); ok {
				sessionDataDest.Sub = sub
			}
			if email, ok := mapData["email"].(string); ok {
				sessionDataDest.Email = email
			}
			if name, ok := mapData["name"].(string); ok {
				sessionDataDest.Name = name
			}
			if provider, ok := mapData["provider"].(string); ok {
				sessionDataDest.Provider = provider
			}
			if repository, ok := mapData["repository"].(string); ok {
				sessionDataDest.Repository = repository
			}
			if ref, ok := mapData["ref"].(string); ok {
				sessionDataDest.Ref = ref
			}
			return nil
		}
	}

	if mapVal, ok := val.(map[string]interface{}); ok {
		if sessionData, ok := dest.(*map[string]interface{}); ok {
			*sessionData = mapVal
			return nil
		}
	}
	return fmt.Errorf("GetJSON: type mismatch - cannot convert stored value to destination type")
}

func (m *mockCacheClient) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.cache[key] = value
	return nil
}

func (m *mockCacheClient) Delete(ctx context.Context, key string) error {
	delete(m.cache, key)
	return nil
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

func TestAuthUseCase_AuthenticateSession(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		setupMock   func() *mockCacheClient
		expectError bool
		expectedSub string
	}{
		{
			name:      "正常系: セッションから認証できる",
			sessionID: "test-session-id",
			setupMock: func() *mockCacheClient {
				mockCache := newMockCacheClient()
				sessionData := map[string]interface{}{
					"sub":      "user-123",
					"email":    "test@example.com",
					"name":     "Test User",
					"provider": "github",
				}
				mockCache.cache["lfs:session:test-session-id"] = sessionData
				return mockCache
			},
			expectError: false,
			expectedSub: "user-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := tt.setupMock()
			mockGitHub := &mockGitHubOIDCProvider{}
			mockKeyGenerator := &mockCacheKeyGenerator{}
			mockRepoAllowlist := &mockRepositoryAllowlistRepository{}

			authUC := usecase.NewAuthUseCase(mockGitHub, mockRepoAllowlist, mockCache, mockKeyGenerator)

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
			mockCache := newMockCacheClient()
			mockKeyGenerator := &mockCacheKeyGenerator{}
			mockRepoAllowlist := &mockRepositoryAllowlistRepository{}

			authUC := usecase.NewAuthUseCase(mockGitHub, mockRepoAllowlist, mockCache, mockKeyGenerator)

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
	_ domain.CacheClient                   = (*mockCacheClient)(nil)
	_ domain.CacheKeyGenerator             = (*mockCacheKeyGenerator)(nil)
	_ domain.RepositoryAllowlistRepository = (*mockRepositoryAllowlistRepository)(nil)
)
