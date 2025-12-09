//go:generate mockgen -source=$GOFILE -destination=../../../tests/infrastructure/oidc/mock_token_exchanger.go -package=oidc
package oidc

import (
	"context"
)

type TokenExchanger interface {
	GetAuthorizationURL(state string, scopes []string) string
	ExchangeCode(ctx context.Context, code string) (*OAuthToken, error)
}
