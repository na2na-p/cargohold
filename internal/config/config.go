package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// ServerConfig はHTTPサーバーの設定を保持します
type ServerConfig struct {
	TrustProxy        bool
	TrustedProxyCIDRs []string
}

// Config はアプリケーション全体の設定を保持します
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	S3       S3Config
	OIDC     OIDCConfig
}

// DatabaseConfig はデータベース接続の設定を保持します
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string // PostgreSQL SSL mode: disable, require, verify-ca, verify-full
}

// RedisConfig はRedis接続の設定を保持します
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// S3Config はS3接続の設定を保持します
type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Region          string
}

// OIDCConfig はOIDCプロバイダーの設定を保持します
type OIDCConfig struct {
	GitHub GitHubOIDCConfig
}

// GitHubOIDCConfig はGitHub OIDC設定を保持します
type GitHubOIDCConfig struct {
	Enabled  bool // OIDC認証の有効/無効フラグ
	Audience string
	JWKSURL  string
}

// Load は設定ファイルを読み込み、Config構造体を返します
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()

	viper.SetDefault("server.trustproxy", false)
	viper.SetDefault("oidc.github.enabled", true)
	viper.SetDefault("database.sslmode", "require")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c DatabaseConfig) String() string {
	return fmt.Sprintf("DatabaseConfig{Host: %s, Port: %d, User: %s, Password: ***, DBName: %s, SSLMode: %s}",
		c.Host, c.Port, c.User, c.DBName, c.SSLMode)
}

func (c RedisConfig) String() string {
	return fmt.Sprintf("RedisConfig{Host: %s, Port: %d, Password: ***, DB: %d}",
		c.Host, c.Port, c.DB)
}

func (c S3Config) String() string {
	return fmt.Sprintf("S3Config{Endpoint: %s, AccessKeyID: %s, SecretAccessKey: ***, BucketName: %s, Region: %s}",
		c.Endpoint, c.AccessKeyID, c.BucketName, c.Region)
}
