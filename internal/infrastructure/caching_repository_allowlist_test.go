package infrastructure_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/infrastructure"
	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
	mockdomain "github.com/na2na-p/cargohold/tests/domain"
	"go.uber.org/mock/gomock"
)

func TestCachingRepositoryAllowlist_IsAllowed(t *testing.T) {
	type args struct {
		owner string
		repo  string
	}
	type fields struct {
		pgRepo func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository
	}
	tests := []struct {
		name           string
		args           args
		fields         fields
		redisMockSetup func(mock redismock.ClientMock, args args)
		wantAllowed    bool
		wantErr        bool
	}{
		{
			name: "正常系: Redisキャッシュにヒット（許可済み）",
			args: args{
				owner: "owner",
				repo:  "repo",
			},
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					return m
				},
			},
			redisMockSetup: func(mock redismock.ClientMock, args args) {
				cacheKey := "lfs:oidc:github:repo:" + args.owner + "/" + args.repo
				mock.ExpectGet(cacheKey).SetVal("true")
			},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name: "正常系: Redisキャッシュにヒット（許可されていない）",
			args: args{
				owner: "unknown",
				repo:  "repo",
			},
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					return m
				},
			},
			redisMockSetup: func(mock redismock.ClientMock, args args) {
				cacheKey := "lfs:oidc:github:repo:" + args.owner + "/" + args.repo
				mock.ExpectGet(cacheKey).SetVal("false")
			},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name: "正常系: Redisキャッシュミス、PostgreSQLで許可",
			args: args{
				owner: "owner",
				repo:  "repo-not-cached",
			},
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().IsAllowed(gomock.Any(), gomock.Any()).Return(true, nil)
					return m
				},
			},
			redisMockSetup: func(mock redismock.ClientMock, args args) {
				cacheKey := "lfs:oidc:github:repo:" + args.owner + "/" + args.repo
				mock.ExpectGet(cacheKey).RedisNil()
				mock.ExpectSet(cacheKey, "true", 5*time.Minute).SetVal("OK")
			},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name: "正常系: Redisキャッシュミス、PostgreSQLで非許可",
			args: args{
				owner: "owner",
				repo:  "unauthorized-repo",
			},
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().IsAllowed(gomock.Any(), gomock.Any()).Return(false, nil)
					return m
				},
			},
			redisMockSetup: func(mock redismock.ClientMock, args args) {
				cacheKey := "lfs:oidc:github:repo:" + args.owner + "/" + args.repo
				mock.ExpectGet(cacheKey).RedisNil()
				mock.ExpectSet(cacheKey, "false", 5*time.Minute).SetVal("OK")
			},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name: "正常系: Redis障害時のフォールバック（PostgreSQL成功）",
			args: args{
				owner: "owner",
				repo:  "repo-redis-down",
			},
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().IsAllowed(gomock.Any(), gomock.Any()).Return(true, nil)
					return m
				},
			},
			redisMockSetup: func(mock redismock.ClientMock, args args) {
				cacheKey := "lfs:oidc:github:repo:" + args.owner + "/" + args.repo
				mock.ExpectGet(cacheKey).SetErr(errors.New("redis connection error"))
				mock.ExpectSet(cacheKey, "true", 5*time.Minute).SetErr(errors.New("redis connection error"))
			},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name: "異常系: PostgreSQLエラー",
			args: args{
				owner: "owner",
				repo:  "repo-pg-error",
			},
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().IsAllowed(gomock.Any(), gomock.Any()).Return(false, errors.New("database error"))
					return m
				},
			},
			redisMockSetup: func(mock redismock.ClientMock, args args) {
				cacheKey := "lfs:oidc:github:repo:" + args.owner + "/" + args.repo
				mock.ExpectGet(cacheKey).RedisNil()
			},
			wantAllowed: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.redisMockSetup != nil {
				tt.redisMockSetup(mock, tt.args)
			}

			redisClient := redis.NewRedisClient(db)
			pgRepo := tt.fields.pgRepo(ctrl, tt.args)

			allowlist := infrastructure.NewCachingRepositoryAllowlist(pgRepo, redisClient)
			ctx := context.Background()

			allowedRepo := mustNewAllowedRepository(t, tt.args.owner, tt.args.repo)
			allowed, err := allowlist.IsAllowed(ctx, allowedRepo)

			if (err != nil) != tt.wantErr {
				t.Errorf("IsAllowed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if allowed != tt.wantAllowed {
				t.Errorf("IsAllowed() = %v, want %v", allowed, tt.wantAllowed)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

func TestCachingRepositoryAllowlist_Add(t *testing.T) {
	type args struct {
		owner string
		repo  string
	}
	type fields struct {
		pgRepo func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository
	}
	tests := []struct {
		name           string
		args           args
		fields         fields
		redisMockSetup func(mock redismock.ClientMock, args args)
		wantErr        bool
	}{
		{
			name: "正常系: リポジトリの追加に成功",
			args: args{
				owner: "owner",
				repo:  "new-repo",
			},
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)
					return m
				},
			},
			redisMockSetup: func(mock redismock.ClientMock, args args) {
				cacheKey := "lfs:oidc:github:repo:" + args.owner + "/" + args.repo
				mock.ExpectSet(cacheKey, "true", 5*time.Minute).SetVal("OK")
			},
			wantErr: false,
		},
		{
			name: "正常系: リポジトリ追加成功（Redis障害時もエラーなし）",
			args: args{
				owner: "owner",
				repo:  "repo-redis-fail",
			},
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)
					return m
				},
			},
			redisMockSetup: func(mock redismock.ClientMock, args args) {
				cacheKey := "lfs:oidc:github:repo:" + args.owner + "/" + args.repo
				mock.ExpectSet(cacheKey, "true", 5*time.Minute).SetErr(errors.New("redis error"))
			},
			wantErr: false,
		},
		{
			name: "異常系: PostgreSQLエラー",
			args: args{
				owner: "owner",
				repo:  "repo-pg-error",
			},
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().Add(gomock.Any(), gomock.Any()).Return(errors.New("database error"))
					return m
				},
			},
			redisMockSetup: func(mock redismock.ClientMock, args args) {
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.redisMockSetup != nil {
				tt.redisMockSetup(mock, tt.args)
			}

			redisClient := redis.NewRedisClient(db)
			pgRepo := tt.fields.pgRepo(ctrl, tt.args)

			allowlist := infrastructure.NewCachingRepositoryAllowlist(pgRepo, redisClient)
			ctx := context.Background()

			allowedRepo := mustNewAllowedRepository(t, tt.args.owner, tt.args.repo)
			err := allowlist.Add(ctx, allowedRepo)

			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

func TestCachingRepositoryAllowlist_Remove(t *testing.T) {
	type args struct {
		owner string
		repo  string
	}
	type fields struct {
		pgRepo func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository
	}
	tests := []struct {
		name           string
		args           args
		fields         fields
		redisMockSetup func(mock redismock.ClientMock, args args)
		wantErr        bool
	}{
		{
			name: "正常系: リポジトリの削除に成功",
			args: args{
				owner: "owner",
				repo:  "repo-to-remove",
			},
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)
					return m
				},
			},
			redisMockSetup: func(mock redismock.ClientMock, args args) {
				cacheKey := "lfs:oidc:github:repo:" + args.owner + "/" + args.repo
				mock.ExpectDel(cacheKey).SetVal(1)
			},
			wantErr: false,
		},
		{
			name: "正常系: リポジトリ削除成功（Redis障害時もエラーなし）",
			args: args{
				owner: "owner",
				repo:  "repo-redis-fail",
			},
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)
					return m
				},
			},
			redisMockSetup: func(mock redismock.ClientMock, args args) {
				cacheKey := "lfs:oidc:github:repo:" + args.owner + "/" + args.repo
				mock.ExpectDel(cacheKey).SetErr(errors.New("redis error"))
			},
			wantErr: false,
		},
		{
			name: "異常系: PostgreSQLエラー",
			args: args{
				owner: "owner",
				repo:  "repo-pg-error",
			},
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller, args args) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(errors.New("database error"))
					return m
				},
			},
			redisMockSetup: func(mock redismock.ClientMock, args args) {
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.redisMockSetup != nil {
				tt.redisMockSetup(mock, tt.args)
			}

			redisClient := redis.NewRedisClient(db)
			pgRepo := tt.fields.pgRepo(ctrl, tt.args)

			allowlist := infrastructure.NewCachingRepositoryAllowlist(pgRepo, redisClient)
			ctx := context.Background()

			allowedRepo := mustNewAllowedRepository(t, tt.args.owner, tt.args.repo)
			err := allowlist.Remove(ctx, allowedRepo)

			if (err != nil) != tt.wantErr {
				t.Errorf("Remove() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

func TestCachingRepositoryAllowlist_List(t *testing.T) {
	type fields struct {
		pgRepo func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository
	}
	tests := []struct {
		name    string
		fields  fields
		want    []*domain.AllowedRepository
		wantErr bool
	}{
		{
			name: "正常系: リポジトリ一覧の取得に成功",
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().List(gomock.Any()).Return([]*domain.AllowedRepository{
						mustNewAllowedRepositoryHelper("owner", "repo1"),
						mustNewAllowedRepositoryHelper("owner", "repo2"),
						mustNewAllowedRepositoryHelper("other", "repo3"),
					}, nil)
					return m
				},
			},
			want: []*domain.AllowedRepository{
				mustNewAllowedRepositoryHelper("owner", "repo1"),
				mustNewAllowedRepositoryHelper("owner", "repo2"),
				mustNewAllowedRepositoryHelper("other", "repo3"),
			},
			wantErr: false,
		},
		{
			name: "正常系: 空のリスト",
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().List(gomock.Any()).Return([]*domain.AllowedRepository{}, nil)
					return m
				},
			},
			want:    []*domain.AllowedRepository{},
			wantErr: false,
		},
		{
			name: "異常系: PostgreSQLエラー",
			fields: fields{
				pgRepo: func(ctrl *gomock.Controller) *mockdomain.MockRepositoryAllowlistRepository {
					m := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)
					m.EXPECT().List(gomock.Any()).Return(nil, errors.New("database error"))
					return m
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			redisClient := redis.NewRedisClient(db)
			pgRepo := tt.fields.pgRepo(ctrl)

			allowlist := infrastructure.NewCachingRepositoryAllowlist(pgRepo, redisClient)
			ctx := context.Background()

			got, err := allowlist.List(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("List() returned %d items, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].String() != tt.want[i].String() {
					t.Errorf("List()[%d] = %s, want %s", i, got[i].String(), tt.want[i].String())
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

func TestCachingRepositoryAllowlist_CustomTTL(t *testing.T) {
	type args struct {
		owner string
		repo  string
	}
	tests := []struct {
		name           string
		ttl            time.Duration
		args           args
		redisMockSetup func(mock redismock.ClientMock, args args, ttl time.Duration)
	}{
		{
			name: "正常系: カスタムTTL（1分）でのキャッシュ",
			ttl:  1 * time.Minute,
			args: args{
				owner: "owner",
				repo:  "repo-custom-ttl",
			},
			redisMockSetup: func(mock redismock.ClientMock, args args, ttl time.Duration) {
				cacheKey := "lfs:oidc:github:repo:" + args.owner + "/" + args.repo
				mock.ExpectGet(cacheKey).RedisNil()
				mock.ExpectSet(cacheKey, "true", ttl).SetVal("OK")
			},
		},
		{
			name: "正常系: カスタムTTL（10分）でのキャッシュ",
			ttl:  10 * time.Minute,
			args: args{
				owner: "owner",
				repo:  "repo-long-ttl",
			},
			redisMockSetup: func(mock redismock.ClientMock, args args, ttl time.Duration) {
				cacheKey := "lfs:oidc:github:repo:" + args.owner + "/" + args.repo
				mock.ExpectGet(cacheKey).RedisNil()
				mock.ExpectSet(cacheKey, "false", ttl).SetVal("OK")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.redisMockSetup != nil {
				tt.redisMockSetup(mock, tt.args, tt.ttl)
			}

			redisClient := redis.NewRedisClient(db)
			pgRepo := mockdomain.NewMockRepositoryAllowlistRepository(ctrl)

			if tt.ttl == 1*time.Minute {
				pgRepo.EXPECT().IsAllowed(gomock.Any(), gomock.Any()).Return(true, nil)
			} else {
				pgRepo.EXPECT().IsAllowed(gomock.Any(), gomock.Any()).Return(false, nil)
			}

			allowlist := infrastructure.NewCachingRepositoryAllowlistWithTTL(pgRepo, redisClient, tt.ttl)
			ctx := context.Background()

			allowedRepo := mustNewAllowedRepository(t, tt.args.owner, tt.args.repo)
			_, err := allowlist.IsAllowed(ctx, allowedRepo)
			if err != nil {
				t.Errorf("IsAllowed() error = %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

func mustNewAllowedRepository(t *testing.T, owner, repo string) *domain.AllowedRepository {
	t.Helper()
	ar, err := domain.NewAllowedRepository(owner, repo)
	if err != nil {
		t.Fatalf("failed to create AllowedRepository: %v", err)
	}
	return ar
}

func mustNewAllowedRepositoryHelper(owner, repo string) *domain.AllowedRepository {
	ar, err := domain.NewAllowedRepository(owner, repo)
	if err != nil {
		panic(err)
	}
	return ar
}
