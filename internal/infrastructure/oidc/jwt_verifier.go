package oidc

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// JWTVerifier はJWT検証を担当します
type JWTVerifier struct {
	jwksFetcher *JWKSFetcher
}

// NewJWTVerifier は新しいJWTVerifierを作成します
func NewJWTVerifier(cacheClient CacheClient) *JWTVerifier {
	return &JWTVerifier{
		jwksFetcher: NewJWKSFetcher(cacheClient),
	}
}

// VerifyJWT はJWTトークンを検証します
// provider: OIDCプロバイダー名（例: "github"）
func (v *JWTVerifier) VerifyJWT(ctx context.Context, tokenString string, jwksURL string, audience string, issuer string, provider string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("予期しない署名アルゴリズムです: %v", token.Header["alg"])
		}

		keyID, ok := token.Header["kid"].(string)
		if !ok {
			return nil, ErrMissingKeyID
		}

		publicKey, err := v.jwksFetcher.GetPublicKey(ctx, jwksURL, keyID, provider)
		if err != nil {
			return nil, fmt.Errorf("公開鍵の取得に失敗しました: %w", err)
		}

		return publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, ErrMissingKeyID) {
			return nil, ErrMissingKeyID
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("%w: claimsの取得に失敗しました", ErrInvalidToken)
	}

	if err := v.validateIssuer(claims, issuer); err != nil {
		return nil, err
	}

	if err := v.validateAudience(claims, audience); err != nil {
		return nil, err
	}

	return token, nil
}

// validateIssuer はissuer (iss) claimを検証します
func (v *JWTVerifier) validateIssuer(claims jwt.MapClaims, expectedIssuer string) error {
	iss, ok := claims["iss"].(string)
	if !ok {
		return fmt.Errorf("%w: issuerが含まれていません", ErrInvalidIssuer)
	}

	if iss != expectedIssuer {
		return fmt.Errorf("%w: expected=%s, got=%s", ErrInvalidIssuer, expectedIssuer, iss)
	}

	return nil
}

// validateAudience はaudience (aud) claimを検証します
// audienceは文字列または文字列配列の可能性がある
func (v *JWTVerifier) validateAudience(claims jwt.MapClaims, expectedAudience string) error {
	aud, ok := claims["aud"]
	if !ok {
		return fmt.Errorf("%w: audienceが含まれていません", ErrInvalidAudience)
	}

	if audStr, ok := aud.(string); ok {
		if audStr != expectedAudience {
			return fmt.Errorf("%w: expected=%s, got=%s", ErrInvalidAudience, expectedAudience, audStr)
		}
		return nil
	}

	if audArray, ok := aud.([]interface{}); ok {
		for _, a := range audArray {
			if audStr, ok := a.(string); ok && audStr == expectedAudience {
				return nil
			}
		}
		return fmt.Errorf("%w: expected=%s が配列に含まれていません", ErrInvalidAudience, expectedAudience)
	}

	return fmt.Errorf("%w: audienceの型が不正です", ErrInvalidAudience)
}
