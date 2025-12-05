package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/na2na-p/cargohold/internal/domain"
)

func (c *RedisClient) SetSession(ctx context.Context, sessionID string, userInfo *domain.UserInfo, ttl time.Duration) error {
	if userInfo == nil {
		return fmt.Errorf("userInfo is nil")
	}
	if ttl == 0 {
		ttl = SessionTTL
	}

	data, err := c.serializer.Serialize(userInfo)
	if err != nil {
		return fmt.Errorf("セッション情報のシリアライズに失敗しました: %w", err)
	}

	key := SessionKey(sessionID)
	err = c.Set(ctx, key, data, ttl)
	if err != nil {
		return fmt.Errorf("セッション情報の保存に失敗しました: %w", err)
	}
	return nil
}

func (c *RedisClient) GetSession(ctx context.Context, sessionID string) (*domain.UserInfo, error) {
	key := SessionKey(sessionID)
	data, err := c.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("セッション情報の取得に失敗しました: %w", err)
	}

	userInfo, err := c.serializer.Deserialize([]byte(data))
	if err != nil {
		return nil, fmt.Errorf("セッション情報のデシリアライズに失敗しました: %w", err)
	}
	return userInfo, nil
}

func (c *RedisClient) DeleteSession(ctx context.Context, sessionID string) error {
	key := SessionKey(sessionID)
	err := c.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("セッション情報の削除に失敗しました: %w", err)
	}
	return nil
}
