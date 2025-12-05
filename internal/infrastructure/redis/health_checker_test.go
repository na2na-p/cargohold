package redis_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
)

func TestRedisHealthChecker_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "正常系: 'redis'が返る",
			want: "redis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := redismock.NewClientMock()
			redisClient := redis.NewRedisClient(client)
			checker := redis.NewRedisHealthChecker(redisClient)

			got := checker.Name()

			if got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisHealthChecker_Check(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name      string
		setupMock func(mock redismock.ClientMock)
		args      args
		wantErr   bool
	}{
		{
			name: "正常系: Pingが成功した場合、nilが返る",
			setupMock: func(mock redismock.ClientMock) {
				mock.ExpectPing().SetVal("PONG")
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "異常系: Pingが失敗した場合、エラーが返る",
			setupMock: func(mock redismock.ClientMock) {
				mock.ExpectPing().SetErr(errors.New("connection refused"))
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			tt.setupMock(mock)

			redisClient := redis.NewRedisClient(client)
			checker := redis.NewRedisHealthChecker(redisClient)

			err := checker.Check(tt.args.ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("mock expectations not met: %v", err)
			}
		})
	}
}
