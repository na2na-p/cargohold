//go:generate mockgen -source=$GOFILE -destination=../../../tests/infrastructure/redis/mock_session_store_adapter.go -package=redis
package redis

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
)

var _ usecase.SessionStoreInterface = (*SessionStoreAdapter)(nil)

type SessionClient interface {
	SetSession(ctx context.Context, sessionID string, userInfo *domain.UserInfo, ttl time.Duration) error
	GetSession(ctx context.Context, sessionID string) (*domain.UserInfo, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

type UUIDGenerator interface {
	Generate() string
}

type DefaultUUIDGenerator struct{}

func (g *DefaultUUIDGenerator) Generate() string {
	return uuid.New().String()
}

type SessionStoreAdapter struct {
	client        SessionClient
	uuidGenerator UUIDGenerator
}

func NewSessionStoreAdapter(client SessionClient, uuidGenerator UUIDGenerator) *SessionStoreAdapter {
	return &SessionStoreAdapter{
		client:        client,
		uuidGenerator: uuidGenerator,
	}
}

func NewSessionStoreAdapterWithDefaults(client SessionClient) *SessionStoreAdapter {
	return &SessionStoreAdapter{
		client:        client,
		uuidGenerator: &DefaultUUIDGenerator{},
	}
}

func (a *SessionStoreAdapter) CreateSession(ctx context.Context, userInfo *domain.UserInfo, ttl time.Duration) (string, error) {
	sessionID := a.uuidGenerator.Generate()
	if err := a.client.SetSession(ctx, sessionID, userInfo, ttl); err != nil {
		return "", err
	}
	return sessionID, nil
}

func (a *SessionStoreAdapter) GetSession(ctx context.Context, sessionID string) (*domain.UserInfo, error) {
	return a.client.GetSession(ctx, sessionID)
}

func (a *SessionStoreAdapter) DeleteSession(ctx context.Context, sessionID string) error {
	return a.client.DeleteSession(ctx, sessionID)
}
