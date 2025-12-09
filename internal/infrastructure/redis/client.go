package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisConfig はRedisクライアントの設定を保持します
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

// RedisClientInterface はRedisクライアントの操作を抽象化するインターフェース
type RedisClientInterface interface {
	Ping(ctx context.Context) error
	Close() error
}

// ClientFactory はRedisクライアントを生成するファクトリ関数の型
type ClientFactory func(opt *redis.Options) RedisClientInterface

// redisClientAdapter は *redis.Client を RedisClientInterface に適合させるアダプタ
type redisClientAdapter struct {
	client *redis.Client
}

func (a *redisClientAdapter) Ping(ctx context.Context) error {
	return a.client.Ping(ctx).Err()
}

func (a *redisClientAdapter) Close() error {
	return a.client.Close()
}

func (a *redisClientAdapter) UnwrapClient() *redis.Client {
	return a.client
}

// DefaultClientFactory はデフォルトのクライアント生成関数
func DefaultClientFactory(opt *redis.Options) RedisClientInterface {
	return &redisClientAdapter{client: redis.NewClient(opt)}
}

// RedisClient はRedisクライアントのラッパーです
type RedisClient struct {
	client     *redis.Client
	serializer UserInfoSerializer
}

// NewRedisConnectionWithFactory はファクトリを使用してRedis接続を作成する
func NewRedisConnectionWithFactory(cfg RedisConfig, factory ClientFactory) (RedisClientInterface, error) {
	if factory == nil {
		factory = DefaultClientFactory
	}

	if cfg.PoolSize == 0 {
		cfg.PoolSize = 10
	}

	client := factory(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// 初期化処理のためHTTPリクエストコンテキストが存在しないため context.Background() を使用
	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		if client != nil {
			_ = client.Close()
		}
		return nil, fmt.Errorf("redis接続に失敗しました: %w", err)
	}

	return client, nil
}

// clientUnwrapper は内部のredis.Clientを取得するためのインターフェース
type clientUnwrapper interface {
	UnwrapClient() *redis.Client
}

// NewRedisConnection は新しいRedis接続を作成します（本番用ヘルパー・後方互換）
func NewRedisConnection(cfg RedisConfig) (*redis.Client, error) {
	client, err := NewRedisConnectionWithFactory(cfg, nil)
	if err != nil {
		return nil, err
	}
	unwrapper, ok := client.(clientUnwrapper)
	if !ok {
		return nil, fmt.Errorf("クライアントが*redis.Clientを返すアダプタではありません")
	}
	return unwrapper.UnwrapClient(), nil
}

// NewRedisClient はネイティブのRedisクライアントからRedisClientを作成します（DI用）
func NewRedisClient(client *redis.Client) *RedisClient {
	return &RedisClient{
		client:     client,
		serializer: NewUserInfoSerializer(),
	}
}

// Close はRedisクライアントをクローズします
func (c *RedisClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Ping はRedisサーバーとの接続確認を行います
func (c *RedisClient) Ping(ctx context.Context) error {
	if c.client == nil {
		return errors.New("redis client is nil")
	}
	return c.client.Ping(ctx).Err()
}
