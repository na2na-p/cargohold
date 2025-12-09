package oidc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"time"

	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
)

const (
	jwksCacheTTL    = 24 * time.Hour
	maxJWKSBodySize = 1 << 20 // 1 MiB - JWKSレスポンスの最大サイズ
)

// JWK はJSON Web Keyを表します
type JWK struct {
	Kid string `json:"kid"` // Key ID
	Kty string `json:"kty"` // Key Type
	Use string `json:"use"` // Public Key Use
	N   string `json:"n"`   // Modulus
	E   string `json:"e"`   // Exponent
}

// JWKSet はJWK Setを表します
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// JWKSFetcher はJWKS公開鍵の取得を担当します
type JWKSFetcher struct {
	cacheClient CacheClient
	httpClient  *http.Client
}

// NewJWKSFetcher は新しいJWKSFetcherを作成します
func NewJWKSFetcher(cacheClient CacheClient) *JWKSFetcher {
	return &JWKSFetcher{
		cacheClient: cacheClient,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetPublicKey はJWKS EndpointからJWKを取得し、指定されたKey IDの公開鍵を返します
// プロバイダー名はJWKS URLから抽出されます（キャッシュキーに使用）
func (f *JWKSFetcher) GetPublicKey(ctx context.Context, jwksURL string, keyID string, provider string) (interface{}, error) {
	cacheKey := redis.OIDCJWKSKey(provider)

	var jwkSet JWKSet
	err := f.cacheClient.GetJSON(ctx, cacheKey, &jwkSet)
	if err == nil {
		key, err := f.findPublicKey(&jwkSet, keyID)
		if err == nil {
			return key, nil
		}
	}

	jwkSet, err = f.fetchJWKSet(ctx, jwksURL)
	if err != nil {
		return nil, fmt.Errorf("JWKS Endpointからの取得に失敗しました: %w", err)
	}

	_ = f.cacheClient.SetJSON(ctx, cacheKey, jwkSet, jwksCacheTTL)

	return f.findPublicKey(&jwkSet, keyID)
}

// fetchJWKSet はJWKS EndpointからJWK Setを取得します
func (f *JWKSFetcher) fetchJWKSet(ctx context.Context, jwksURL string) (JWKSet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return JWKSet{}, fmt.Errorf("HTTPリクエストの作成に失敗しました: %w", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return JWKSet{}, fmt.Errorf("HTTPリクエストに失敗しました: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return JWKSet{}, fmt.Errorf("%w: status code %d", ErrJWKSFetchFailed, resp.StatusCode)
	}

	limitedReader := io.LimitReader(resp.Body, maxJWKSBodySize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return JWKSet{}, fmt.Errorf("レスポンスボディの読み取りに失敗しました: %w", err)
	}

	var jwkSet JWKSet
	if err := json.Unmarshal(body, &jwkSet); err != nil {
		return JWKSet{}, fmt.Errorf("JWK Setのパースに失敗しました: %w", err)
	}

	return jwkSet, nil
}

// findPublicKey はJWK SetからKey IDに対応する公開鍵を探します
func (f *JWKSFetcher) findPublicKey(jwkSet *JWKSet, keyID string) (interface{}, error) {
	for _, jwk := range jwkSet.Keys {
		if jwk.Kid == keyID && jwk.Kty == "RSA" {
			return f.parseRSAPublicKey(&jwk)
		}
	}
	return nil, fmt.Errorf("%w: key ID '%s'", ErrKeyIDNotFound, keyID)
}

// parseRSAPublicKey はJWKからRSA公開鍵を生成します
func (f *JWKSFetcher) parseRSAPublicKey(jwk *JWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("modulus のデコードに失敗しました: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("exponent のデコードに失敗しました: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	if !e.IsInt64() {
		return nil, fmt.Errorf("%w: %v", ErrExponentOutOfRange, e)
	}
	exponent := e.Int64()
	if exponent < 0 || exponent > int64(math.MaxInt) {
		return nil, fmt.Errorf("%w: %v", ErrExponentOutOfRange, exponent)
	}

	publicKey := &rsa.PublicKey{
		N: n,
		E: int(exponent),
	}

	return publicKey, nil
}
