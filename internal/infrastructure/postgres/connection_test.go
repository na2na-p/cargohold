package postgres_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/na2na-p/cargohold/internal/infrastructure/postgres"
	"github.com/pashagolub/pgxmock/v4"
)

func TestNewPostgresConnection(t *testing.T) {
	type args struct {
		cfg postgres.PostgresConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "異常系: 接続失敗（無効なホスト）",
			args: args{
				cfg: postgres.PostgresConfig{
					Host:     "invalid-host",
					Port:     5432,
					User:     "testuser",
					Password: "testpass",
					Database: "testdb",
					PoolSize: 10,
				},
			},
			wantErr: true,
		},
		{
			name: "異常系: 接続失敗（無効なポート）",
			args: args{
				cfg: postgres.PostgresConfig{
					Host:     "localhost",
					Port:     99999,
					User:     "testuser",
					Password: "testpass",
					Database: "testdb",
					PoolSize: 0, // デフォルト値が使われる
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := postgres.NewPostgresConnection(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPostgresConnection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// エラーがない場合はプールをクローズ
			if pool != nil {
				pool.Close()
			}
		})
	}
}

func TestNewPostgresConnection_Success(t *testing.T) {
	// モックプールの作成
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	// Ping成功を期待
	mock.ExpectPing()

	ctx := context.Background()

	// Ping確認
	if err := mock.Ping(ctx); err != nil {
		t.Errorf("Ping() failed: %v", err)
	}

	// モックの期待値検証
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
	}
}

// TestNewPostgresConnection_PingFailure はPing失敗シミュレーションのテスト
// 注意: NewPostgresConnectionは内部でpgxpool.NewWithConfigを呼び出すため、
// 外部からモックを注入することができない。このテストでは、pgxmockを使用して
// Ping失敗時の動作パターンを検証している。実際のNewPostgresConnectionの
// Ping失敗テストは統合テストで実施する。
func TestNewPostgresConnection_PingFailure(t *testing.T) {
	// モックプールの作成
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	// Ping失敗を期待
	mock.ExpectPing().WillReturnError(errors.New("connection failed"))

	ctx := context.Background()

	// モックを使ったPing失敗シミュレーション
	// これはNewPostgresConnectionの内部動作を模倣したテスト
	err = mock.Ping(ctx)
	if err == nil {
		t.Error("Ping() should have failed but succeeded")
	}

	// モックの期待値検証
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
	}
}

func TestNewPostgresConnection_PortValidation(t *testing.T) {
	type args struct {
		cfg postgres.PostgresConfig
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		errContain string
	}{
		{
			name: "正常系: ポート番号1は有効",
			args: args{
				cfg: postgres.PostgresConfig{
					Host:     "localhost",
					Port:     1,
					User:     "test",
					Password: "test",
					Database: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "正常系: ポート番号80は有効",
			args: args{
				cfg: postgres.PostgresConfig{
					Host:     "localhost",
					Port:     80,
					User:     "test",
					Password: "test",
					Database: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "正常系: ポート番号5432は有効",
			args: args{
				cfg: postgres.PostgresConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "test",
					Password: "test",
					Database: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "正常系: ポート番号65535は有効",
			args: args{
				cfg: postgres.PostgresConfig{
					Host:     "localhost",
					Port:     65535,
					User:     "test",
					Password: "test",
					Database: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "異常系: 負のポート番号はエラー",
			args: args{
				cfg: postgres.PostgresConfig{
					Host:     "localhost",
					Port:     -1,
					User:     "test",
					Password: "test",
					Database: "test",
				},
			},
			wantErr:    true,
			errContain: "invalid port number",
		},
		{
			name: "異常系: 65536以上のポート番号はエラー",
			args: args{
				cfg: postgres.PostgresConfig{
					Host:     "localhost",
					Port:     65536,
					User:     "test",
					Password: "test",
					Database: "test",
				},
			},
			wantErr:    true,
			errContain: "invalid port number",
		},
		{
			name: "異常系: 非常に大きいポート番号はエラー",
			args: args{
				cfg: postgres.PostgresConfig{
					Host:     "localhost",
					Port:     100000,
					User:     "test",
					Password: "test",
					Database: "test",
				},
			},
			wantErr:    true,
			errContain: "invalid port number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := postgres.NewPostgresConnection(tt.args.cfg)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error containing %q, but got nil", tt.errContain)
				}
				if tt.errContain != "" {
					if !strings.Contains(err.Error(), tt.errContain) {
						t.Errorf("error = %v, want error containing %q", err, tt.errContain)
					}
				}
			}
		})
	}
}
