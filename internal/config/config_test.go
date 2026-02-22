package config_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/na2na-p/cargohold/internal/config"
)

func setRequiredEnvVars(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_USER", "test")
	t.Setenv("DATABASE_DBNAME", "test")
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("S3_ACCESSKEYID", "test")
	t.Setenv("S3_SECRETACCESSKEY", "test")
	t.Setenv("S3_BUCKETNAME", "test")
	t.Setenv("S3_REGION", "us-east-1")
}

func TestLoad_Required(t *testing.T) {
	t.Run("正常系: 必須環境変数が全て設定されている場合", func(t *testing.T) {
		setRequiredEnvVars(t)

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Load()がエラーを返した: %v", err)
		}

		if cfg.Database.Host != "localhost" {
			t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "localhost")
		}
	})

	t.Run("異常系: DATABASE_HOSTが設定されていない場合", func(t *testing.T) {
		t.Setenv("DATABASE_USER", "test")
		t.Setenv("DATABASE_DBNAME", "test")
		t.Setenv("REDIS_HOST", "localhost")
		t.Setenv("S3_ENDPOINT", "http://localhost:9000")
		t.Setenv("S3_ACCESSKEYID", "test")
		t.Setenv("S3_SECRETACCESSKEY", "test")
		t.Setenv("S3_BUCKETNAME", "test")
		t.Setenv("S3_REGION", "us-east-1")

		_, err := config.Load()
		if err == nil {
			t.Fatal("Load()がエラーを返すべき")
		}
	})

	t.Run("正常系: DATABASE_PASSWORDが未設定でもロードできる（IAM認証用）", func(t *testing.T) {
		setRequiredEnvVars(t)

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Load()がエラーを返した: %v", err)
		}

		if cfg.Database.Password != "" {
			t.Errorf("Database.Password = %q, want %q", cfg.Database.Password, "")
		}
	})
}

func TestLoad_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T, cfg *config.Config)
	}{
		{
			name:    "正常系: DATABASE_PORTのデフォルト値は5432",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Database.Port != 5432 {
					t.Errorf("Database.Port = %d, want %d", cfg.Database.Port, 5432)
				}
			},
		},
		{
			name:    "正常系: DATABASE_SSLMODEのデフォルト値はrequire",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Database.SSLMode != "require" {
					t.Errorf("Database.SSLMode = %q, want %q", cfg.Database.SSLMode, "require")
				}
			},
		},
		{
			name:    "正常系: REDIS_PORTのデフォルト値は6379",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Redis.Port != 6379 {
					t.Errorf("Redis.Port = %d, want %d", cfg.Redis.Port, 6379)
				}
			},
		},
		{
			name:    "正常系: REDIS_DBのデフォルト値は0",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Redis.DB != 0 {
					t.Errorf("Redis.DB = %d, want %d", cfg.Redis.DB, 0)
				}
			},
		},
		{
			name:    "正常系: SERVER_TRUST_PROXYのデフォルト値はfalse",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Server.TrustProxy != false {
					t.Errorf("Server.TrustProxy = %t, want %t", cfg.Server.TrustProxy, false)
				}
			},
		},
		{
			name:    "正常系: SERVER_PROXY_TIMEOUTのデフォルト値は10分",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Server.ProxyTimeout != 10*time.Minute {
					t.Errorf("Server.ProxyTimeout = %v, want %v", cfg.Server.ProxyTimeout, 10*time.Minute)
				}
			},
		},
		{
			name:    "正常系: OIDC_GITHUB_ENABLEDのデフォルト値はtrue",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.OIDC.GitHub.Enabled != true {
					t.Errorf("OIDC.GitHub.Enabled = %t, want %t", cfg.OIDC.GitHub.Enabled, true)
				}
			},
		},
		{
			name:    "正常系: OAUTH_GITHUB_ENABLEDのデフォルト値はfalse",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.OAuth.GitHub.Enabled != false {
					t.Errorf("OAuth.GitHub.Enabled = %t, want %t", cfg.OAuth.GitHub.Enabled, false)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnvVars(t)
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Load()がエラーを返した: %v", err)
			}

			tt.validate(t, cfg)
		})
	}
}

func TestLoad_CustomValues(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T, cfg *config.Config)
	}{
		{
			name: "正常系: DATABASE_PORTをカスタム値に設定",
			envVars: map[string]string{
				"DATABASE_PORT": "5433",
			},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Database.Port != 5433 {
					t.Errorf("Database.Port = %d, want %d", cfg.Database.Port, 5433)
				}
			},
		},
		{
			name: "正常系: SERVER_PROXY_TIMEOUTを5分に設定",
			envVars: map[string]string{
				"SERVER_PROXY_TIMEOUT": "5m",
			},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Server.ProxyTimeout != 5*time.Minute {
					t.Errorf("Server.ProxyTimeout = %v, want %v", cfg.Server.ProxyTimeout, 5*time.Minute)
				}
			},
		},
		{
			name: "正常系: SERVER_TRUSTED_PROXY_CIDRSをカンマ区切りで設定",
			envVars: map[string]string{
				"SERVER_TRUSTED_PROXY_CIDRS": "10.0.0.0/8,172.16.0.0/12",
			},
			validate: func(t *testing.T, cfg *config.Config) {
				want := []string{"10.0.0.0/8", "172.16.0.0/12"}
				if diff := cmp.Diff(want, cfg.Server.TrustedProxyCIDRs); diff != "" {
					t.Errorf("Server.TrustedProxyCIDRs mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "正常系: OIDC_GITHUB_ENABLEDをfalseに設定",
			envVars: map[string]string{
				"OIDC_GITHUB_ENABLED": "false",
			},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.OIDC.GitHub.Enabled != false {
					t.Errorf("OIDC.GitHub.Enabled = %t, want %t", cfg.OIDC.GitHub.Enabled, false)
				}
			},
		},
		{
			name: "正常系: OAUTH_GITHUB_ALLOWED_HOSTSをカンマ区切りで設定",
			envVars: map[string]string{
				"OAUTH_GITHUB_ENABLED":       "true",
				"OAUTH_GITHUB_ALLOWED_HOSTS": "example.com,example.org",
			},
			validate: func(t *testing.T, cfg *config.Config) {
				want := []string{"example.com", "example.org"}
				if diff := cmp.Diff(want, cfg.OAuth.GitHub.AllowedHosts); diff != "" {
					t.Errorf("OAuth.GitHub.AllowedHosts mismatch (-want +got):\n%s", diff)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnvVars(t)
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Load()がエラーを返した: %v", err)
			}

			tt.validate(t, cfg)
		})
	}
}

func TestDatabaseConfig_String(t *testing.T) {
	tests := []struct {
		name   string
		config config.DatabaseConfig
		want   string
	}{
		{
			name: "正常系: パスワードがマスクされる",
			config: config.DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "testuser",
				Password: "secretpassword",
				DBName:   "testdb",
				SSLMode:  "disable",
			},
			want: "DatabaseConfig{Host: localhost, Port: 5432, User: testuser, Password: ***, DBName: testdb, SSLMode: disable}",
		},
		{
			name: "正常系: 空のパスワードでもマスクされる",
			config: config.DatabaseConfig{
				Host:     "db.example.com",
				Port:     5433,
				User:     "admin",
				Password: "",
				DBName:   "production",
				SSLMode:  "require",
			},
			want: "DatabaseConfig{Host: db.example.com, Port: 5433, User: admin, Password: ***, DBName: production, SSLMode: require}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.String()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("String() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRedisConfig_String(t *testing.T) {
	tests := []struct {
		name   string
		config config.RedisConfig
		want   string
	}{
		{
			name: "正常系: パスワードがマスクされる",
			config: config.RedisConfig{
				Host:     "localhost",
				Port:     6379,
				Password: "redispassword",
				DB:       0,
			},
			want: "RedisConfig{Host: localhost, Port: 6379, Password: ***, DB: 0}",
		},
		{
			name: "正常系: 空のパスワードでもマスクされる",
			config: config.RedisConfig{
				Host:     "redis.example.com",
				Port:     6380,
				Password: "",
				DB:       1,
			},
			want: "RedisConfig{Host: redis.example.com, Port: 6380, Password: ***, DB: 1}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.String()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("String() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestS3Config_String(t *testing.T) {
	tests := []struct {
		name   string
		config config.S3Config
		want   string
	}{
		{
			name: "正常系: SecretAccessKeyがマスクされる",
			config: config.S3Config{
				Endpoint:        "http://localhost:9000",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				BucketName:      "my-bucket",
				Region:          "us-east-1",
			},
			want: "S3Config{Endpoint: http://localhost:9000, AccessKeyID: AKIAIOSFODNN7EXAMPLE, SecretAccessKey: ***, BucketName: my-bucket, Region: us-east-1}",
		},
		{
			name: "正常系: 空のSecretAccessKeyでもマスクされる",
			config: config.S3Config{
				Endpoint:        "https://s3.amazonaws.com",
				AccessKeyID:     "AKIAI44QH8DHBEXAMPLE",
				SecretAccessKey: "",
				BucketName:      "production-bucket",
				Region:          "ap-northeast-1",
			},
			want: "S3Config{Endpoint: https://s3.amazonaws.com, AccessKeyID: AKIAI44QH8DHBEXAMPLE, SecretAccessKey: ***, BucketName: production-bucket, Region: ap-northeast-1}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.String()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("String() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGitHubOAuthConfig_String(t *testing.T) {
	tests := []struct {
		name   string
		config config.GitHubOAuthConfig
		want   string
	}{
		{
			name: "正常系: ClientSecretがマスクされる",
			config: config.GitHubOAuthConfig{
				Enabled:             true,
				ClientID:            "client-id-123",
				ClientSecret:        "super-secret",
				AllowedHosts:        []string{"example.com"},
				AllowedRedirectURIs: []string{"https://example.com/callback"},
			},
			want: "GitHubOAuthConfig{Enabled: true, ClientID: client-id-123, ClientSecret: ***, AllowedHosts: [example.com], AllowedRedirectURIs: [https://example.com/callback]}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.String()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("String() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
