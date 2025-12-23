//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_session_auth_usecase.go -package=usecase
package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/na2na-p/cargohold/internal/domain"
)

type SessionData struct {
	Sub        string `json:"sub"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Provider   string `json:"provider,omitempty"`
	Repository string `json:"repository,omitempty"`
	Ref        string `json:"ref,omitempty"`
}

type SessionAuthUseCase struct {
	redisClient  domain.CacheClient
	keyGenerator domain.CacheKeyGenerator
}

func NewSessionAuthUseCase(
	redisClient domain.CacheClient,
	keyGenerator domain.CacheKeyGenerator,
) *SessionAuthUseCase {
	return &SessionAuthUseCase{
		redisClient:  redisClient,
		keyGenerator: keyGenerator,
	}
}

func (uc *SessionAuthUseCase) Authenticate(ctx context.Context, sessionID string) (*domain.UserInfo, error) {
	sessionKey := uc.keyGenerator.SessionKey(sessionID)

	var sessionData SessionData
	if err := uc.redisClient.GetJSON(ctx, sessionKey, &sessionData); err != nil {
		if errors.Is(err, ErrCacheMiss) {
			return nil, fmt.Errorf("%w: %v", ErrSessionNotFound, err)
		}

		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
			return nil, fmt.Errorf("%w: %v", ErrInvalidSessionData, err)
		}

		return nil, fmt.Errorf("redis error: %w", err)
	}

	if sessionData.Sub == "" {
		return nil, fmt.Errorf("%w: sub is required", ErrInvalidSessionData)
	}

	var repo *domain.RepositoryIdentifier
	if sessionData.Repository != "" {
		var err error
		repo, err = domain.NewRepositoryIdentifier(sessionData.Repository)
		if err != nil {
			return nil, fmt.Errorf("invalid repository: %w", err)
		}
	}

	providerType, err := domain.NewProviderType(sessionData.Provider)
	if err != nil {
		return nil, fmt.Errorf("invalid provider type: %w", err)
	}

	userInfo, err := domain.NewUserInfo(
		sessionData.Sub,
		sessionData.Email,
		sessionData.Name,
		providerType,
		repo,
		sessionData.Ref,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info: %w", err)
	}

	return userInfo, nil
}
