//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_oidc_provider.go -package=usecase
package usecase

import (
	"context"

	"github.com/na2na-p/cargohold/internal/domain"
)

type OIDCProvider interface {
	VerifyIDToken(ctx context.Context, idToken string) (*domain.UserInfo, error)
	GetAuthURL(ctx context.Context, state string) (string, error)
	ExchangeCode(ctx context.Context, code string) (*domain.Token, error)
}
