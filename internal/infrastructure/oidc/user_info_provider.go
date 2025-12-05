//go:generate mockgen -source=$GOFILE -destination=../../../tests/infrastructure/oidc/mock_user_info_provider.go -package=oidc
package oidc

import (
	"context"
)

type UserInfoProvider interface {
	GetUserInfo(ctx context.Context, token *OAuthToken) (*GitHubUser, error)
}
