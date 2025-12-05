package redis

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/redis/go-redis/v9"
)

func mustNewUserInfo(t *testing.T, sub, email, name string, provider domain.ProviderType, repository *domain.RepositoryIdentifier, ref string) *domain.UserInfo {
	t.Helper()
	userInfo, err := domain.NewUserInfo(sub, email, name, provider, repository, ref)
	if err != nil {
		t.Fatalf("failed to create UserInfo: %v", err)
	}
	return userInfo
}

// TestRedisClientImpl_Ping はPing処理のテーブルドリブンテスト
func TestRedisClientImpl_Ping(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(mock redismock.ClientMock)
		wantErr   bool
	}{
		{
			name: "正常系: Pingに成功",
			mockSetup: func(mock redismock.ClientMock) {
				mock.ExpectPing().SetVal("PONG")
			},
			wantErr: false,
		},
		{
			name: "異常系: Pingに失敗",
			mockSetup: func(mock redismock.ClientMock) {
				mock.ExpectPing().SetErr(redis.ErrClosed)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			err := client.Ping(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRedisClientImpl_Set はSet処理のテーブルドリブンテスト
func TestRedisClientImpl_Set(t *testing.T) {
	type args struct {
		key   string
		value string
		ttl   time.Duration
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock redismock.ClientMock, args args)
		wantErr   bool
	}{
		{
			name: "正常系: TTL無しでSetに成功",
			args: args{
				key:   "test:key1",
				value: "test value",
				ttl:   0,
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectSet(args.key, args.value, args.ttl).SetVal("OK")
			},
			wantErr: false,
		},
		{
			name: "正常系: TTL付きでSetに成功",
			args: args{
				key:   "test:key2",
				value: "test value with ttl",
				ttl:   5 * time.Minute,
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectSet(args.key, args.value, args.ttl).SetVal("OK")
			},
			wantErr: false,
		},
		{
			name: "正常系: 空の値をSet",
			args: args{
				key:   "test:key3",
				value: "",
				ttl:   0,
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectSet(args.key, args.value, args.ttl).SetVal("OK")
			},
			wantErr: false,
		},
		{
			name: "異常系: Setに失敗",
			args: args{
				key:   "test:key4",
				value: "test value",
				ttl:   0,
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectSet(args.key, args.value, args.ttl).SetErr(redis.ErrClosed)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.args)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			err := client.Set(ctx, tt.args.key, tt.args.value, tt.args.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRedisClientImpl_Get はGet処理のテーブルドリブンテスト
func TestRedisClientImpl_Get(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock redismock.ClientMock, args args)
		wantValue string
		wantErr   bool
	}{
		{
			name: "正常系: Getに成功",
			args: args{
				key: "test:key1",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectGet(args.key).SetVal("test value")
			},
			wantValue: "test value",
			wantErr:   false,
		},
		{
			name: "異常系: 存在しないキーのGet",
			args: args{
				key: "test:nonexistent",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectGet(args.key).SetErr(redis.Nil)
			},
			wantValue: "",
			wantErr:   true,
		},
		{
			name: "異常系: Redisエラー",
			args: args{
				key: "test:key2",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectGet(args.key).SetErr(redis.ErrClosed)
			},
			wantValue: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.args)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			result, err := client.Get(ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if diff := cmp.Diff(tt.wantValue, result); diff != "" {
					t.Errorf("Get() mismatch (-want +got):\n%s", diff)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRedisClientImpl_Exists は存在確認のテーブルドリブンテスト
func TestRedisClientImpl_Exists(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name       string
		args       args
		mockSetup  func(mock redismock.ClientMock, args args)
		wantExists bool
		wantErr    bool
	}{
		{
			name: "正常系: キーが存在する",
			args: args{
				key: "test:key1",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectExists(args.key).SetVal(1)
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "正常系: キーが存在しない",
			args: args{
				key: "test:nonexistent",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectExists(args.key).SetVal(0)
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "異常系: Redisエラー",
			args: args{
				key: "test:key2",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectExists(args.key).SetErr(redis.ErrClosed)
			},
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.args)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			exists, err := client.Exists(ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if diff := cmp.Diff(tt.wantExists, exists); diff != "" {
					t.Errorf("Exists() mismatch (-want +got):\n%s", diff)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRedisClientImpl_Delete はDelete処理のテーブルドリブンテスト
func TestRedisClientImpl_Delete(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock redismock.ClientMock, args args)
		wantErr   bool
	}{
		{
			name: "正常系: Deleteに成功",
			args: args{
				key: "test:key1",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectDel(args.key).SetVal(1)
			},
			wantErr: false,
		},
		{
			name: "正常系: 存在しないキーのDelete",
			args: args{
				key: "test:nonexistent",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectDel(args.key).SetVal(0)
			},
			wantErr: false,
		},
		{
			name: "異常系: Redisエラー",
			args: args{
				key: "test:key2",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectDel(args.key).SetErr(redis.ErrClosed)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.args)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			err := client.Delete(ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRedisClientImpl_SetWithTTL はTTL付きSetのテーブルドリブンテスト
func TestRedisClientImpl_SetWithTTL(t *testing.T) {
	type args struct {
		key   string
		value string
		ttl   time.Duration
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock redismock.ClientMock, args args)
		wantErr   bool
	}{
		{
			name: "正常系: TTL付きSetに成功",
			args: args{
				key:   "test:ttl",
				value: "test value",
				ttl:   2 * time.Second,
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectSet(args.key, args.value, args.ttl).SetVal("OK")
			},
			wantErr: false,
		},
		{
			name: "異常系: TTL付きSetに失敗",
			args: args{
				key:   "test:ttl",
				value: "test value",
				ttl:   2 * time.Second,
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectSet(args.key, args.value, args.ttl).SetErr(redis.ErrClosed)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.args)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			err := client.Set(ctx, tt.args.key, tt.args.value, tt.args.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRedisClientImpl_SetJSON はSetJSON処理のテーブルドリブンテスト
func TestRedisClientImpl_SetJSON(t *testing.T) {
	type TestData struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	type args struct {
		key  string
		data TestData
		ttl  time.Duration
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock redismock.ClientMock, args args)
		wantErr   bool
	}{
		{
			name: "正常系: SetJSONに成功",
			args: args{
				key:  "test:json1",
				data: TestData{ID: 123, Name: "Test User"},
				ttl:  0,
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				jsonData, _ := json.Marshal(args.data)
				mock.ExpectSet(args.key, jsonData, args.ttl).SetVal("OK")
			},
			wantErr: false,
		},
		{
			name: "正常系: TTL付きSetJSON",
			args: args{
				key:  "test:json2",
				data: TestData{ID: 456, Name: "Another User"},
				ttl:  5 * time.Minute,
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				jsonData, _ := json.Marshal(args.data)
				mock.ExpectSet(args.key, jsonData, args.ttl).SetVal("OK")
			},
			wantErr: false,
		},
		{
			name: "異常系: SetJSONに失敗",
			args: args{
				key:  "test:json3",
				data: TestData{ID: 789, Name: "Third User"},
				ttl:  0,
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				jsonData, _ := json.Marshal(args.data)
				mock.ExpectSet(args.key, jsonData, args.ttl).SetErr(redis.ErrClosed)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.args)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			err := client.SetJSON(ctx, tt.args.key, tt.args.data, tt.args.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRedisClientImpl_GetJSON はGetJSON処理のテーブルドリブンテスト
func TestRedisClientImpl_GetJSON(t *testing.T) {
	type TestData struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	type args struct {
		key string
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock redismock.ClientMock, args args)
		want      TestData
		wantErr   bool
	}{
		{
			name: "正常系: GetJSONに成功",
			args: args{
				key: "test:json1",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				data := TestData{ID: 123, Name: "Test User"}
				jsonData, _ := json.Marshal(data)
				mock.ExpectGet(args.key).SetVal(string(jsonData))
			},
			want:    TestData{ID: 123, Name: "Test User"},
			wantErr: false,
		},
		{
			name: "異常系: 存在しないキーのGetJSON",
			args: args{
				key: "test:json_nonexistent",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectGet(args.key).SetErr(redis.Nil)
			},
			want:    TestData{},
			wantErr: true,
		},
		{
			name: "異常系: Redisエラー",
			args: args{
				key: "test:json2",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				mock.ExpectGet(args.key).SetErr(redis.ErrClosed)
			},
			want:    TestData{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.args)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			var result TestData
			err := client.GetJSON(ctx, tt.args.key, &result)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if diff := cmp.Diff(tt.want, result); diff != "" {
					t.Errorf("GetJSON() mismatch (-want +got):\n%s", diff)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRedisClientImpl_SetSession はSetSession処理のテーブルドリブンテスト
func TestRedisClientImpl_SetSession(t *testing.T) {
	serializer := NewUserInfoSerializer()
	type userInfoData struct {
		sub      string
		email    string
		name     string
		provider domain.ProviderType
	}
	type args struct {
		sessionID    string
		userInfoData userInfoData
		ttl          time.Duration
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock redismock.ClientMock, args args, userInfo *domain.UserInfo)
		wantErr   bool
	}{
		{
			name: "正常系: SetSessionに成功",
			args: args{
				sessionID:    "test-session-1",
				userInfoData: userInfoData{sub: "user123", email: "test@example.com", name: "Test User", provider: domain.ProviderTypeGitHub},
				ttl:          0,
			},
			mockSetup: func(mock redismock.ClientMock, args args, userInfo *domain.UserInfo) {
				key := "lfs:session:" + args.sessionID
				jsonData, _ := serializer.Serialize(userInfo)
				expectedTTL := 24 * time.Hour
				mock.ExpectSet(key, jsonData, expectedTTL).SetVal("OK")
			},
			wantErr: false,
		},
		{
			name: "正常系: TTL付きSetSession",
			args: args{
				sessionID:    "test-session-2",
				userInfoData: userInfoData{sub: "user456", email: "test2@example.com", name: "Test User 2", provider: domain.ProviderTypeGitHub},
				ttl:          30 * time.Minute,
			},
			mockSetup: func(mock redismock.ClientMock, args args, userInfo *domain.UserInfo) {
				key := "lfs:session:" + args.sessionID
				jsonData, _ := serializer.Serialize(userInfo)
				mock.ExpectSet(key, jsonData, args.ttl).SetVal("OK")
			},
			wantErr: false,
		},
		{
			name: "異常系: SetSessionに失敗",
			args: args{
				sessionID:    "test-session-3",
				userInfoData: userInfoData{sub: "user789", email: "test3@example.com", name: "Test User 3", provider: domain.ProviderTypeGitHub},
				ttl:          0,
			},
			mockSetup: func(mock redismock.ClientMock, args args, userInfo *domain.UserInfo) {
				key := "lfs:session:" + args.sessionID
				jsonData, _ := serializer.Serialize(userInfo)
				expectedTTL := 24 * time.Hour
				mock.ExpectSet(key, jsonData, expectedTTL).SetErr(redis.ErrClosed)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			userInfo := mustNewUserInfo(t, tt.args.userInfoData.sub, tt.args.userInfoData.email, tt.args.userInfoData.name, tt.args.userInfoData.provider, nil, "")

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.args, userInfo)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			err := client.SetSession(ctx, tt.args.sessionID, userInfo, tt.args.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetSession() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRedisClientImpl_GetSession はGetSession処理のテーブルドリブンテスト
func TestRedisClientImpl_GetSession(t *testing.T) {
	serializer := NewUserInfoSerializer()
	type args struct {
		sessionID string
	}
	type userInfoData struct {
		sub      string
		email    string
		name     string
		provider domain.ProviderType
	}
	tests := []struct {
		name         string
		args         args
		userInfoData *userInfoData
		mockSetup    func(mock redismock.ClientMock, args args, userInfo *domain.UserInfo)
		wantErr      bool
	}{
		{
			name: "正常系: GetSessionに成功",
			args: args{
				sessionID: "test-session-1",
			},
			userInfoData: &userInfoData{sub: "user123", email: "test@example.com", name: "Test User", provider: domain.ProviderTypeGitHub},
			mockSetup: func(mock redismock.ClientMock, args args, userInfo *domain.UserInfo) {
				key := "lfs:session:" + args.sessionID
				jsonData, _ := serializer.Serialize(userInfo)
				mock.ExpectGet(key).SetVal(string(jsonData))
			},
			wantErr: false,
		},
		{
			name: "異常系: 存在しないセッションのGet",
			args: args{
				sessionID: "test-session-nonexistent",
			},
			userInfoData: nil,
			mockSetup: func(mock redismock.ClientMock, args args, userInfo *domain.UserInfo) {
				key := "lfs:session:" + args.sessionID
				mock.ExpectGet(key).SetErr(redis.Nil)
			},
			wantErr: true,
		},
		{
			name: "異常系: Redisエラー",
			args: args{
				sessionID: "test-session-2",
			},
			userInfoData: nil,
			mockSetup: func(mock redismock.ClientMock, args args, userInfo *domain.UserInfo) {
				key := "lfs:session:" + args.sessionID
				mock.ExpectGet(key).SetErr(redis.ErrClosed)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			var userInfo *domain.UserInfo
			var want *domain.UserInfo
			if tt.userInfoData != nil {
				userInfo = mustNewUserInfo(t, tt.userInfoData.sub, tt.userInfoData.email, tt.userInfoData.name, tt.userInfoData.provider, nil, "")
				want = mustNewUserInfo(t, tt.userInfoData.sub, tt.userInfoData.email, tt.userInfoData.name, tt.userInfoData.provider, nil, "")
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.args, userInfo)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			result, err := client.GetSession(ctx, tt.args.sessionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if diff := cmp.Diff(want, result, cmp.AllowUnexported(domain.UserInfo{}, domain.ProviderType{}, domain.RepositoryIdentifier{})); diff != "" {
					t.Errorf("GetSession() mismatch (-want +got):\n%s", diff)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRedisClientImpl_DeleteSession はDeleteSession処理のテーブルドリブンテスト
func TestRedisClientImpl_DeleteSession(t *testing.T) {
	type args struct {
		sessionID string
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock redismock.ClientMock, args args)
		wantErr   bool
	}{
		{
			name: "正常系: DeleteSessionに成功",
			args: args{
				sessionID: "test-session-1",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				key := "lfs:session:" + args.sessionID
				mock.ExpectDel(key).SetVal(1)
			},
			wantErr: false,
		},
		{
			name: "正常系: 存在しないセッションのDelete",
			args: args{
				sessionID: "test-session-nonexistent",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				key := "lfs:session:" + args.sessionID
				mock.ExpectDel(key).SetVal(0)
			},
			wantErr: false,
		},
		{
			name: "異常系: Redisエラー",
			args: args{
				sessionID: "test-session-2",
			},
			mockSetup: func(mock redismock.ClientMock, args args) {
				key := "lfs:session:" + args.sessionID
				mock.ExpectDel(key).SetErr(redis.ErrClosed)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.args)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			err := client.DeleteSession(ctx, tt.args.sessionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteSession() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRedisClientImpl_SessionWithCustomTTL はカスタムTTLのセッション処理テスト
func TestRedisClientImpl_SessionWithCustomTTL(t *testing.T) {
	serializer := NewUserInfoSerializer()
	type userInfoData struct {
		sub      string
		email    string
		name     string
		provider domain.ProviderType
	}
	type args struct {
		sessionID    string
		userInfoData userInfoData
		ttl          time.Duration
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock redismock.ClientMock, args args, userInfo *domain.UserInfo)
		wantErr   bool
	}{
		{
			name: "正常系: カスタムTTLでのセッション管理",
			args: args{
				sessionID:    "test-session-ttl",
				userInfoData: userInfoData{sub: "user789", email: "test3@example.com", name: "Test User 3", provider: domain.ProviderTypeGitHub},
				ttl:          2 * time.Second,
			},
			mockSetup: func(mock redismock.ClientMock, args args, userInfo *domain.UserInfo) {
				key := "lfs:session:" + args.sessionID
				jsonData, _ := serializer.Serialize(userInfo)
				mock.ExpectSet(key, jsonData, args.ttl).SetVal("OK")
			},
			wantErr: false,
		},
		{
			name: "異常系: カスタムTTLでのセッション管理失敗",
			args: args{
				sessionID:    "test-session-ttl-fail",
				userInfoData: userInfoData{sub: "user000", email: "test4@example.com", name: "Test User 4", provider: domain.ProviderTypeGitHub},
				ttl:          2 * time.Second,
			},
			mockSetup: func(mock redismock.ClientMock, args args, userInfo *domain.UserInfo) {
				key := "lfs:session:" + args.sessionID
				jsonData, _ := serializer.Serialize(userInfo)
				mock.ExpectSet(key, jsonData, args.ttl).SetErr(redis.ErrClosed)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			userInfo := mustNewUserInfo(t, tt.args.userInfoData.sub, tt.args.userInfoData.email, tt.args.userInfoData.name, tt.args.userInfoData.provider, nil, "")

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.args, userInfo)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			err := client.SetSession(ctx, tt.args.sessionID, userInfo, tt.args.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetSession() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRedisClientImpl_MultipleOperations は複数キー操作のテーブルドリブンテスト
func TestRedisClientImpl_MultipleOperations(t *testing.T) {
	type keyValue struct {
		key   string
		value string
	}

	tests := []struct {
		name      string
		data      []keyValue
		mockSetup func(mock redismock.ClientMock, data []keyValue)
	}{
		{
			name: "正常系: 複数キーの操作",
			data: []keyValue{
				{"test:multi1", "value1"},
				{"test:multi2", "value2"},
				{"test:multi3", "value3"},
			},
			mockSetup: func(mock redismock.ClientMock, data []keyValue) {
				// 先にすべてのSet操作を期待
				for _, kv := range data {
					mock.ExpectSet(kv.key, kv.value, time.Duration(0)).SetVal("OK")
				}
				// 次にすべてのGet操作を期待
				for _, kv := range data {
					mock.ExpectGet(kv.key).SetVal(kv.value)
				}
			},
		},
		{
			name: "正常系: 大量キーの操作",
			data: []keyValue{
				{"test:batch1", "data1"},
				{"test:batch2", "data2"},
				{"test:batch3", "data3"},
				{"test:batch4", "data4"},
				{"test:batch5", "data5"},
			},
			mockSetup: func(mock redismock.ClientMock, data []keyValue) {
				// 先にすべてのSet操作を期待
				for _, kv := range data {
					mock.ExpectSet(kv.key, kv.value, time.Duration(0)).SetVal("OK")
				}
				// 次にすべてのGet操作を期待
				for _, kv := range data {
					mock.ExpectGet(kv.key).SetVal(kv.value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			defer func() { _ = db.Close() }()

			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.data)
			}

			client := NewRedisClient(db)
			ctx := context.Background()

			// 全てのキーをSet
			for _, kv := range tt.data {
				if err := client.Set(ctx, kv.key, kv.value, 0); err != nil {
					t.Fatalf("Setに失敗しました (key=%s): %v", kv.key, err)
				}
			}

			// 全てのキーが取得できることを確認
			for _, kv := range tt.data {
				result, err := client.Get(ctx, kv.key)
				if err != nil {
					t.Fatalf("Getに失敗しました (key=%s): %v", kv.key, err)
				}
				if diff := cmp.Diff(kv.value, result); diff != "" {
					t.Errorf("値の不一致 (key=%s) (-want +got):\n%s", kv.key, diff)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// mockRedisClient はテスト用のモッククライアント
type mockRedisClient struct {
	pingErr error
	closed  bool
}

func (m *mockRedisClient) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *mockRedisClient) Close() error {
	m.closed = true
	return nil
}

// TestNewRedisConnectionWithFactory はNewRedisConnectionWithFactory処理のテスト
func TestNewRedisConnectionWithFactory(t *testing.T) {
	type args struct {
		cfg     RedisConfig
		factory ClientFactory
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantClosed bool
	}{
		{
			name: "正常系: Ping成功時",
			args: args{
				cfg: RedisConfig{
					Host:     "localhost",
					Port:     6379,
					Password: "",
					DB:       0,
					PoolSize: 10,
				},
				factory: func(opt *redis.Options) RedisClientInterface {
					return &mockRedisClient{pingErr: nil}
				},
			},
			wantErr:    false,
			wantClosed: false,
		},
		{
			name: "異常系: Ping失敗時のエラーとCloseが呼ばれる",
			args: args{
				cfg: RedisConfig{
					Host:     "localhost",
					Port:     6379,
					Password: "",
					DB:       0,
					PoolSize: 10,
				},
				factory: nil,
			},
			wantErr:    true,
			wantClosed: true,
		},
		{
			name: "正常系: デフォルトプールサイズ適用",
			args: args{
				cfg: RedisConfig{
					Host:     "localhost",
					Port:     6379,
					Password: "",
					DB:       0,
					PoolSize: 0,
				},
				factory: func(opt *redis.Options) RedisClientInterface {
					return &mockRedisClient{pingErr: nil}
				},
			},
			wantErr:    false,
			wantClosed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var createdMock *mockRedisClient
			factory := tt.args.factory
			if tt.wantClosed {
				createdMock = &mockRedisClient{pingErr: errors.New("connection refused")}
				factory = func(opt *redis.Options) RedisClientInterface {
					return createdMock
				}
			}
			_, err := NewRedisConnectionWithFactory(tt.args.cfg, factory)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRedisConnectionWithFactory() error = %v, wantErr %v", err, tt.wantErr)
			}
			if createdMock != nil && createdMock.closed != tt.wantClosed {
				t.Errorf("client.closed = %v, want %v", createdMock.closed, tt.wantClosed)
			}
		})
	}
}

// TestNewRedisClient はNewRedisClient処理のテスト
func TestNewRedisClient(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer func() { _ = db.Close() }()

	mock.ExpectPing().SetVal("PONG")

	client := NewRedisClient(db)
	if client == nil {
		t.Fatal("NewRedisClient() returned nil")
	}

	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping() failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
