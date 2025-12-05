package postgres

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	PoolSize int
	SSLMode  string
	CAFile   string
}

func NewPostgresConnection(cfg PostgresConfig) (*pgxpool.Pool, error) {
	if cfg.Port < 0 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid port number: %d, must be between 0 and 65535", cfg.Port)
	}

	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 25
	}

	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "require"
	}

	config, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("failed to create base config: %w", err)
	}

	config.ConnConfig.Host = cfg.Host
	config.ConnConfig.Port = uint16(cfg.Port)
	config.ConnConfig.User = cfg.User
	config.ConnConfig.Password = cfg.Password
	config.ConnConfig.Database = cfg.Database
	config.MaxConns = int32(cfg.PoolSize)

	switch sslMode {
	case "disable":
		config.ConnConfig.TLSConfig = nil
	case "require":
		config.ConnConfig.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	case "verify-ca":
		tlsConfig, err := createTLSConfigWithCA(cfg.CAFile)
		if err != nil {
			return nil, err
		}
		config.ConnConfig.TLSConfig = tlsConfig
	case "verify-full":
		tlsConfig, err := createTLSConfigWithCA(cfg.CAFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.ServerName = cfg.Host
		config.ConnConfig.TLSConfig = tlsConfig
	default:
		return nil, fmt.Errorf("unknown sslMode: %s", sslMode)
	}

	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

func createTLSConfigWithCA(caFile string) (*tls.Config, error) {
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		RootCAs:            certPool,
		InsecureSkipVerify: false,
	}, nil
}
