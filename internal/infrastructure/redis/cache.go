package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCacheMiss はキャッシュにキーが存在しない場合のセンチネルエラーです
var ErrCacheMiss = redis.Nil

// Get は指定されたキーの値を取得します
func (c *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrCacheMiss
	}
	if err != nil {
		return "", fmt.Errorf("キーの取得に失敗しました: %w", err)
	}
	return val, nil
}

// Set は指定されたキーに値を設定します
func (c *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	err := c.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("キーの設定に失敗しました: %w", err)
	}
	return nil
}

// Delete は指定されたキーを削除します
func (c *RedisClient) Delete(ctx context.Context, key string) error {
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("キーの削除に失敗しました: %w", err)
	}
	return nil
}

// Exists は指定されたキーが存在するかを確認します
func (c *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("キーの存在確認に失敗しました: %w", err)
	}
	return result > 0, nil
}

// SetJSON は指定されたキーにJSON形式で値を設定します
func (c *RedisClient) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("JSONシリアライズに失敗しました: %w", err)
	}

	err = c.client.Set(ctx, key, jsonBytes, ttl).Err()
	if err != nil {
		return fmt.Errorf("キーの設定に失敗しました: %w", err)
	}
	return nil
}

// GetJSON は指定されたキーの値をJSON形式で取得します
func (c *RedisClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("キーの取得に失敗しました: %w", err)
	}

	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return fmt.Errorf("JSONデシリアライズに失敗しました: %w", err)
	}
	return nil
}

func (c *RedisClient) GetDelJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := c.client.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("キーの取得と削除に失敗しました: %w", err)
	}

	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return fmt.Errorf("JSONデシリアライズに失敗しました: %w", err)
	}
	return nil
}
