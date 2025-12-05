package oidc_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/infrastructure"
	"github.com/na2na-p/cargohold/internal/infrastructure/oidc"
	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
	mockdomain "github.com/na2na-p/cargohold/tests/domain"
	"go.uber.org/mock/gomock"
)

func createTestGitHubJWT(t *testing.T, privateKey interface{}, keyID, repository, ref, actor string, expiresAt time.Time) string {
	t.Helper()

	claims := jwt.MapClaims{
		"iss":        oidc.GitHubIssuer,
		"aud":        "cargohold",
		"sub":        "repo:" + repository + ":ref:" + ref,
		"repository": repository,
		"ref":        ref,
		"actor":      actor,
		"exp":        expiresAt.Unix(),
		"nbf":        time.Now().Add(-1 * time.Minute).Unix(),
		"iat":        time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("JWTトークンの署名に失敗しました: %v", err)
	}

	return tokenString
}

func TestGitHubOIDCProvider_VerifyIDToken(t *testing.T) {
	tests := []struct {
		name        string
		createToken func(t *testing.T, privateKey interface{}, keyID string) string
		want        *domain.GitHubUserInfo
		wantErr     error
	}{
		{
			name: "正常系: 有効なJWT",
			createToken: func(t *testing.T, privateKey interface{}, keyID string) string {
				repository := "na2na-p/test-repo"
				ref := "refs/heads/main"
				actor := "test-user"
				expiresAt := time.Now().Add(1 * time.Hour)
				return createTestGitHubJWT(t, privateKey, keyID, repository, ref, actor, expiresAt)
			},
			want: domain.NewGitHubUserInfo(
				"repo:na2na-p/test-repo:ref:refs/heads/main",
				"na2na-p/test-repo",
				"refs/heads/main",
				"test-user",
			),
			wantErr: nil,
		},
		{
			name: "異常系: 不正なissuer",
			createToken: func(t *testing.T, privateKey interface{}, keyID string) string {
				repository := "na2na-p/test-repo"
				claims := jwt.MapClaims{
					"iss":        "https://invalid-issuer.com",
					"aud":        "cargohold",
					"sub":        "repo:" + repository + ":ref:refs/heads/main",
					"repository": repository,
					"ref":        "refs/heads/main",
					"actor":      "test-user",
					"exp":        time.Now().Add(1 * time.Hour).Unix(),
					"nbf":        time.Now().Add(-1 * time.Minute).Unix(),
					"iat":        time.Now().Unix(),
				}

				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				token.Header["kid"] = keyID

				tokenString, err := token.SignedString(privateKey)
				if err != nil {
					t.Fatalf("JWTトークンの署名に失敗しました: %v", err)
				}

				return tokenString
			},
			want:    nil,
			wantErr: oidc.ErrInvalidIssuer,
		},
		{
			name: "異常系: 不正なaudience",
			createToken: func(t *testing.T, privateKey interface{}, keyID string) string {
				repository := "na2na-p/test-repo"
				claims := jwt.MapClaims{
					"iss":        oidc.GitHubIssuer,
					"aud":        "invalid-audience",
					"sub":        "repo:" + repository + ":ref:refs/heads/main",
					"repository": repository,
					"ref":        "refs/heads/main",
					"actor":      "test-user",
					"exp":        time.Now().Add(1 * time.Hour).Unix(),
					"nbf":        time.Now().Add(-1 * time.Minute).Unix(),
					"iat":        time.Now().Unix(),
				}

				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				token.Header["kid"] = keyID

				tokenString, err := token.SignedString(privateKey)
				if err != nil {
					t.Fatalf("JWTトークンの署名に失敗しました: %v", err)
				}

				return tokenString
			},
			want:    nil,
			wantErr: oidc.ErrInvalidAudience,
		},
		{
			name: "異常系: 期限切れトークン",
			createToken: func(t *testing.T, privateKey interface{}, keyID string) string {
				repository := "na2na-p/test-repo"
				ref := "refs/heads/main"
				actor := "test-user"
				expiresAt := time.Now().Add(-1 * time.Hour)
				return createTestGitHubJWT(t, privateKey, keyID, repository, ref, actor, expiresAt)
			},
			want:    nil,
			wantErr: oidc.ErrExpiredToken,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redisClient, _ := setupRedisMock(t)

			privateKey := generateTestRSAKey(t)
			publicKey := &privateKey.PublicKey

			keyID := "test-key-id"
			jwkSet := createMockJWKSet(t, keyID, publicKey)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(jwkSet)
			}))
			defer server.Close()

			tokenString := tt.createToken(t, privateKey, keyID)

			provider, err := oidc.NewGitHubOIDCProvider("cargohold", redisClient, server.URL)
			if err != nil {
				t.Fatalf("GitHubOIDCProviderの作成に失敗しました: %v", err)
			}

			ctx := context.Background()

			got, err := provider.VerifyIDToken(ctx, tokenString)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("VerifyIDToken() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("VerifyIDToken() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("VerifyIDToken() unexpected error = %v", err)
				return
			}

			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(domain.GitHubUserInfo{})); diff != "" {
				t.Errorf("VerifyIDToken() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepositoryAllowlist_IsAllowed(t *testing.T) {
	type args struct {
		owner string
		repo  string
	}
	tests := []struct {
		name           string
		args           args
		setupRedisMock func(t *testing.T, mock redismock.ClientMock)
		setupPgMock    func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository
		want           bool
		wantErr        bool
	}{
		{
			name: "正常系: Redisキャッシュヒット（許可されたリポジトリ）",
			args: args{
				owner: "na2na-p",
				repo:  "test-repo",
			},
			setupRedisMock: func(t *testing.T, mock redismock.ClientMock) {
				t.Helper()
				mock.ExpectGet("lfs:oidc:github:repo:na2na-p/test-repo").SetVal("true")
			},
			setupPgMock: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
				return mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "正常系: Redisキャッシュヒット（許可されていないリポジトリ）",
			args: args{
				owner: "na2na-p",
				repo:  "unauthorized-repo",
			},
			setupRedisMock: func(t *testing.T, mock redismock.ClientMock) {
				t.Helper()
				mock.ExpectGet("lfs:oidc:github:repo:na2na-p/unauthorized-repo").SetVal("false")
			},
			setupPgMock: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
				return mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			var redisClient *redis.RedisClient
			var mock redismock.ClientMock

			if tt.setupRedisMock != nil {
				redisClient, mock = setupRedisMock(t)
				tt.setupRedisMock(t, mock)
			} else {
				redisClient, _ = setupRedisMock(t)
			}

			pgRepo := tt.setupPgMock(ctrl)
			allowlist := infrastructure.NewCachingRepositoryAllowlist(pgRepo, redisClient)

			ctx := context.Background()
			allowedRepo := mustNewAllowedRepository(t, tt.args.owner, tt.args.repo)
			got, err := allowlist.IsAllowed(ctx, allowedRepo)

			if (err != nil) != tt.wantErr {
				t.Errorf("IsAllowed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("IsAllowed() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepositoryAllowlist_Add(t *testing.T) {
	type args struct {
		owner string
		repo  string
	}
	tests := []struct {
		name           string
		args           args
		setupRedisMock func(t *testing.T, mock redismock.ClientMock)
		setupPgMock    func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository
		wantErr        bool
	}{
		{
			name: "正常系: リポジトリを追加する",
			args: args{
				owner: "na2na-p",
				repo:  "test-repo",
			},
			setupRedisMock: func(t *testing.T, mock redismock.ClientMock) {
				t.Helper()
				mock.ExpectSet("lfs:oidc:github:repo:na2na-p/test-repo", "true", 5*time.Minute).SetVal("OK")
				mock.ExpectGet("lfs:oidc:github:repo:na2na-p/test-repo").SetVal("true")
			},
			setupPgMock: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
				m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
				m.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)
				return m
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			redisClient, mock := setupRedisMock(t)

			pgRepo := tt.setupPgMock(ctrl)
			allowlist := infrastructure.NewCachingRepositoryAllowlist(pgRepo, redisClient)

			if tt.setupRedisMock != nil {
				tt.setupRedisMock(t, mock)
			}

			ctx := context.Background()
			allowedRepo := mustNewAllowedRepository(t, tt.args.owner, tt.args.repo)
			err := allowlist.Add(ctx, allowedRepo)

			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			allowed, err := allowlist.IsAllowed(ctx, allowedRepo)
			if err != nil {
				t.Errorf("IsAllowed() unexpected error = %v", err)
				return
			}

			if !allowed {
				t.Errorf("Add() did not add repository to allowlist")
			}
		})
	}
}

func TestRepositoryAllowlist_Remove(t *testing.T) {
	type args struct {
		owner string
		repo  string
	}
	tests := []struct {
		name           string
		args           args
		setupRedisMock func(t *testing.T, mock redismock.ClientMock)
		setupPgMock    func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository
		wantErr        bool
	}{
		{
			name: "正常系: リポジトリを削除する",
			args: args{
				owner: "na2na-p",
				repo:  "test-repo",
			},
			setupRedisMock: func(t *testing.T, mock redismock.ClientMock) {
				t.Helper()
				mock.ExpectDel("lfs:oidc:github:repo:na2na-p/test-repo").SetVal(1)
				mock.ExpectGet("lfs:oidc:github:repo:na2na-p/test-repo").RedisNil()
				mock.ExpectSet("lfs:oidc:github:repo:na2na-p/test-repo", "false", 5*time.Minute).SetVal("OK")
			},
			setupPgMock: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
				m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
				m.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)
				m.EXPECT().IsAllowed(gomock.Any(), gomock.Any()).Return(false, nil)
				return m
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			redisClient, mock := setupRedisMock(t)

			pgRepo := tt.setupPgMock(ctrl)
			allowlist := infrastructure.NewCachingRepositoryAllowlist(pgRepo, redisClient)

			if tt.setupRedisMock != nil {
				tt.setupRedisMock(t, mock)
			}

			ctx := context.Background()

			allowedRepo := mustNewAllowedRepository(t, tt.args.owner, tt.args.repo)
			err := allowlist.Remove(ctx, allowedRepo)

			if (err != nil) != tt.wantErr {
				t.Errorf("Remove() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			allowed, err := allowlist.IsAllowed(ctx, allowedRepo)
			if err != nil {
				t.Errorf("IsAllowed() unexpected error = %v", err)
				return
			}

			if allowed {
				t.Errorf("Remove() did not remove repository from allowlist")
			}
		})
	}
}

func mustNewAllowedRepository(t *testing.T, owner, repo string) *domain.AllowedRepository {
	t.Helper()
	ar, err := domain.NewAllowedRepository(owner, repo)
	if err != nil {
		t.Fatalf("failed to create AllowedRepository: %v", err)
	}
	return ar
}
