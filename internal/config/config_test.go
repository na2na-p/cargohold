package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/viper"

	"github.com/na2na-p/cargohold/internal/config"
)

func TestLoad_OIDCEnabled(t *testing.T) {
	tests := []struct {
		name              string
		configContent     string
		wantGitHubEnabled bool
	}{
		{
			name: "正常系: OIDC設定が指定されていない場合、デフォルトでtrue",
			configContent: `
database:
  host: localhost
  port: 5432
  user: test
  password: test
  dbname: test
redis:
  host: localhost
  port: 6379
s3:
  endpoint: http://localhost:9000
  accesskeyid: test
  secretaccesskey: test
  bucketname: test
  region: us-east-1
`,
			wantGitHubEnabled: true,
		},
		{
			name: "正常系: GitHub OIDC enabled: true の場合",
			configContent: `
database:
  host: localhost
  port: 5432
  user: test
  password: test
  dbname: test
redis:
  host: localhost
  port: 6379
s3:
  endpoint: http://localhost:9000
  accesskeyid: test
  secretaccesskey: test
  bucketname: test
  region: us-east-1
oidc:
  github:
    enabled: true
    audience: https://github.com/test
    jwksurl: https://token.actions.githubusercontent.com/.well-known/jwks
`,
			wantGitHubEnabled: true,
		},
		{
			name: "正常系: GitHub OIDC enabled: false の場合",
			configContent: `
database:
  host: localhost
  port: 5432
  user: test
  password: test
  dbname: test
redis:
  host: localhost
  port: 6379
s3:
  endpoint: http://localhost:9000
  accesskeyid: test
  secretaccesskey: test
  bucketname: test
  region: us-east-1
oidc:
  github:
    enabled: false
    audience: https://github.com/test
    jwksurl: https://token.actions.githubusercontent.com/.well-known/jwks
`,
			wantGitHubEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// viperをリセット
			viper.Reset()

			// 一時ディレクトリとconfig.yamlを作成
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("設定ファイルの作成に失敗: %v", err)
			}

			// 作業ディレクトリを変更
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("現在のディレクトリの取得に失敗: %v", err)
			}
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("ディレクトリの変更に失敗: %v", err)
			}
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Errorf("元のディレクトリへの復帰に失敗: %v", err)
				}
			}()

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Load()がエラーを返した: %v", err)
			}

			if diff := cmp.Diff(tt.wantGitHubEnabled, cfg.OIDC.GitHub.Enabled); diff != "" {
				t.Errorf("OIDC.GitHub.Enabled mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoad_TrustedProxyCIDRs(t *testing.T) {
	tests := []struct {
		name                  string
		configContent         string
		wantTrustedProxyCIDRs []string
	}{
		{
			name: "正常系: TrustedProxyCIDRsが指定されていない場合、空スライス",
			configContent: `
database:
  host: localhost
  port: 5432
  user: test
  password: test
  dbname: test
redis:
  host: localhost
  port: 6379
s3:
  endpoint: http://localhost:9000
  accesskeyid: test
  secretaccesskey: test
  bucketname: test
  region: us-east-1
`,
			wantTrustedProxyCIDRs: nil,
		},
		{
			name: "正常系: 単一のCIDRが指定されている場合",
			configContent: `
database:
  host: localhost
  port: 5432
  user: test
  password: test
  dbname: test
redis:
  host: localhost
  port: 6379
s3:
  endpoint: http://localhost:9000
  accesskeyid: test
  secretaccesskey: test
  bucketname: test
  region: us-east-1
server:
  trustedproxycidrs:
    - 10.0.0.0/8
`,
			wantTrustedProxyCIDRs: []string{"10.0.0.0/8"},
		},
		{
			name: "正常系: 複数のCIDRが指定されている場合",
			configContent: `
database:
  host: localhost
  port: 5432
  user: test
  password: test
  dbname: test
redis:
  host: localhost
  port: 6379
s3:
  endpoint: http://localhost:9000
  accesskeyid: test
  secretaccesskey: test
  bucketname: test
  region: us-east-1
server:
  trustedproxycidrs:
    - 10.0.0.0/8
    - 172.16.0.0/12
    - 192.168.0.0/16
`,
			wantTrustedProxyCIDRs: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("設定ファイルの作成に失敗: %v", err)
			}

			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("現在のディレクトリの取得に失敗: %v", err)
			}
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("ディレクトリの変更に失敗: %v", err)
			}
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Errorf("元のディレクトリへの復帰に失敗: %v", err)
				}
			}()

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Load()がエラーを返した: %v", err)
			}

			if diff := cmp.Diff(tt.wantTrustedProxyCIDRs, cfg.Server.TrustedProxyCIDRs); diff != "" {
				t.Errorf("Server.TrustedProxyCIDRs mismatch (-want +got):\n%s", diff)
			}
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

func TestLoad_ProxyTimeout(t *testing.T) {
	tests := []struct {
		name             string
		configContent    string
		wantProxyTimeout time.Duration
	}{
		{
			name: "正常系: ProxyTimeoutが指定されていない場合、デフォルト値10分",
			configContent: `
database:
  host: localhost
  port: 5432
  user: test
  password: test
  dbname: test
redis:
  host: localhost
  port: 6379
s3:
  endpoint: http://localhost:9000
  accesskeyid: test
  secretaccesskey: test
  bucketname: test
  region: us-east-1
`,
			wantProxyTimeout: 10 * time.Minute,
		},
		{
			name: "正常系: ProxyTimeoutが5分に設定されている場合",
			configContent: `
database:
  host: localhost
  port: 5432
  user: test
  password: test
  dbname: test
redis:
  host: localhost
  port: 6379
s3:
  endpoint: http://localhost:9000
  accesskeyid: test
  secretaccesskey: test
  bucketname: test
  region: us-east-1
server:
  proxytimeout: 5m
`,
			wantProxyTimeout: 5 * time.Minute,
		},
		{
			name: "正常系: ProxyTimeoutが30秒に設定されている場合",
			configContent: `
database:
  host: localhost
  port: 5432
  user: test
  password: test
  dbname: test
redis:
  host: localhost
  port: 6379
s3:
  endpoint: http://localhost:9000
  accesskeyid: test
  secretaccesskey: test
  bucketname: test
  region: us-east-1
server:
  proxytimeout: 30s
`,
			wantProxyTimeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("設定ファイルの作成に失敗: %v", err)
			}

			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("現在のディレクトリの取得に失敗: %v", err)
			}
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("ディレクトリの変更に失敗: %v", err)
			}
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Errorf("元のディレクトリへの復帰に失敗: %v", err)
				}
			}()

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Load()がエラーを返した: %v", err)
			}

			if diff := cmp.Diff(tt.wantProxyTimeout, cfg.Server.ProxyTimeout); diff != "" {
				t.Errorf("Server.ProxyTimeout mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
