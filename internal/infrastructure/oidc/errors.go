package oidc

import "errors"

// エラー定義
var (
	// ErrInvalidToken はトークンが無効な場合に返されます
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken はトークンが期限切れの場合に返されます
	ErrExpiredToken = errors.New("token expired")
	// ErrInvalidIssuer はissuerが不正な場合に返されます
	ErrInvalidIssuer = errors.New("invalid issuer")
	// ErrInvalidAudience はaudienceが不正な場合に返されます
	ErrInvalidAudience = errors.New("invalid audience")
	// ErrJWKSFetchFailed はJWKS Endpointからの取得に失敗した場合に返されます
	ErrJWKSFetchFailed = errors.New("failed to fetch JWKS")
	// ErrKeyIDNotFound は指定されたKey IDが見つからない場合に返されます
	ErrKeyIDNotFound = errors.New("key ID not found")
	// ErrMissingKeyID はトークンヘッダーにKey IDが含まれていない場合に返されます
	ErrMissingKeyID = errors.New("missing key ID in token header")
	// ErrInvalidRepository はリポジトリが許可されていない場合に返されます
	ErrInvalidRepository = errors.New("repository not allowed")
	// ErrExponentOutOfRange は指数値がプラットフォームのint範囲外の場合に返されます
	ErrExponentOutOfRange = errors.New("exponent out of range")
)
