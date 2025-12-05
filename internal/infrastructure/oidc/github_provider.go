package oidc

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/na2na-p/cargohold/internal/domain"
)

const (
	// GitHubJWKSURL はGitHub ActionsのJWKS Endpointです
	GitHubJWKSURL = "https://token.actions.githubusercontent.com/.well-known/jwks"
	// GitHubIssuer はGitHub ActionsのIssuerです
	GitHubIssuer = "https://token.actions.githubusercontent.com"
)

// githubUserClaims はGitHub ActionsのJWTトークンから取得したクレーム情報を表します
// Infrastructure層内部でのみ使用される構造体です
type githubUserClaims struct {
	sub        string // subject: repo:<owner>/<repo>:ref:<ref>
	repository string // リポジトリ: owner/repo
	ref        string // ブランチ/タグ: refs/heads/main
	actor      string // GitHub Actor
}

// GitHubOIDCProvider はGitHub Actions向けのOIDCプロバイダーです
type GitHubOIDCProvider struct {
	cacheClient CacheClient
	jwtVerifier *JWTVerifier
	jwksURL     string
	issuer      string
	audience    string
}

// NewGitHubOIDCProvider は新しいGitHubOIDCProviderを作成します
func NewGitHubOIDCProvider(
	audience string,
	cacheClient CacheClient,
	jwksURL string,
) (*GitHubOIDCProvider, error) {
	audience = strings.TrimSpace(audience)
	if audience == "" {
		return nil, fmt.Errorf("audience is required")
	}

	if cacheClient == nil {
		return nil, fmt.Errorf("cacheClient is required")
	}

	if jwksURL == "" {
		jwksURL = GitHubJWKSURL
	}
	return &GitHubOIDCProvider{
		cacheClient: cacheClient,
		jwtVerifier: NewJWTVerifier(cacheClient),
		jwksURL:     jwksURL,
		issuer:      GitHubIssuer,
		audience:    audience,
	}, nil
}

// VerifyIDToken はGitHub Actions発行のJWT Bearer Tokenを検証します
// 内部でJWTクレームを処理し、domain.GitHubUserInfoに変換して返します
func (p *GitHubOIDCProvider) VerifyIDToken(ctx context.Context, token string) (*domain.GitHubUserInfo, error) {
	verifiedToken, err := p.jwtVerifier.VerifyJWT(ctx, token, p.jwksURL, p.audience, p.issuer, "github")
	if err != nil {
		return nil, fmt.Errorf("JWT検証に失敗しました: %w", err)
	}

	jwtClaims, ok := verifiedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("%w: claimsの取得に失敗しました", ErrInvalidToken)
	}

	claims := &githubUserClaims{}

	if sub, ok := jwtClaims["sub"].(string); ok {
		claims.sub = sub
	} else {
		return nil, fmt.Errorf("%w: sub claimが含まれていません", ErrInvalidToken)
	}

	if repository, ok := jwtClaims["repository"].(string); ok {
		claims.repository = repository
	} else {
		return nil, fmt.Errorf("%w: repository claimが含まれていません", ErrInvalidToken)
	}

	if ref, ok := jwtClaims["ref"].(string); ok {
		claims.ref = ref
	} else {
		return nil, fmt.Errorf("%w: ref claimが含まれていません", ErrInvalidToken)
	}

	if actor, ok := jwtClaims["actor"].(string); ok {
		claims.actor = actor
	} else {
		return nil, fmt.Errorf("%w: actor claimが含まれていません", ErrInvalidToken)
	}

	return toGitHubDomainUserInfo(claims), nil
}

// toGitHubDomainUserInfo はgithubUserClaimsをdomain.GitHubUserInfoに変換します
func toGitHubDomainUserInfo(claims *githubUserClaims) *domain.GitHubUserInfo {
	return domain.NewGitHubUserInfo(
		claims.sub,
		claims.repository,
		claims.ref,
		claims.actor,
	)
}
