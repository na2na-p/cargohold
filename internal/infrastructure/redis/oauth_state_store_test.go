package redis_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
)

type oauthStateDTO struct {
	Repository  string `json:"repository"`
	RedirectURI string `json:"redirect_uri"`
	Shell       string `json:"shell,omitempty"`
}

func TestOAuthStateStore_SaveState(t *testing.T) {
	type args struct {
		ctx   context.Context
		state string
		data  *domain.OAuthState
		ttl   time.Duration
	}
	tests := []struct {
		name      string
		setupMock func(mock redismock.ClientMock, args args)
		args      args
		wantErr   bool
	}{
		{
			name: "正常系: stateデータが正常に保存される",
			setupMock: func(mock redismock.ClientMock, args args) {
				key := redis.OIDCStateKey(args.state)
				dto := &oauthStateDTO{
					Repository:  args.data.Repository(),
					RedirectURI: args.data.RedirectURI(),
				}
				jsonBytes, _ := json.Marshal(dto)
				mock.ExpectSet(key, jsonBytes, args.ttl).SetVal("OK")
			},
			args: args{
				ctx:   context.Background(),
				state: "test-state-123",
				data:  domain.NewOAuthState("owner/repo", "https://example.com/callback", ""),
				ttl:   redis.OIDCStateTTL,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			tt.setupMock(mock, tt.args)

			redisClient := redis.NewRedisClient(client)
			store := redis.NewOAuthStateStore(redisClient)

			err := store.SaveState(tt.args.ctx, tt.args.state, tt.args.data, tt.args.ttl)

			if (err != nil) != tt.wantErr {
				t.Errorf("SaveState() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("mock expectations not met: %v", err)
			}
		})
	}
}

func TestOAuthStateStore_GetAndDeleteState(t *testing.T) {
	type args struct {
		ctx   context.Context
		state string
	}
	tests := []struct {
		name      string
		setupMock func(mock redismock.ClientMock, args args)
		args      args
		want      *domain.OAuthState
		wantErr   bool
	}{
		{
			name: "正常系: stateデータがアトミックに取得・削除される",
			setupMock: func(mock redismock.ClientMock, args args) {
				dto := &oauthStateDTO{
					Repository:  "owner/repo",
					RedirectURI: "https://example.com/callback",
				}
				key := redis.OIDCStateKey(args.state)
				jsonBytes, _ := json.Marshal(dto)
				mock.ExpectGetDel(key).SetVal(string(jsonBytes))
			},
			args: args{
				ctx:   context.Background(),
				state: "test-state-123",
			},
			want:    domain.NewOAuthState("owner/repo", "https://example.com/callback", ""),
			wantErr: false,
		},
		{
			name: "異常系: 存在しないstateを取得するとエラー",
			setupMock: func(mock redismock.ClientMock, args args) {
				key := redis.OIDCStateKey(args.state)
				mock.ExpectGetDel(key).RedisNil()
			},
			args: args{
				ctx:   context.Background(),
				state: "non-existent-state",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			tt.setupMock(mock, tt.args)

			redisClient := redis.NewRedisClient(client)
			store := redis.NewOAuthStateStore(redisClient)

			got, err := store.GetAndDeleteState(tt.args.ctx, tt.args.state)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAndDeleteState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			opts := cmp.Options{
				cmp.Comparer(func(a, b *domain.OAuthState) bool {
					if a == nil && b == nil {
						return true
					}
					if a == nil || b == nil {
						return false
					}
					return a.Repository() == b.Repository() && a.RedirectURI() == b.RedirectURI() && a.Shell() == b.Shell()
				}),
			}
			if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
				t.Errorf("GetAndDeleteState() mismatch (-want +got):\n%s", diff)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("mock expectations not met: %v", err)
			}
		})
	}
}
