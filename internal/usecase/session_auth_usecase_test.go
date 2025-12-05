package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_usecase "github.com/na2na-p/cargohold/tests/usecase"
	"go.uber.org/mock/gomock"
)

func mustNewUserInfoInSessionTest(t *testing.T, sub, email, name string, provider domain.ProviderType, repository *domain.RepositoryIdentifier, ref string) *domain.UserInfo {
	t.Helper()
	userInfo, err := domain.NewUserInfo(sub, email, name, provider, repository, ref)
	if err != nil {
		t.Fatalf("failed to create UserInfo: %v", err)
	}
	return userInfo
}

func TestSessionAuthUseCase_Authenticate(t *testing.T) {
	ownerRepo, _ := domain.NewRepositoryIdentifier("owner/repo")

	type args struct {
		sessionID string
	}
	type mockFields struct {
		redisClient  *mock_usecase.MockCacheClient
		keyGenerator *mock_usecase.MockCacheKeyGenerator
	}
	tests := []struct {
		name            string
		setupMocks      func(ctrl *gomock.Controller) mockFields
		args            args
		want            *domain.UserInfo
		wantErr         error
		wantErrContains string
	}{
		{
			name: "正常系: セッションが存在する場合、ユーザー情報を返す",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				redisClient := mock_usecase.NewMockCacheClient(ctrl)
				keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
				keyGenerator.EXPECT().SessionKey("test-session-id").Return("lfs:session:test-session-id")
				redisClient.EXPECT().GetJSON(gomock.Any(), "lfs:session:test-session-id", gomock.Any()).DoAndReturn(
					func(ctx context.Context, key string, dest interface{}) error {
						sessionData := &usecase.SessionData{
							Sub:      "test-sub",
							Email:    "test@example.com",
							Name:     "Test User",
							Provider: "github",
						}
						if m, ok := dest.(*usecase.SessionData); ok {
							*m = *sessionData
						}
						return nil
					},
				)
				return mockFields{redisClient: redisClient, keyGenerator: keyGenerator}
			},
			args: args{
				sessionID: "test-session-id",
			},
			want:    mustNewUserInfoInSessionTest(t, "test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, nil, ""),
			wantErr: nil,
		},
		{
			name: "正常系: GitHub Actionsセッションが存在する場合、リポジトリ情報を含むユーザー情報を返す",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				redisClient := mock_usecase.NewMockCacheClient(ctrl)
				keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
				keyGenerator.EXPECT().SessionKey("github-session-id").Return("lfs:session:github-session-id")
				redisClient.EXPECT().GetJSON(gomock.Any(), "lfs:session:github-session-id", gomock.Any()).DoAndReturn(
					func(ctx context.Context, key string, dest interface{}) error {
						sessionData := &usecase.SessionData{
							Sub:        "repo:owner/repo:ref:refs/heads/main",
							Email:      "",
							Name:       "github-actor",
							Provider:   "github",
							Repository: "owner/repo",
							Ref:        "refs/heads/main",
						}
						if m, ok := dest.(*usecase.SessionData); ok {
							*m = *sessionData
						}
						return nil
					},
				)
				return mockFields{redisClient: redisClient, keyGenerator: keyGenerator}
			},
			args: args{
				sessionID: "github-session-id",
			},
			want: mustNewUserInfoInSessionTest(t,
				"repo:owner/repo:ref:refs/heads/main",
				"",
				"github-actor",
				domain.ProviderTypeGitHub,
				ownerRepo,
				"refs/heads/main",
			),
			wantErr: nil,
		},
		{
			name: "正常系: GitHub Actions PRセッションが存在する場合",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				redisClient := mock_usecase.NewMockCacheClient(ctrl)
				keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
				keyGenerator.EXPECT().SessionKey("github-pr-session-id").Return("lfs:session:github-pr-session-id")
				redisClient.EXPECT().GetJSON(gomock.Any(), "lfs:session:github-pr-session-id", gomock.Any()).DoAndReturn(
					func(ctx context.Context, key string, dest interface{}) error {
						sessionData := &usecase.SessionData{
							Sub:        "repo:owner/repo:ref:refs/pull/42/merge",
							Email:      "",
							Name:       "pr-author",
							Provider:   "github",
							Repository: "owner/repo",
							Ref:        "refs/pull/42/merge",
						}
						if m, ok := dest.(*usecase.SessionData); ok {
							*m = *sessionData
						}
						return nil
					},
				)
				return mockFields{redisClient: redisClient, keyGenerator: keyGenerator}
			},
			args: args{
				sessionID: "github-pr-session-id",
			},
			want: mustNewUserInfoInSessionTest(t,
				"repo:owner/repo:ref:refs/pull/42/merge",
				"",
				"pr-author",
				domain.ProviderTypeGitHub,
				ownerRepo,
				"refs/pull/42/merge",
			),
			wantErr: nil,
		},
		{
			name: "異常系: セッションが存在しない場合、ErrSessionNotFound",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				redisClient := mock_usecase.NewMockCacheClient(ctrl)
				keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
				keyGenerator.EXPECT().SessionKey("invalid-session-id").Return("lfs:session:invalid-session-id")
				redisClient.EXPECT().GetJSON(gomock.Any(), "lfs:session:invalid-session-id", gomock.Any()).Return(usecase.ErrCacheMiss)
				return mockFields{redisClient: redisClient, keyGenerator: keyGenerator}
			},
			args: args{
				sessionID: "invalid-session-id",
			},
			want:    nil,
			wantErr: usecase.ErrSessionNotFound,
		},
		{
			name: "異常系: Redis接続エラーの場合、redis errorとして返す",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				redisClient := mock_usecase.NewMockCacheClient(ctrl)
				keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
				keyGenerator.EXPECT().SessionKey("error-session-id").Return("lfs:session:error-session-id")
				redisClient.EXPECT().GetJSON(gomock.Any(), "lfs:session:error-session-id", gomock.Any()).Return(errors.New("Redis接続エラー"))
				return mockFields{redisClient: redisClient, keyGenerator: keyGenerator}
			},
			args: args{
				sessionID: "error-session-id",
			},
			want:            nil,
			wantErrContains: "redis error",
		},
		{
			name: "異常系: JSONデシリアライズエラーの場合、ErrInvalidSessionData",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				redisClient := mock_usecase.NewMockCacheClient(ctrl)
				keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
				keyGenerator.EXPECT().SessionKey("json-error-session-id").Return("lfs:session:json-error-session-id")
				redisClient.EXPECT().GetJSON(gomock.Any(), "lfs:session:json-error-session-id", gomock.Any()).Return(&json.SyntaxError{})
				return mockFields{redisClient: redisClient, keyGenerator: keyGenerator}
			},
			args: args{
				sessionID: "json-error-session-id",
			},
			want:    nil,
			wantErr: usecase.ErrInvalidSessionData,
		},
		{
			name: "異常系: セッションデータにsubが含まれていない場合、ErrInvalidSessionData",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				redisClient := mock_usecase.NewMockCacheClient(ctrl)
				keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
				keyGenerator.EXPECT().SessionKey("malformed-session-id").Return("lfs:session:malformed-session-id")
				redisClient.EXPECT().GetJSON(gomock.Any(), "lfs:session:malformed-session-id", gomock.Any()).DoAndReturn(
					func(ctx context.Context, key string, dest interface{}) error {
						sessionData := &usecase.SessionData{
							Sub:   "",
							Email: "test@example.com",
							Name:  "Test User",
						}
						if m, ok := dest.(*usecase.SessionData); ok {
							*m = *sessionData
						}
						return nil
					},
				)
				return mockFields{redisClient: redisClient, keyGenerator: keyGenerator}
			},
			args: args{
				sessionID: "malformed-session-id",
			},
			want:    nil,
			wantErr: usecase.ErrInvalidSessionData,
		},
		{
			name: "異常系: セッションデータが空の場合、ErrInvalidSessionData",
			setupMocks: func(ctrl *gomock.Controller) mockFields {
				redisClient := mock_usecase.NewMockCacheClient(ctrl)
				keyGenerator := mock_usecase.NewMockCacheKeyGenerator(ctrl)
				keyGenerator.EXPECT().SessionKey("empty-session-id").Return("lfs:session:empty-session-id")
				redisClient.EXPECT().GetJSON(gomock.Any(), "lfs:session:empty-session-id", gomock.Any()).DoAndReturn(
					func(ctx context.Context, key string, dest interface{}) error {
						sessionData := &usecase.SessionData{}
						if m, ok := dest.(*usecase.SessionData); ok {
							*m = *sessionData
						}
						return nil
					},
				)
				return mockFields{redisClient: redisClient, keyGenerator: keyGenerator}
			},
			args: args{
				sessionID: "empty-session-id",
			},
			want:    nil,
			wantErr: usecase.ErrInvalidSessionData,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mocks := tt.setupMocks(ctrl)
			uc := usecase.NewSessionAuthUseCase(mocks.redisClient, mocks.keyGenerator)

			got, err := uc.Authenticate(ctx, tt.args.sessionID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Authenticate() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf("Authenticate() error = nil, wantErrContains %q", tt.wantErrContains)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("Authenticate() error = %v, wantErrContains %q", err, tt.wantErrContains)
				}
			} else {
				if err != nil {
					t.Fatalf("Authenticate() unexpected error: %v", err)
				}
				if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(domain.UserInfo{}, domain.ProviderType{}, domain.RepositoryIdentifier{})); diff != "" {
					t.Errorf("Authenticate() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
