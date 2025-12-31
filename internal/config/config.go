package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type ServerConfig struct {
	TrustProxy        bool          `envconfig:"SERVER_TRUST_PROXY" default:"false"`
	TrustedProxyCIDRs []string      `envconfig:"SERVER_TRUSTED_PROXY_CIDRS"`
	ProxyTimeout      time.Duration `envconfig:"SERVER_PROXY_TIMEOUT" default:"10m"`
}

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	S3       S3Config
	OIDC     OIDCConfig
	OAuth    OAuthConfig
}

type DatabaseConfig struct {
	Host     string `envconfig:"DATABASE_HOST" required:"true"`
	Port     int    `envconfig:"DATABASE_PORT" default:"5432"`
	User     string `envconfig:"DATABASE_USER" required:"true"`
	Password string `envconfig:"DATABASE_PASSWORD" required:"true"`
	DBName   string `envconfig:"DATABASE_DBNAME" required:"true"`
	SSLMode  string `envconfig:"DATABASE_SSLMODE" default:"require"`
}

type RedisConfig struct {
	Host     string `envconfig:"REDIS_HOST" required:"true"`
	Port     int    `envconfig:"REDIS_PORT" default:"6379"`
	Password string `envconfig:"REDIS_PASSWORD"`
	DB       int    `envconfig:"REDIS_DB" default:"0"`
}

type S3Config struct {
	Endpoint        string `envconfig:"S3_ENDPOINT" required:"true"`
	AccessKeyID     string `envconfig:"S3_ACCESSKEYID" required:"true"`
	SecretAccessKey string `envconfig:"S3_SECRETACCESSKEY" required:"true"`
	BucketName      string `envconfig:"S3_BUCKETNAME" required:"true"`
	Region          string `envconfig:"S3_REGION" required:"true"`
}

type OIDCConfig struct {
	GitHub GitHubOIDCConfig
}

type GitHubOIDCConfig struct {
	Enabled  bool   `envconfig:"OIDC_GITHUB_ENABLED" default:"true"`
	Audience string `envconfig:"OIDC_GITHUB_AUDIENCE"`
	JWKSURL  string `envconfig:"OIDC_GITHUB_JWKSURL"`
}

type OAuthConfig struct {
	GitHub GitHubOAuthConfig
}

type GitHubOAuthConfig struct {
	Enabled             bool     `envconfig:"OAUTH_GITHUB_ENABLED" default:"false"`
	ClientID            string   `envconfig:"GITHUB_OAUTH_CLIENT_ID"`
	ClientSecret        string   `envconfig:"GITHUB_OAUTH_CLIENT_SECRET"`
	AllowedHosts        []string `envconfig:"OAUTH_GITHUB_ALLOWED_HOSTS"`
	AllowedRedirectURIs []string `envconfig:"OAUTH_GITHUB_ALLOWED_REDIRECT_URIS"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
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

func (c GitHubOAuthConfig) String() string {
	return fmt.Sprintf("GitHubOAuthConfig{Enabled: %t, ClientID: %s, ClientSecret: ***, AllowedHosts: %v, AllowedRedirectURIs: %v}",
		c.Enabled, c.ClientID, c.AllowedHosts, c.AllowedRedirectURIs)
}
