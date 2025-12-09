package redis

import (
	"context"
	"fmt"
)

// RedisHealthChecker はRedisのヘルスチェックを行う
type RedisHealthChecker struct {
	client *RedisClient
}

// NewRedisHealthChecker は新しいRedisHealthCheckerを生成する
func NewRedisHealthChecker(client *RedisClient) *RedisHealthChecker {
	return &RedisHealthChecker{
		client: client,
	}
}

// Name はチェッカーの名前を返す
func (c *RedisHealthChecker) Name() string {
	return "redis"
}

// Check はRedisへのヘルスチェックを実行する
func (c *RedisHealthChecker) Check(ctx context.Context) error {
	if err := c.client.Ping(ctx); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}
	return nil
}
