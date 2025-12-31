//go:generate mockgen -source=$GOFILE -destination=mock_token_exchanger_test.go -package=oidc
package oidc

import (
	"context"
)

type TokenExchanger interface {
	GetAuthorizationURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*oauthToken, error)
}
