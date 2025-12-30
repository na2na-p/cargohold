package redis_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
	goredis "github.com/redis/go-redis/v9"
)

type testDTO struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestRedisClient_GetDelJSON(t *testing.T) {
	type args struct {
		ctx context.Context
		key string
	}
	tests := []struct {
		name      string
		setupMock func(mock redismock.ClientMock, args args)
		args      args
		want      *testDTO
		wantErr   error
	}{
		{
			name: "正常系: キーの値を取得して削除する",
			setupMock: func(mock redismock.ClientMock, args args) {
				dto := &testDTO{Name: "test", Value: 123}
				jsonBytes, _ := json.Marshal(dto)
				mock.ExpectGetDel(args.key).SetVal(string(jsonBytes))
			},
			args: args{
				ctx: context.Background(),
				key: "test-key",
			},
			want:    &testDTO{Name: "test", Value: 123},
			wantErr: nil,
		},
		{
			name: "異常系: 存在しないキーを取得するとErrCacheMissが返る",
			setupMock: func(mock redismock.ClientMock, args args) {
				mock.ExpectGetDel(args.key).RedisNil()
			},
			args: args{
				ctx: context.Background(),
				key: "non-existent-key",
			},
			want:    nil,
			wantErr: goredis.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			tt.setupMock(mock, tt.args)

			redisClient := redis.NewRedisClient(client)
			var got testDTO
			err := redisClient.GetDelJSON(tt.args.ctx, tt.args.key, &got)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("GetDelJSON() error = nil, wantErr %v", tt.wantErr)
				}
				if err != tt.wantErr && err.Error() != tt.wantErr.Error() {
					t.Errorf("GetDelJSON() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetDelJSON() unexpected error = %v", err)
			}

			if diff := cmp.Diff(tt.want, &got); diff != "" {
				t.Errorf("GetDelJSON() mismatch (-want +got):\n%s", diff)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("mock expectations not met: %v", err)
			}
		})
	}
}
