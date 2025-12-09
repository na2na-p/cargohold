package redis_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
)

func TestMetadataKey(t *testing.T) {
	tests := []struct {
		name string
		oid  string
		want string
	}{
		{
			name: "正常系: OIDからメタデータキーが生成される",
			oid:  "abc123",
			want: "lfs:meta:abc123",
		},
		{
			name: "正常系: 空文字のOIDでもキーが生成される",
			oid:  "",
			want: "lfs:meta:",
		},
		{
			name: "正常系: SHA256形式のOIDでキーが生成される",
			oid:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			want: "lfs:meta:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redis.MetadataKey(tt.oid)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MetadataKey() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSessionKey(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		want      string
	}{
		{
			name:      "正常系: セッションIDからセッションキーが生成される",
			sessionID: "session-12345",
			want:      "lfs:session:session-12345",
		},
		{
			name:      "正常系: 空文字のセッションIDでもキーが生成される",
			sessionID: "",
			want:      "lfs:session:",
		},
		{
			name:      "正常系: UUID形式のセッションIDでキーが生成される",
			sessionID: "550e8400-e29b-41d4-a716-446655440000",
			want:      "lfs:session:550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redis.SessionKey(tt.sessionID)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("SessionKey() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBatchUploadKey(t *testing.T) {
	tests := []struct {
		name string
		oid  string
		want string
	}{
		{
			name: "正常系: OIDからバッチアップロードキーが生成される",
			oid:  "upload-abc123",
			want: "lfs:batch:upload:upload-abc123",
		},
		{
			name: "正常系: 空文字のOIDでもキーが生成される",
			oid:  "",
			want: "lfs:batch:upload:",
		},
		{
			name: "正常系: SHA256形式のOIDでキーが生成される",
			oid:  "a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a",
			want: "lfs:batch:upload:a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redis.BatchUploadKey(tt.oid)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("BatchUploadKey() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestOIDCGitHubRepoKey(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		want       string
	}{
		{
			name:       "正常系: リポジトリ名からOIDC GitHubリポジトリキーが生成される",
			repository: "owner/repo",
			want:       "lfs:oidc:github:repo:owner/repo",
		},
		{
			name:       "正常系: 空文字のリポジトリ名でもキーが生成される",
			repository: "",
			want:       "lfs:oidc:github:repo:",
		},
		{
			name:       "正常系: 複雑なリポジトリ名でキーが生成される",
			repository: "my-org/my-complex-repo-name",
			want:       "lfs:oidc:github:repo:my-org/my-complex-repo-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redis.OIDCGitHubRepoKey(tt.repository)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("OIDCGitHubRepoKey() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestOIDCStateKey(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  string
	}{
		{
			name:  "正常系: ステートパラメータからOIDCステートキーが生成される",
			state: "random-state-value",
			want:  "lfs:oidc:state:random-state-value",
		},
		{
			name:  "正常系: 空文字のステートでもキーが生成される",
			state: "",
			want:  "lfs:oidc:state:",
		},
		{
			name:  "正常系: Base64形式のステートでキーが生成される",
			state: "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=",
			want:  "lfs:oidc:state:YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redis.OIDCStateKey(tt.state)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("OIDCStateKey() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestOIDCJWKSKey(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{
			name:     "正常系: プロバイダー名からOIDC JWKSキーが生成される",
			provider: "github",
			want:     "lfs:oidc:jwks:github",
		},
		{
			name:     "正常系: 空文字のプロバイダーでもキーが生成される",
			provider: "",
			want:     "lfs:oidc:jwks:",
		},
		{
			name:     "正常系: URL形式のプロバイダーでキーが生成される",
			provider: "https://token.actions.githubusercontent.com",
			want:     "lfs:oidc:jwks:https://token.actions.githubusercontent.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redis.OIDCJWKSKey(tt.provider)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("OIDCJWKSKey() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
