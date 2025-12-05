//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_github_oauth_interfaces.go -package=usecase
package usecase

import (
	"context"
	"time"

	"github.com/na2na-p/cargohold/internal/domain"
)

const (
	OIDCStateTTL = 10 * time.Minute
	SessionTTL   = 24 * time.Hour
)

type GitHubOAuthProviderInterface interface {
	GetAuthorizationURL(state string, scopes []string) string
	ExchangeCode(ctx context.Context, code string) (*OAuthTokenResult, error)
	GetUserInfo(ctx context.Context, token *OAuthTokenResult) (*GitHubUserResult, error)
	CanAccessRepository(ctx context.Context, token *OAuthTokenResult, repo *domain.RepositoryIdentifier) (bool, error)
}

type OAuthStateStoreInterface interface {
	SaveState(ctx context.Context, state string, data *OAuthStateData, ttl time.Duration) error
	GetAndDeleteState(ctx context.Context, state string) (*OAuthStateData, error)
}

type SessionStoreInterface interface {
	CreateSession(ctx context.Context, userInfo *domain.UserInfo, ttl time.Duration) (string, error)
	GetSession(ctx context.Context, sessionID string) (*domain.UserInfo, error)
	DeleteSession(ctx context.Context, sessionID string) error
}
