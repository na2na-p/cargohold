package oidc_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/infrastructure/oidc"
)

// createTestJWT はテスト用のJWTトークンを生成します
func createTestJWT(t *testing.T, privateKey *rsa.PrivateKey, keyID string, issuer string, audience string, expiresAt time.Time, notBefore time.Time) string {
	t.Helper()

	claims := jwt.MapClaims{
		"iss": issuer,
		"aud": audience,
		"exp": expiresAt.Unix(),
		"nbf": notBefore.Unix(),
		"iat": time.Now().Unix(),
		"sub": "test-user",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("JWTトークンの署名に失敗しました: %v", err)
	}

	return tokenString
}

// createTestJWTWithAudienceArray はaudience配列を持つテスト用のJWTトークンを生成します
func createTestJWTWithAudienceArray(t *testing.T, privateKey *rsa.PrivateKey, keyID string, issuer string, audiences []string, expiresAt time.Time, notBefore time.Time) string {
	t.Helper()

	claims := jwt.MapClaims{
		"iss": issuer,
		"aud": audiences,
		"exp": expiresAt.Unix(),
		"nbf": notBefore.Unix(),
		"iat": time.Now().Unix(),
		"sub": "test-user",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("JWTトークンの署名に失敗しました: %v", err)
	}

	return tokenString
}

func TestJWTVerifier_VerifyJWT(t *testing.T) {
	type args struct {
		audience string
		issuer   string
		provider string
	}
	tests := []struct {
		name      string
		args      args
		setup     func(t *testing.T) (*httptest.Server, string, func())
		wantValid bool
		wantErr   error
	}{
		{
			name: "正常系: 有効なJWTトークンを検証する",
			args: args{
				audience: "test-audience",
				issuer:   "https://example.com",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, string, func()) {
				t.Helper()

				// テスト用のRSA鍵を生成
				privateKey := generateTestRSAKey(t)
				publicKey := &privateKey.PublicKey

				// モックJWK Setを作成
				keyID := "test-key-id"
				jwkSet := createMockJWKSet(t, keyID, publicKey)

				// モックJWKS Endpoint
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(jwkSet)
				}))

				// JWTトークンを生成
				issuer := "https://example.com"
				audience := "test-audience"
				expiresAt := time.Now().Add(1 * time.Hour)
				notBefore := time.Now().Add(-1 * time.Minute)
				tokenString := createTestJWT(t, privateKey, keyID, issuer, audience, expiresAt, notBefore)

				return server, tokenString, func() {
					server.Close()
				}
			},
			wantValid: true,
			wantErr:   nil,
		},
		{
			name: "正常系: audience配列を持つ有効なJWTトークンを検証する",
			args: args{
				audience: "test-audience-1",
				issuer:   "https://example.com",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, string, func()) {
				t.Helper()

				// テスト用のRSA鍵を生成
				privateKey := generateTestRSAKey(t)
				publicKey := &privateKey.PublicKey

				// モックJWK Setを作成
				keyID := "test-key-id"
				jwkSet := createMockJWKSet(t, keyID, publicKey)

				// モックJWKS Endpoint
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(jwkSet)
				}))

				// JWTトークンを生成（audience配列）
				issuer := "https://example.com"
				audiences := []string{"test-audience-1", "test-audience-2"}
				expiresAt := time.Now().Add(1 * time.Hour)
				notBefore := time.Now().Add(-1 * time.Minute)
				tokenString := createTestJWTWithAudienceArray(t, privateKey, keyID, issuer, audiences, expiresAt, notBefore)

				return server, tokenString, func() {
					server.Close()
				}
			},
			wantValid: true,
			wantErr:   nil,
		},
		{
			name: "異常系: 期限切れトークンの場合、エラーが返る",
			args: args{
				audience: "test-audience",
				issuer:   "https://example.com",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, string, func()) {
				t.Helper()

				// テスト用のRSA鍵を生成
				privateKey := generateTestRSAKey(t)
				publicKey := &privateKey.PublicKey

				// モックJWK Setを作成
				keyID := "test-key-id"
				jwkSet := createMockJWKSet(t, keyID, publicKey)

				// モックJWKS Endpoint
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(jwkSet)
				}))

				// 期限切れのJWTトークンを生成
				issuer := "https://example.com"
				audience := "test-audience"
				expiresAt := time.Now().Add(-1 * time.Hour) // 1時間前に期限切れ
				notBefore := time.Now().Add(-2 * time.Hour)
				tokenString := createTestJWT(t, privateKey, keyID, issuer, audience, expiresAt, notBefore)

				return server, tokenString, func() {
					server.Close()
				}
			},
			wantValid: false,
			wantErr:   oidc.ErrExpiredToken,
		},
		{
			name: "異常系: 不正な署名の場合、エラーが返る",
			args: args{
				audience: "test-audience",
				issuer:   "https://example.com",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, string, func()) {
				t.Helper()

				// テスト用のRSA鍵を2つ生成（署名用と検証用で異なる鍵を使用）
				privateKeyForSigning := generateTestRSAKey(t)
				privateKeyForVerification := generateTestRSAKey(t)
				publicKeyForVerification := &privateKeyForVerification.PublicKey

				// モックJWK Set（検証用の鍵を使用）
				keyID := "test-key-id"
				jwkSet := createMockJWKSet(t, keyID, publicKeyForVerification)

				// モックJWKS Endpoint
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(jwkSet)
				}))

				// JWTトークンを生成（署名用の鍵で署名）
				issuer := "https://example.com"
				audience := "test-audience"
				expiresAt := time.Now().Add(1 * time.Hour)
				notBefore := time.Now().Add(-1 * time.Minute)
				tokenString := createTestJWT(t, privateKeyForSigning, keyID, issuer, audience, expiresAt, notBefore)

				return server, tokenString, func() {
					server.Close()
				}
			},
			wantValid: false,
			wantErr:   oidc.ErrInvalidToken,
		},
		{
			name: "異常系: 不正なissuerの場合、エラーが返る",
			args: args{
				audience: "test-audience",
				issuer:   "https://example.com",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, string, func()) {
				t.Helper()

				// テスト用のRSA鍵を生成
				privateKey := generateTestRSAKey(t)
				publicKey := &privateKey.PublicKey

				// モックJWK Setを作成
				keyID := "test-key-id"
				jwkSet := createMockJWKSet(t, keyID, publicKey)

				// モックJWKS Endpoint
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(jwkSet)
				}))

				// JWTトークンを生成（異なるissuerを使用）
				tokenIssuer := "https://wrong-issuer.com"
				audience := "test-audience"
				expiresAt := time.Now().Add(1 * time.Hour)
				notBefore := time.Now().Add(-1 * time.Minute)
				tokenString := createTestJWT(t, privateKey, keyID, tokenIssuer, audience, expiresAt, notBefore)

				return server, tokenString, func() {
					server.Close()
				}
			},
			wantValid: false,
			wantErr:   oidc.ErrInvalidIssuer,
		},
		{
			name: "異常系: 不正なaudienceの場合、エラーが返る",
			args: args{
				audience: "test-audience",
				issuer:   "https://example.com",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, string, func()) {
				t.Helper()

				// テスト用のRSA鍵を生成
				privateKey := generateTestRSAKey(t)
				publicKey := &privateKey.PublicKey

				// モックJWK Setを作成
				keyID := "test-key-id"
				jwkSet := createMockJWKSet(t, keyID, publicKey)

				// モックJWKS Endpoint
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(jwkSet)
				}))

				// JWTトークンを生成（異なるaudienceを使用）
				issuer := "https://example.com"
				tokenAudience := "wrong-audience"
				expiresAt := time.Now().Add(1 * time.Hour)
				notBefore := time.Now().Add(-1 * time.Minute)
				tokenString := createTestJWT(t, privateKey, keyID, issuer, tokenAudience, expiresAt, notBefore)

				return server, tokenString, func() {
					server.Close()
				}
			},
			wantValid: false,
			wantErr:   oidc.ErrInvalidAudience,
		},
		{
			name: "異常系: Key IDがないトークンの場合、エラーが返る",
			args: args{
				audience: "test-audience",
				issuer:   "https://example.com",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, string, func()) {
				t.Helper()

				// テスト用のRSA鍵を生成
				privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					t.Fatalf("RSA鍵の生成に失敗しました: %v", err)
				}

				// JWTトークンを生成（Key IDなし）
				issuer := "https://example.com"
				audience := "test-audience"
				claims := jwt.MapClaims{
					"iss": issuer,
					"aud": audience,
					"exp": time.Now().Add(1 * time.Hour).Unix(),
					"nbf": time.Now().Add(-1 * time.Minute).Unix(),
					"iat": time.Now().Unix(),
					"sub": "test-user",
				}

				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				// Key IDをヘッダーに設定しない

				tokenString, err := token.SignedString(privateKey)
				if err != nil {
					t.Fatalf("JWTトークンの署名に失敗しました: %v", err)
				}

				// ダミーサーバー（呼ばれない）
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))

				return server, tokenString, func() {
					server.Close()
				}
			},
			wantValid: false,
			wantErr:   oidc.ErrMissingKeyID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Redisモックのセットアップ
			redisClient, mock := setupRedisMock(t)

			server, tokenString, cleanupServer := tt.setup(t)
			defer cleanupServer()

			// Redisモックの期待値を設定
			cacheKey := "lfs:oidc:jwks:" + tt.args.provider
			// GetJSONでキャッシュミス（全テストケースで一貫性のため）
			mock.ExpectGet(cacheKey).RedisNil()

			// JWTVerifierの作成
			verifier := oidc.NewJWTVerifier(redisClient)

			ctx := context.Background()
			jwksURL := server.URL

			// JWT検証
			token, err := verifier.VerifyJWT(ctx, tokenString, jwksURL, tt.args.audience, tt.args.issuer, tt.args.provider)

			// エラー検証
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("VerifyJWT() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("VerifyJWT() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("VerifyJWT() unexpected error = %v", err)
				return
			}

			// トークンの有効性を検証
			if diff := cmp.Diff(tt.wantValid, token.Valid); diff != "" {
				t.Errorf("Token.Valid mismatch (-want +got):\n%s", diff)
			}

			// 正常系の場合、Claimsも検証
			if tt.wantValid {
				claims, ok := token.Claims.(jwt.MapClaims)
				if !ok {
					t.Errorf("Failed to get claims from token")
					return
				}

				if diff := cmp.Diff(tt.args.issuer, claims["iss"]); diff != "" {
					t.Errorf("Issuer claim mismatch (-want +got):\n%s", diff)
				}

				// audienceは文字列または配列の可能性がある
				aud := claims["aud"]
				if audStr, ok := aud.(string); ok {
					if diff := cmp.Diff(tt.args.audience, audStr); diff != "" {
						t.Errorf("Audience claim mismatch (-want +got):\n%s", diff)
					}
				} else if audArray, ok := aud.([]interface{}); ok {
					found := false
					for _, a := range audArray {
						if audStr, ok := a.(string); ok && audStr == tt.args.audience {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Audience %s not found in array %v", tt.args.audience, audArray)
					}
				}
			}
		})
	}
}
