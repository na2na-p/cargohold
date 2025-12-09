package oidc_test

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/na2na-p/cargohold/internal/infrastructure/oidc"
	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
)

func TestJWKSFetcher_GetPublicKey(t *testing.T) {
	type args struct {
		jwksURL  string
		keyID    string
		provider string
	}
	tests := []struct {
		name            string
		args            args
		setup           func(t *testing.T) (*httptest.Server, *redis.RedisClient, redismock.ClientMock, *rsa.PublicKey, func())
		want            *rsa.PublicKey
		wantErr         error
		wantErrContains string
	}{
		{
			name: "正常系: JWKS Endpointから公開鍵を取得する",
			args: args{
				keyID:    "test-key-id",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, *redis.RedisClient, redismock.ClientMock, *rsa.PublicKey, func()) {
				t.Helper()

				// Redisモックのセットアップ
				redisClient, mock := setupRedisMock(t)

				// テスト用のRSA鍵を生成
				privateKey := generateTestRSAKey(t)
				publicKey := &privateKey.PublicKey

				// モックJWK Setを作成
				jwkSet := createMockJWKSet(t, "test-key-id", publicKey)

				// モックJWKS Endpointを作成
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(jwkSet)
				}))

				// Redisキャッシュキー
				cacheKey := "lfs:oidc:jwks:test-provider"

				// Redisモックの期待値を設定
				// 1. GetJSONでキャッシュミス（redis.Nilを返す）
				mock.ExpectGet(cacheKey).RedisNil()
				// 2. SetJSONでキャッシュ保存
				jwkSetJSON, _ := json.Marshal(jwkSet)
				mock.ExpectSet(cacheKey, jwkSetJSON, 24*time.Hour).SetVal("OK")

				return server, redisClient, mock, publicKey, func() {
					server.Close()
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Errorf("Redisモックの期待値が満たされていません: %v", err)
					}
				}
			},
			wantErr: nil,
		},
		{
			name: "正常系: Redisキャッシュから公開鍵を取得する",
			args: args{
				keyID:    "test-key-id",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, *redis.RedisClient, redismock.ClientMock, *rsa.PublicKey, func()) {
				t.Helper()

				// Redisモックのセットアップ
				redisClient, mock := setupRedisMock(t)

				// テスト用のRSA鍵を生成
				privateKey := generateTestRSAKey(t)
				publicKey := &privateKey.PublicKey

				// モックJWK Setを作成
				jwkSet := createMockJWKSet(t, "test-key-id", publicKey)

				// モックJWKS Endpoint（呼ばれないはず）
				callCount := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					callCount++
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(jwkSet)
				}))

				// Redisキャッシュキー
				cacheKey := "lfs:oidc:jwks:test-provider"

				// Redisモックの期待値を設定
				// GetJSONでキャッシュヒット
				jwkSetJSON, _ := json.Marshal(jwkSet)
				mock.ExpectGet(cacheKey).SetVal(string(jwkSetJSON))

				return server, redisClient, mock, publicKey, func() {
					server.Close()
					// HTTPエンドポイントが呼ばれていないことを確認
					if callCount != 0 {
						t.Errorf("JWKS Endpointが呼ばれました（キャッシュヒットするはず）: callCount=%d", callCount)
					}
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Errorf("Redisモックの期待値が満たされていません: %v", err)
					}
				}
			},
			wantErr: nil,
		},
		{
			name: "異常系: JWKS Endpointが404を返す場合、エラーが返る",
			args: args{
				keyID:    "test-key-id",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, *redis.RedisClient, redismock.ClientMock, *rsa.PublicKey, func()) {
				t.Helper()

				// Redisモックのセットアップ
				redisClient, mock := setupRedisMock(t)

				// モックJWKS Endpoint（404を返す）
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))

				// Redisキャッシュキー
				cacheKey := "lfs:oidc:jwks:test-provider"

				// Redisモックの期待値を設定
				// GetJSONでキャッシュミス
				mock.ExpectGet(cacheKey).RedisNil()

				return server, redisClient, mock, nil, func() {
					server.Close()
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Errorf("Redisモックの期待値が満たされていません: %v", err)
					}
				}
			},
			want:    nil,
			wantErr: oidc.ErrJWKSFetchFailed,
		},
		{
			name: "異常系: Key IDが見つからない場合、エラーが返る",
			args: args{
				keyID:    "non-existent-key-id",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, *redis.RedisClient, redismock.ClientMock, *rsa.PublicKey, func()) {
				t.Helper()

				// Redisモックのセットアップ
				redisClient, mock := setupRedisMock(t)

				// テスト用のRSA鍵を生成
				privateKey := generateTestRSAKey(t)
				publicKey := &privateKey.PublicKey

				// モックJWK Set（異なるKey IDを持つ）
				jwkSet := createMockJWKSet(t, "different-key-id", publicKey)

				// モックJWKS Endpoint
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(jwkSet)
				}))

				// Redisキャッシュキー
				cacheKey := "lfs:oidc:jwks:test-provider"

				// Redisモックの期待値を設定
				// 1. GetJSONでキャッシュミス
				mock.ExpectGet(cacheKey).RedisNil()
				// 2. SetJSONでキャッシュ保存
				jwkSetJSON, _ := json.Marshal(jwkSet)
				mock.ExpectSet(cacheKey, jwkSetJSON, 24*time.Hour).SetVal("OK")

				return server, redisClient, mock, nil, func() {
					server.Close()
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Errorf("Redisモックの期待値が満たされていません: %v", err)
					}
				}
			},
			want:    nil,
			wantErr: oidc.ErrKeyIDNotFound,
		},
		{
			name: "異常系: 指数値がint64の範囲を超える場合、エラーが返る",
			args: args{
				keyID:    "test-key-id",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, *redis.RedisClient, redismock.ClientMock, *rsa.PublicKey, func()) {
				t.Helper()

				// Redisモックのセットアップ
				redisClient, mock := setupRedisMock(t)

				// テスト用のRSA鍵を生成（modulusだけ使用）
				privateKey := generateTestRSAKey(t)
				publicKey := &privateKey.PublicKey

				// 大きすぎる指数値を持つJWK Setを作成
				nBytes := publicKey.N.Bytes()
				n := base64.RawURLEncoding.EncodeToString(nBytes)

				// int64の最大値を超える指数値を作成（big.Int.IsInt64()がfalseを返す）
				largeExponent := new(big.Int).SetUint64(1 << 63)
				largeExponent = largeExponent.Mul(largeExponent, big.NewInt(2))
				eBytes := largeExponent.Bytes()
				e := base64.RawURLEncoding.EncodeToString(eBytes)

				jwkSet := oidc.JWKSet{
					Keys: []oidc.JWK{
						{
							Kid: "test-key-id",
							Kty: "RSA",
							Use: "sig",
							N:   n,
							E:   e,
						},
					},
				}

				// モックJWKS Endpoint
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(jwkSet)
				}))

				// Redisキャッシュキー
				cacheKey := "lfs:oidc:jwks:test-provider"

				// Redisモックの期待値を設定
				// 1. GetJSONでキャッシュミス
				mock.ExpectGet(cacheKey).RedisNil()
				// 2. SetJSONでキャッシュ保存
				jwkSetJSON, _ := json.Marshal(jwkSet)
				mock.ExpectSet(cacheKey, jwkSetJSON, 24*time.Hour).SetVal("OK")

				return server, redisClient, mock, nil, func() {
					server.Close()
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Errorf("Redisモックの期待値が満たされていません: %v", err)
					}
				}
			},
			want:    nil,
			wantErr: oidc.ErrExponentOutOfRange,
		},
		{
			name: "異常系: レスポンスボディがmaxJWKSBodySize(1MiB)を超える場合、JSONパースエラーが返る",
			args: args{
				keyID:    "test-key-id",
				provider: "test-provider",
			},
			setup: func(t *testing.T) (*httptest.Server, *redis.RedisClient, redismock.ClientMock, *rsa.PublicKey, func()) {
				t.Helper()

				// Redisモックのセットアップ
				redisClient, mock := setupRedisMock(t)

				// 1 MiB + 1 バイトのレスポンスを返すサーバー
				// io.LimitReaderにより途中で切り詰められ、不完全なJSONとなりパースエラーになる
				const maxSize = 1 << 20
				largeBody := `{"keys": [{"kid": "test-key-id", "kty": "RSA", "n": "` + strings.Repeat("A", maxSize) + `"}]}`

				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(largeBody))
				}))

				// Redisキャッシュキー
				cacheKey := "lfs:oidc:jwks:test-provider"

				// Redisモックの期待値を設定
				// GetJSONでキャッシュミス
				mock.ExpectGet(cacheKey).RedisNil()

				return server, redisClient, mock, nil, func() {
					server.Close()
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Errorf("Redisモックの期待値が満たされていません: %v", err)
					}
				}
			},
			want:            nil,
			wantErr:         nil,
			wantErrContains: "JWK Setのパースに失敗しました",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, redisClient, _, expectedKey, cleanup := tt.setup(t)
			defer cleanup()

			// jwksURLをserverのURLに設定
			tt.args.jwksURL = server.URL

			fetcher := oidc.NewJWKSFetcher(redisClient)
			ctx := context.Background()
			got, err := fetcher.GetPublicKey(ctx, tt.args.jwksURL, tt.args.keyID, tt.args.provider)

			// エラー検証
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("GetPublicKey() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetPublicKey() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if tt.wantErrContains != "" {
				if err == nil {
					t.Errorf("GetPublicKey() error = nil, wantErrContains %q", tt.wantErrContains)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("GetPublicKey() error = %v, wantErrContains %q", err, tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Errorf("GetPublicKey() unexpected error = %v", err)
				return
			}

			// 戻り値の検証
			gotRSA, ok := got.(*rsa.PublicKey)
			if !ok {
				t.Errorf("GetPublicKey() got type = %T, want *rsa.PublicKey", got)
				return
			}

			// RSA公開鍵のModulusとExponentを比較
			// big.Intはunexportedフィールドを持つため、cmpopts.IgnoreUnexportedを使用
			cmpOpts := cmpopts.IgnoreUnexported(big.Int{})
			if diff := cmp.Diff(expectedKey.N, gotRSA.N, cmpOpts); diff != "" {
				t.Errorf("RSA Modulus mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(expectedKey.E, gotRSA.E); diff != "" {
				t.Errorf("RSA Exponent mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
