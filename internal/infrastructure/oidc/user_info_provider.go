//go:generate mockgen -source=$GOFILE -destination=mock_user_info_provider_test.go -package=oidc
package oidc

import (
	"context"
)

type UserInfoProvider interface {
	GetUserInfo(ctx context.Context, token *oauthToken) (*gitHubUser, error)
}
