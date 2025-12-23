package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
	mockredis "github.com/na2na-p/cargohold/tests/infrastructure/redis"
	"go.uber.org/mock/gomock"
)

func TestSessionStoreAdapter_CreateSession(t *testing.T) {
	type fields struct {
		setupSessionClient func(ctrl *gomock.Controller) *mockredis.MockSessionClient
		setupUUIDGenerator func(ctrl *gomock.Controller) *mockredis.MockUUIDGenerator
	}
	type args struct {
		ctx      context.Context
		userInfo *domain.UserInfo
		ttl      time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr error
	}{
		{
			name: "正常系: セッションIDが返却される",
			fields: fields{
				setupSessionClient: func(ctrl *gomock.Controller) *mockredis.MockSessionClient {
					m := mockredis.NewMockSessionClient(ctrl)
					m.EXPECT().SetSession(gomock.Any(), "test-uuid", gomock.Any(), 24*time.Hour).Return(nil)
					return m
				},
				setupUUIDGenerator: func(ctrl *gomock.Controller) *mockredis.MockUUIDGenerator {
					m := mockredis.NewMockUUIDGenerator(ctrl)
					m.EXPECT().Generate().Return("test-uuid")
					return m
				},
			},
			args: args{
				ctx:      context.Background(),
				userInfo: mustCreateUserInfo(t, "sub123", "test@example.com", "Test User"),
				ttl:      24 * time.Hour,
			},
			want:    "test-uuid",
			wantErr: nil,
		},
		{
			name: "異常系: SetSessionがエラーを返す場合、エラーが返る",
			fields: fields{
				setupSessionClient: func(ctrl *gomock.Controller) *mockredis.MockSessionClient {
					m := mockredis.NewMockSessionClient(ctrl)
					m.EXPECT().SetSession(gomock.Any(), "test-uuid", gomock.Any(), 24*time.Hour).Return(errors.New("redis error"))
					return m
				},
				setupUUIDGenerator: func(ctrl *gomock.Controller) *mockredis.MockUUIDGenerator {
					m := mockredis.NewMockUUIDGenerator(ctrl)
					m.EXPECT().Generate().Return("test-uuid")
					return m
				},
			},
			args: args{
				ctx:      context.Background(),
				userInfo: mustCreateUserInfo(t, "sub123", "test@example.com", "Test User"),
				ttl:      24 * time.Hour,
			},
			want:    "",
			wantErr: errors.New("redis error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockClient := tt.fields.setupSessionClient(ctrl)
			mockUUID := tt.fields.setupUUIDGenerator(ctrl)

			adapter := redis.NewSessionStoreAdapter(mockClient, mockUUID)

			got, err := adapter.CreateSession(tt.args.ctx, tt.args.userInfo, tt.args.ttl)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSessionStoreAdapter_GetSession(t *testing.T) {
	type fields struct {
		setupSessionClient func(ctrl *gomock.Controller) *mockredis.MockSessionClient
	}
	type args struct {
		ctx       context.Context
		sessionID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domain.UserInfo
		wantErr error
	}{
		{
			name: "正常系: セッションが取得できる",
			fields: fields{
				setupSessionClient: func(ctrl *gomock.Controller) *mockredis.MockSessionClient {
					m := mockredis.NewMockSessionClient(ctrl)
					userInfo := mustCreateUserInfo(t, "sub123", "test@example.com", "Test User")
					m.EXPECT().GetSession(gomock.Any(), "session-123").Return(userInfo, nil)
					return m
				},
			},
			args: args{
				ctx:       context.Background(),
				sessionID: "session-123",
			},
			want:    mustCreateUserInfo(t, "sub123", "test@example.com", "Test User"),
			wantErr: nil,
		},
		{
			name: "異常系: GetSessionがエラーを返す場合、エラーが返る",
			fields: fields{
				setupSessionClient: func(ctrl *gomock.Controller) *mockredis.MockSessionClient {
					m := mockredis.NewMockSessionClient(ctrl)
					m.EXPECT().GetSession(gomock.Any(), "session-123").Return(nil, errors.New("session not found"))
					return m
				},
			},
			args: args{
				ctx:       context.Background(),
				sessionID: "session-123",
			},
			want:    nil,
			wantErr: errors.New("session not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockClient := tt.fields.setupSessionClient(ctrl)
			mockUUID := mockredis.NewMockUUIDGenerator(ctrl)

			adapter := redis.NewSessionStoreAdapter(mockClient, mockUUID)

			got, err := adapter.GetSession(tt.args.ctx, tt.args.sessionID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}

			if tt.want != nil && got != nil {
				if got.Sub() != tt.want.Sub() || got.Email() != tt.want.Email() || got.Name() != tt.want.Name() {
					t.Errorf("got = %+v, want = %+v", got, tt.want)
				}
			} else if (tt.want == nil) != (got == nil) {
				t.Errorf("got = %v, want = %v", got, tt.want)
			}
		})
	}
}

func TestSessionStoreAdapter_DeleteSession(t *testing.T) {
	type fields struct {
		setupSessionClient func(ctrl *gomock.Controller) *mockredis.MockSessionClient
	}
	type args struct {
		ctx       context.Context
		sessionID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "正常系: セッションが削除できる",
			fields: fields{
				setupSessionClient: func(ctrl *gomock.Controller) *mockredis.MockSessionClient {
					m := mockredis.NewMockSessionClient(ctrl)
					m.EXPECT().DeleteSession(gomock.Any(), "session-123").Return(nil)
					return m
				},
			},
			args: args{
				ctx:       context.Background(),
				sessionID: "session-123",
			},
			wantErr: nil,
		},
		{
			name: "異常系: DeleteSessionがエラーを返す場合、エラーが返る",
			fields: fields{
				setupSessionClient: func(ctrl *gomock.Controller) *mockredis.MockSessionClient {
					m := mockredis.NewMockSessionClient(ctrl)
					m.EXPECT().DeleteSession(gomock.Any(), "session-123").Return(errors.New("delete failed"))
					return m
				},
			},
			args: args{
				ctx:       context.Background(),
				sessionID: "session-123",
			},
			wantErr: errors.New("delete failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockClient := tt.fields.setupSessionClient(ctrl)
			mockUUID := mockredis.NewMockUUIDGenerator(ctrl)

			adapter := redis.NewSessionStoreAdapter(mockClient, mockUUID)

			err := adapter.DeleteSession(tt.args.ctx, tt.args.sessionID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}
		})
	}
}

func mustCreateUserInfo(t *testing.T, sub, email, name string) *domain.UserInfo {
	t.Helper()
	userInfo, err := domain.NewUserInfo(sub, email, name, domain.ProviderTypeGitHub, nil, "")
	if err != nil {
		t.Fatalf("failed to create UserInfo: %v", err)
	}
	return userInfo
}
