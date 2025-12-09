package oidc_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/na2na-p/cargohold/internal/infrastructure/oidc"
	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
)

// setupRedisMock はredismockを使用してRedisクライアントのモックを作成します
func setupRedisMock(t *testing.T) (*redis.RedisClient, redismock.ClientMock) {
	t.Helper()
	db, mock := redismock.NewClientMock()
	client := redis.NewRedisClient(db)
	return client, mock
}

// generateTestRSAKey はテスト用のRSA鍵ペアを生成します
func generateTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("RSA鍵の生成に失敗しました: %v", err)
	}

	return privateKey
}

// createMockJWKSet はモックJWK Setを作成します
func createMockJWKSet(t *testing.T, keyID string, publicKey *rsa.PublicKey) oidc.JWKSet {
	t.Helper()

	// NとEをBase64 URLエンコード
	nBytes := publicKey.N.Bytes()
	eBytes := big.NewInt(int64(publicKey.E)).Bytes()

	n := base64.RawURLEncoding.EncodeToString(nBytes)
	e := base64.RawURLEncoding.EncodeToString(eBytes)

	return oidc.JWKSet{
		Keys: []oidc.JWK{
			{
				Kid: keyID,
				Kty: "RSA",
				Use: "sig",
				N:   n,
				E:   e,
			},
		},
	}
}
