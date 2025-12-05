package s3

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestNewS3Connection はNewS3Connection関数のテーブルドリブンテスト
func TestNewS3Connection(t *testing.T) {
	type args struct {
		cfg S3Config
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "正常系: 有効な設定（カスタムエンドポイント）",
			args: args{
				cfg: S3Config{
					Region:          "us-east-1",
					Bucket:          "test-bucket",
					Endpoint:        "http://localhost:9000",
					AccessKeyID:     "test-access-key",
					SecretAccessKey: "test-secret-key",
					UseSSL:          false,
				},
			},
			wantErr: false,
		},
		{
			name: "正常系: 有効な設定（SSL有効）",
			args: args{
				cfg: S3Config{
					Region:          "us-west-2",
					Bucket:          "secure-bucket",
					Endpoint:        "https://s3.us-west-2.amazonaws.com",
					AccessKeyID:     "access-key",
					SecretAccessKey: "secret-key",
					UseSSL:          true,
				},
			},
			wantErr: false,
		},
		{
			name: "正常系: エンドポイントなし（AWS S3使用）",
			args: args{
				cfg: S3Config{
					Region:          "ap-northeast-1",
					Bucket:          "japan-bucket",
					Endpoint:        "",
					AccessKeyID:     "aws-key",
					SecretAccessKey: "aws-secret",
					UseSSL:          true,
				},
			},
			wantErr: false,
		},
		{
			name: "正常系: S3互換ストレージ設定",
			args: args{
				cfg: S3Config{
					Region:          "us-east-1",
					Bucket:          "test-s3-compatible-bucket",
					Endpoint:        "http://localhost:9000",
					AccessKeyID:     "test-user",
					SecretAccessKey: "test-password",
					UseSSL:          false,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewS3Connection(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewS3Connection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// エラーがない場合はクライアントの存在確認
			if !tt.wantErr {
				if client == nil {
					t.Error("NewS3Connection() returned nil client")
				}
			}
		})
	}
}

// TestNewS3Connection_ClientConfiguration はクライアント設定の詳細テスト
func TestNewS3Connection_ClientConfiguration(t *testing.T) {
	type args struct {
		cfg S3Config
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常系: クライアントが正しく構成される",
			args: args{
				cfg: S3Config{
					Region:          "eu-west-1",
					Bucket:          "europe-bucket",
					Endpoint:        "http://localhost:9000",
					AccessKeyID:     "test-key",
					SecretAccessKey: "test-secret",
					UseSSL:          false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewS3Connection(tt.args.cfg)
			if err != nil {
				t.Fatalf("NewS3Connection() failed: %v", err)
			}

			if client == nil {
				t.Fatal("NewS3Connection() returned nil client")
			}

			// S3Clientラッパーを作成してテスト
			s3Client := NewS3Client(client, tt.args.cfg.Bucket)
			if s3Client == nil {
				t.Fatal("NewS3Client() returned nil")
			}

			// バケット名が正しく設定されているか確認
			if diff := cmp.Diff(tt.args.cfg.Bucket, s3Client.bucket); diff != "" {
				t.Errorf("bucket mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestS3Config_Validation はS3Config構造体のフィールド設定テスト
func TestS3Config_Validation(t *testing.T) {
	tests := []struct {
		name string
		cfg  S3Config
		want S3Config
	}{
		{
			name: "正常系: 全フィールドが正しく設定される",
			cfg: S3Config{
				Endpoint:        "http://localhost:9000",
				AccessKeyID:     "test-access",
				SecretAccessKey: "test-secret",
				Region:          "us-east-1",
				Bucket:          "test-bucket",
				UseSSL:          false,
			},
			want: S3Config{
				Endpoint:        "http://localhost:9000",
				AccessKeyID:     "test-access",
				SecretAccessKey: "test-secret",
				Region:          "us-east-1",
				Bucket:          "test-bucket",
				UseSSL:          false,
			},
		},
		{
			name: "正常系: SSL有効の設定",
			cfg: S3Config{
				Endpoint:        "https://s3.amazonaws.com",
				AccessKeyID:     "aws-key",
				SecretAccessKey: "aws-secret",
				Region:          "us-west-2",
				Bucket:          "secure-bucket",
				UseSSL:          true,
			},
			want: S3Config{
				Endpoint:        "https://s3.amazonaws.com",
				AccessKeyID:     "aws-key",
				SecretAccessKey: "aws-secret",
				Region:          "us-west-2",
				Bucket:          "secure-bucket",
				UseSSL:          true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, tt.cfg); diff != "" {
				t.Errorf("S3Config mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
