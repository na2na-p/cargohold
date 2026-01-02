package domain_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewUserInfo(t *testing.T) {
	validRepo, _ := domain.NewRepositoryIdentifier("octocat/hello-world")

	type args struct {
		sub        string
		email      string
		name       string
		provider   domain.ProviderType
		repository *domain.RepositoryIdentifier
		ref        string
	}
	tests := []struct {
		name    string
		args    args
		wantSub string
		wantErr error
	}{
		{
			name: "正常系: 全フィールドが設定された場合、UserInfoが生成される",
			args: args{
				sub:        "repo:octocat/hello-world:ref:refs/heads/main",
				email:      "test@example.com",
				name:       "github-actions",
				provider:   domain.ProviderTypeGitHub,
				repository: validRepo,
				ref:        "refs/heads/main",
			},
			wantSub: "repo:octocat/hello-world:ref:refs/heads/main",
			wantErr: nil,
		},
		{
			name: "正常系: repositoryがnilでも生成できる",
			args: args{
				sub:        "user123",
				email:      "test@example.com",
				name:       "testuser",
				provider:   domain.ProviderTypeGitHub,
				repository: nil,
				ref:        "",
			},
			wantSub: "user123",
			wantErr: nil,
		},
		{
			name: "異常系: subが空の場合、ErrEmptySubが返される",
			args: args{
				sub:        "",
				email:      "test@example.com",
				name:       "testuser",
				provider:   domain.ProviderTypeGitHub,
				repository: nil,
				ref:        "",
			},
			wantSub: "",
			wantErr: domain.ErrEmptySub,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewUserInfo(
				tt.args.sub,
				tt.args.email,
				tt.args.name,
				tt.args.provider,
				tt.args.repository,
				tt.args.ref,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error mismatch: want %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("want no error, but got %v", err)
			}

			if diff := cmp.Diff(tt.wantSub, got.Sub()); diff != "" {
				t.Errorf("UserInfo.Sub() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUserInfo_Permissions(t *testing.T) {
	tests := []struct {
		name            string
		permissions     *domain.RepositoryPermissions
		wantCanUpload   bool
		wantCanDownload bool
		wantNil         bool
	}{
		{
			name:            "正常系: 権限がnilの場合、nilを返す",
			permissions:     nil,
			wantCanUpload:   false,
			wantCanDownload: false,
			wantNil:         true,
		},
		{
			name: "正常系: push権限がある場合、Upload可能",
			permissions: func() *domain.RepositoryPermissions {
				p := domain.NewRepositoryPermissions(false, true, false, false, false)
				return &p
			}(),
			wantCanUpload:   true,
			wantCanDownload: true,
			wantNil:         false,
		},
		{
			name: "正常系: pull権限のみの場合、Downloadのみ可能",
			permissions: func() *domain.RepositoryPermissions {
				p := domain.NewRepositoryPermissions(false, false, true, false, false)
				return &p
			}(),
			wantCanUpload:   false,
			wantCanDownload: true,
			wantNil:         false,
		},
		{
			name: "正常系: admin権限がある場合、全て可能",
			permissions: func() *domain.RepositoryPermissions {
				p := domain.NewRepositoryPermissions(true, false, false, false, false)
				return &p
			}(),
			wantCanUpload:   true,
			wantCanDownload: true,
			wantNil:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userInfo, err := domain.NewUserInfo(
				"test-sub",
				"test@example.com",
				"testuser",
				domain.ProviderTypeGitHub,
				nil,
				"",
			)
			if err != nil {
				t.Fatalf("NewUserInfo failed: %v", err)
			}

			userInfo.SetPermissions(tt.permissions)
			got := userInfo.Permissions()

			if tt.wantNil {
				if got != nil {
					t.Errorf("UserInfo.Permissions() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("UserInfo.Permissions() = nil, want non-nil")
			}

			if got.CanUpload() != tt.wantCanUpload {
				t.Errorf("Permissions.CanUpload() = %v, want %v", got.CanUpload(), tt.wantCanUpload)
			}
			if got.CanDownload() != tt.wantCanDownload {
				t.Errorf("Permissions.CanDownload() = %v, want %v", got.CanDownload(), tt.wantCanDownload)
			}
		})
	}
}

func TestUserInfo_Repository(t *testing.T) {
	validRepo, _ := domain.NewRepositoryIdentifier("octocat/hello-world")
	anotherRepo, _ := domain.NewRepositoryIdentifier("user123/repo456")

	type args struct {
		sub        string
		email      string
		name       string
		provider   domain.ProviderType
		repository *domain.RepositoryIdentifier
		ref        string
	}
	tests := []struct {
		name      string
		args      args
		wantOwner string
		wantName  string
		wantNil   bool
	}{
		{
			name: "正常系: 有効なRepositoryIdentifierを取得できる",
			args: args{
				sub:        "repo:octocat/hello-world:ref:refs/heads/main",
				email:      "",
				name:       "github-actions",
				provider:   domain.ProviderTypeGitHub,
				repository: validRepo,
				ref:        "refs/heads/main",
			},
			wantOwner: "octocat",
			wantName:  "hello-world",
			wantNil:   false,
		},
		{
			name: "正常系: 数字を含むrepository",
			args: args{
				sub:        "user123",
				email:      "test@example.com",
				name:       "testuser",
				provider:   domain.ProviderTypeGitHub,
				repository: anotherRepo,
				ref:        "",
			},
			wantOwner: "user123",
			wantName:  "repo456",
			wantNil:   false,
		},
		{
			name: "正常系: nilのrepositoryの場合はnilを返す",
			args: args{
				sub:        "user123",
				email:      "test@example.com",
				name:       "testuser",
				provider:   domain.ProviderTypeGitHub,
				repository: nil,
				ref:        "",
			},
			wantOwner: "",
			wantName:  "",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userInfo, err := domain.NewUserInfo(
				tt.args.sub,
				tt.args.email,
				tt.args.name,
				tt.args.provider,
				tt.args.repository,
				tt.args.ref,
			)
			if err != nil {
				t.Fatalf("NewUserInfo failed: %v", err)
			}

			got := userInfo.Repository()

			if tt.wantNil {
				if got != nil {
					t.Errorf("UserInfo.Repository() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("UserInfo.Repository() = nil, want non-nil")
				return
			}

			if diff := cmp.Diff(tt.wantOwner, got.Owner()); diff != "" {
				t.Errorf("RepositoryIdentifier.Owner() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantName, got.Name()); diff != "" {
				t.Errorf("RepositoryIdentifier.Name() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
