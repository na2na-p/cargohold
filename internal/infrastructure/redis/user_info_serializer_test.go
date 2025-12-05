package redis_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
)

func mustNewUserInfo(t *testing.T, sub, email, name string, provider domain.ProviderType, repository *domain.RepositoryIdentifier, ref string) *domain.UserInfo {
	t.Helper()
	userInfo, err := domain.NewUserInfo(sub, email, name, provider, repository, ref)
	if err != nil {
		t.Fatalf("failed to create UserInfo: %v", err)
	}
	return userInfo
}

func TestUserInfoSerializerImpl_Serialize(t *testing.T) {
	ownerRepo, _ := domain.NewRepositoryIdentifier("owner/repo")

	type userInfoData struct {
		sub        string
		email      string
		name       string
		provider   domain.ProviderType
		repository *domain.RepositoryIdentifier
		ref        string
	}
	tests := []struct {
		name         string
		userInfoData *userInfoData
		wantErr      error
	}{
		{
			name: "正常系: 全フィールドが設定されたUserInfoをシリアライズ",
			userInfoData: &userInfoData{
				sub:        "user123",
				email:      "test@example.com",
				name:       "Test User",
				provider:   domain.ProviderTypeGitHub,
				repository: ownerRepo,
				ref:        "refs/heads/main",
			},
			wantErr: nil,
		},
		{
			name: "正常系: optionalフィールドが空のUserInfoをシリアライズ",
			userInfoData: &userInfoData{
				sub:        "user456",
				email:      "test2@example.com",
				name:       "Test User 2",
				provider:   domain.ProviderTypeGitHub,
				repository: nil,
				ref:        "",
			},
			wantErr: nil,
		},
		{
			name:         "異常系: nilのUserInfoをシリアライズ",
			userInfoData: nil,
			wantErr:      redis.ErrNilUserInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := redis.NewUserInfoSerializer()

			var userInfo *domain.UserInfo
			if tt.userInfoData != nil {
				userInfo = mustNewUserInfo(t, tt.userInfoData.sub, tt.userInfoData.email, tt.userInfoData.name, tt.userInfoData.provider, tt.userInfoData.repository, tt.userInfoData.ref)
			}

			_, err := serializer.Serialize(userInfo)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if err != tt.wantErr {
					t.Errorf("error mismatch: want %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}
		})
	}
}

func TestUserInfoSerializerImpl_Deserialize(t *testing.T) {
	ownerRepo, _ := domain.NewRepositoryIdentifier("owner/repo")

	type userInfoData struct {
		sub        string
		email      string
		name       string
		provider   domain.ProviderType
		repository *domain.RepositoryIdentifier
		ref        string
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name     string
		args     args
		wantData *userInfoData
		wantErr  bool
	}{
		{
			name: "正常系: 全フィールドが設定されたJSONをデシリアライズ",
			args: args{
				data: []byte(`{"sub":"user123","email":"test@example.com","name":"Test User","provider":"github","repository":"owner/repo","ref":"refs/heads/main"}`),
			},
			wantData: &userInfoData{
				sub:        "user123",
				email:      "test@example.com",
				name:       "Test User",
				provider:   domain.ProviderTypeGitHub,
				repository: ownerRepo,
				ref:        "refs/heads/main",
			},
			wantErr: false,
		},
		{
			name: "正常系: optionalフィールドが省略されたJSONをデシリアライズ",
			args: args{
				data: []byte(`{"sub":"user456","email":"test2@example.com","name":"Test User 2","provider":"github"}`),
			},
			wantData: &userInfoData{
				sub:        "user456",
				email:      "test2@example.com",
				name:       "Test User 2",
				provider:   domain.ProviderTypeGitHub,
				repository: nil,
				ref:        "",
			},
			wantErr: false,
		},
		{
			name: "異常系: 不正なJSONをデシリアライズ",
			args: args{
				data: []byte(`{invalid json}`),
			},
			wantData: nil,
			wantErr:  true,
		},
		{
			name: "異常系: 空のデータをデシリアライズ",
			args: args{
				data: []byte{},
			},
			wantData: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := redis.NewUserInfoSerializer()
			got, err := serializer.Deserialize(tt.args.data)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error, but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
				want := mustNewUserInfo(t, tt.wantData.sub, tt.wantData.email, tt.wantData.name, tt.wantData.provider, tt.wantData.repository, tt.wantData.ref)
				if diff := cmp.Diff(want, got, cmp.AllowUnexported(domain.UserInfo{}, domain.ProviderType{}, domain.RepositoryIdentifier{})); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestUserInfoSerializerImpl_RoundTrip(t *testing.T) {
	ownerRepo, _ := domain.NewRepositoryIdentifier("owner/repo")

	type userInfoData struct {
		sub        string
		email      string
		name       string
		provider   domain.ProviderType
		repository *domain.RepositoryIdentifier
		ref        string
	}
	tests := []struct {
		name         string
		userInfoData userInfoData
	}{
		{
			name: "正常系: 全フィールドが設定されたUserInfoのラウンドトリップ",
			userInfoData: userInfoData{
				sub:        "user123",
				email:      "test@example.com",
				name:       "Test User",
				provider:   domain.ProviderTypeGitHub,
				repository: ownerRepo,
				ref:        "refs/heads/main",
			},
		},
		{
			name: "正常系: optionalフィールドが空のUserInfoのラウンドトリップ",
			userInfoData: userInfoData{
				sub:        "user456",
				email:      "test2@example.com",
				name:       "Test User 2",
				provider:   domain.ProviderTypeGitHub,
				repository: nil,
				ref:        "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := redis.NewUserInfoSerializer()
			userInfo := mustNewUserInfo(t, tt.userInfoData.sub, tt.userInfoData.email, tt.userInfoData.name, tt.userInfoData.provider, tt.userInfoData.repository, tt.userInfoData.ref)

			data, err := serializer.Serialize(userInfo)
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			got, err := serializer.Deserialize(data)
			if err != nil {
				t.Fatalf("Deserialize failed: %v", err)
			}

			if diff := cmp.Diff(userInfo, got, cmp.AllowUnexported(domain.UserInfo{}, domain.ProviderType{}, domain.RepositoryIdentifier{})); diff != "" {
				t.Errorf("round trip mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGitHubUserInfoSerializerImpl_Serialize(t *testing.T) {
	type args struct {
		userInfo *domain.GitHubUserInfo
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "正常系: 全フィールドが設定されたGitHubUserInfoをシリアライズ",
			args: args{
				userInfo: domain.NewGitHubUserInfo(
					"repo:owner/repo:ref:refs/heads/main",
					"owner/repo",
					"refs/heads/main",
					"test-actor",
				),
			},
			wantErr: nil,
		},
		{
			name: "異常系: nilのGitHubUserInfoをシリアライズ",
			args: args{
				userInfo: nil,
			},
			wantErr: redis.ErrNilGitHubUserInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := redis.NewGitHubUserInfoSerializer()
			_, err := serializer.Serialize(tt.args.userInfo)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if err != tt.wantErr {
					t.Errorf("error mismatch: want %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}
		})
	}
}

func TestGitHubUserInfoSerializerImpl_Deserialize(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *domain.GitHubUserInfo
		wantErr bool
	}{
		{
			name: "正常系: 全フィールドが設定されたJSONをデシリアライズ",
			args: args{
				data: []byte(`{"sub":"repo:owner/repo:ref:refs/heads/main","repository":"owner/repo","ref":"refs/heads/main","actor":"test-actor"}`),
			},
			want: domain.NewGitHubUserInfo(
				"repo:owner/repo:ref:refs/heads/main",
				"owner/repo",
				"refs/heads/main",
				"test-actor",
			),
			wantErr: false,
		},
		{
			name: "異常系: 不正なJSONをデシリアライズ",
			args: args{
				data: []byte(`{invalid json}`),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "異常系: 空のデータをデシリアライズ",
			args: args{
				data: []byte{},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := redis.NewGitHubUserInfoSerializer()
			got, err := serializer.Deserialize(tt.args.data)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error, but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
				if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(domain.GitHubUserInfo{})); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGitHubUserInfoSerializerImpl_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		userInfo *domain.GitHubUserInfo
	}{
		{
			name: "正常系: GitHubUserInfoのラウンドトリップ",
			userInfo: domain.NewGitHubUserInfo(
				"repo:owner/repo:ref:refs/heads/main",
				"owner/repo",
				"refs/heads/main",
				"test-actor",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := redis.NewGitHubUserInfoSerializer()

			data, err := serializer.Serialize(tt.userInfo)
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			got, err := serializer.Deserialize(data)
			if err != nil {
				t.Fatalf("Deserialize failed: %v", err)
			}

			if diff := cmp.Diff(tt.userInfo, got, cmp.AllowUnexported(domain.GitHubUserInfo{})); diff != "" {
				t.Errorf("round trip mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
