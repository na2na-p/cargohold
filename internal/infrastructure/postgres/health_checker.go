package postgres

import (
	"context"
	"fmt"
)

// Pinger はデータベース接続のヘルスチェックを行うインターフェース
type Pinger interface {
	Ping(ctx context.Context) error
}

// PostgresHealthChecker はPostgreSQLのヘルスチェックを行う
type PostgresHealthChecker struct {
	pool Pinger
}

// NewPostgresHealthChecker は新しいPostgresHealthCheckerを生成する
func NewPostgresHealthChecker(pool Pinger) *PostgresHealthChecker {
	return &PostgresHealthChecker{
		pool: pool,
	}
}

// Name はチェッカーの名前を返す
func (c *PostgresHealthChecker) Name() string {
	return "postgres"
}

// Check はPostgreSQLへのヘルスチェックを実行する
func (c *PostgresHealthChecker) Check(ctx context.Context) error {
	if err := c.pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres health check failed: %w", err)
	}
	return nil
}
