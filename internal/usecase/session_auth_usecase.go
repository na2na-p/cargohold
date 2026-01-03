//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_session_auth_usecase.go -package=usecase
package usecase

import (
	"context"
	"fmt"

	"github.com/na2na-p/cargohold/internal/domain"
)

type SessionClient interface {
	GetSession(ctx context.Context, sessionID string) (*domain.UserInfo, error)
}

type SessionAuthUseCase struct {
	sessionClient SessionClient
}

func NewSessionAuthUseCase(sessionClient SessionClient) *SessionAuthUseCase {
	return &SessionAuthUseCase{
		sessionClient: sessionClient,
	}
}

func (uc *SessionAuthUseCase) Authenticate(ctx context.Context, sessionID string) (*domain.UserInfo, error) {
	userInfo, err := uc.sessionClient.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSessionNotFound, err)
	}
	return userInfo, nil
}
