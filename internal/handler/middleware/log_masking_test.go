package middleware_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/handler/middleware"
)

func TestMaskSensitiveParams(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "正常系: session_id がマスキングされる",
			args: args{
				uri: "/auth/session?session_id=e3ea92f8-a131-4056-aed8-46cd8175cab8",
			},
			want: "/auth/session?session_id=***",
		},
		{
			name: "正常系: code がマスキングされる",
			args: args{
				uri: "/auth/github/callback?code=abc123def456",
			},
			want: "/auth/github/callback?code=***",
		},
		{
			name: "正常系: state がマスキングされる",
			args: args{
				uri: "/auth/github/callback?state=xyz789",
			},
			want: "/auth/github/callback?state=***",
		},
		{
			name: "正常系: 複数の機微パラメータが同時にマスキングされる",
			args: args{
				uri: "/auth/github/callback?code=abc123&state=xyz789&redirect_uri=https://example.com",
			},
			want: "/auth/github/callback?code=***&redirect_uri=https%3A%2F%2Fexample.com&state=***",
		},
		{
			name: "正常系: クエリパラメータがない場合はそのまま返される",
			args: args{
				uri: "/auth/session",
			},
			want: "/auth/session",
		},
		{
			name: "正常系: 機微でないパラメータはマスキングされない",
			args: args{
				uri: "/api/users?page=1&limit=10",
			},
			want: "/api/users?limit=10&page=1",
		},
		{
			name: "正常系: 空文字列が渡された場合は空文字列を返す",
			args: args{
				uri: "",
			},
			want: "",
		},
		{
			name: "正常系: session_id と他のパラメータが混在する場合",
			args: args{
				uri: "/auth/session?session_id=secret123&host=example.com&other=value",
			},
			want: "/auth/session?host=example.com&other=value&session_id=***",
		},
		{
			name: "正常系: エンコードが必要な値もマスキング後にエンコードされる",
			args: args{
				uri: "/auth/callback?code=abc%2F123&normal=test",
			},
			want: "/auth/callback?code=***&normal=test",
		},
		{
			name: "正常系: パスのみでクエリが空の場合",
			args: args{
				uri: "/healthz",
			},
			want: "/healthz",
		},
		{
			name: "正常系: 不正なURLの場合はそのまま返される",
			args: args{
				uri: "://invalid-url",
			},
			want: "://invalid-url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := middleware.MaskSensitiveParams(tt.args.uri)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MaskSensitiveParams() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
