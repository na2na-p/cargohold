package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/na2na-p/cargohold/internal/domain"
)

type oauthStateDTO struct {
	Repository  string `json:"repository"`
	RedirectURI string `json:"redirect_uri"`
	Shell       string `json:"shell,omitempty"`
}

type OAuthStateStore struct {
	client *RedisClient
}

func NewOAuthStateStore(client *RedisClient) *OAuthStateStore {
	return &OAuthStateStore{
		client: client,
	}
}

func (s *OAuthStateStore) SaveState(ctx context.Context, state string, data *domain.OAuthState, ttl time.Duration) error {
	key := OIDCStateKey(state)
	dto := &oauthStateDTO{
		Repository:  data.Repository(),
		RedirectURI: data.RedirectURI(),
		Shell:       data.Shell().String(),
	}
	err := s.client.SetJSON(ctx, key, dto, ttl)
	if err != nil {
		return fmt.Errorf("OAuth state の保存に失敗しました: %w", err)
	}
	return nil
}

func (s *OAuthStateStore) GetAndDeleteState(ctx context.Context, state string) (*domain.OAuthState, error) {
	key := OIDCStateKey(state)

	var dto oauthStateDTO
	err := s.client.GetDelJSON(ctx, key, &dto)
	if err != nil {
		return nil, fmt.Errorf("OAuth state の取得と削除に失敗しました: %w", err)
	}

	var shellType domain.ShellType
	if dto.Shell != "" {
		shellType, _ = domain.ParseShellType(dto.Shell)
	}
	return domain.NewOAuthState(dto.Repository, dto.RedirectURI, shellType), nil
}
