package logging_test

import (
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/infrastructure/logging"
)

func TestMaskSensitiveAttrs(t *testing.T) {
	type args struct {
		groups []string
		attr   slog.Attr
	}
	tests := []struct {
		name string
		args args
		want slog.Attr
	}{
		{
			name: "正常系: 機密キー(token)が完全一致でマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("token", "secret-value-123"),
			},
			want: slog.String("token", "[REDACTED]"),
		},
		{
			name: "正常系: 機密キー(password)が完全一致でマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("password", "my-password"),
			},
			want: slog.String("password", "[REDACTED]"),
		},
		{
			name: "正常系: 機密キー(access_token)が完全一致でマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("access_token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9"),
			},
			want: slog.String("access_token", "[REDACTED]"),
		},
		{
			name: "正常系: 機密キー(email)が完全一致でマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("email", "user@example.com"),
			},
			want: slog.String("email", "[REDACTED]"),
		},
		{
			name: "正常系: 機密キー(oidc_subject)が完全一致でマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("oidc_subject", "user-id-12345"),
			},
			want: slog.String("oidc_subject", "[REDACTED]"),
		},
		{
			name: "正常系: 機密キー(api_key)が完全一致でマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("api_key", "sk-1234567890"),
			},
			want: slog.String("api_key", "[REDACTED]"),
		},
		{
			name: "正常系: 部分一致(user_token)がマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("user_token", "user-secret-token"),
			},
			want: slog.String("user_token", "[REDACTED]"),
		},
		{
			name: "正常系: 部分一致(auth_password_hash)がマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("auth_password_hash", "hashed-password"),
			},
			want: slog.String("auth_password_hash", "[REDACTED]"),
		},
		{
			name: "正常系: 部分一致(user_email_address)がマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("user_email_address", "user@example.com"),
			},
			want: slog.String("user_email_address", "[REDACTED]"),
		},
		{
			name: "正常系: 大文字(Token)でもマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("Token", "secret-value"),
			},
			want: slog.String("Token", "[REDACTED]"),
		},
		{
			name: "正常系: 全大文字(TOKEN)でもマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("TOKEN", "secret-value"),
			},
			want: slog.String("TOKEN", "[REDACTED]"),
		},
		{
			name: "正常系: 混在ケース(AcCeSs_ToKeN)でもマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("AcCeSs_ToKeN", "secret-value"),
			},
			want: slog.String("AcCeSs_ToKeN", "[REDACTED]"),
		},
		{
			name: "正常系: 非機密キー(user_id)はそのまま出力される",
			args: args{
				groups: nil,
				attr:   slog.String("user_id", "12345"),
			},
			want: slog.String("user_id", "12345"),
		},
		{
			name: "正常系: 非機密キー(status)はそのまま出力される",
			args: args{
				groups: nil,
				attr:   slog.String("status", "active"),
			},
			want: slog.String("status", "active"),
		},
		{
			name: "正常系: 非機密キー(count)はそのまま出力される",
			args: args{
				groups: nil,
				attr:   slog.Int("count", 42),
			},
			want: slog.Int("count", 42),
		},
		{
			name: "正常系: 非機密キー(message)はそのまま出力される",
			args: args{
				groups: nil,
				attr:   slog.String("message", "Operation completed successfully"),
			},
			want: slog.String("message", "Operation completed successfully"),
		},
		{
			name: "正常系: グループが指定されていても機密キーはマスクされる",
			args: args{
				groups: []string{"auth", "user"},
				attr:   slog.String("token", "secret-value"),
			},
			want: slog.String("token", "[REDACTED]"),
		},
		{
			name: "正常系: 機密キー(authorization)がマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("authorization", "Bearer xyz123"),
			},
			want: slog.String("authorization", "[REDACTED]"),
		},
		{
			name: "正常系: 機密キー(session_id)がマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("session_id", "sess-abc123"),
			},
			want: slog.String("session_id", "[REDACTED]"),
		},
		{
			name: "正常系: 機密キー(client_secret)がマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("client_secret", "client-secret-value"),
			},
			want: slog.String("client_secret", "[REDACTED]"),
		},
		{
			name: "正常系: 機密キー(private_key)がマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("private_key", "-----BEGIN RSA PRIVATE KEY-----"),
			},
			want: slog.String("private_key", "[REDACTED]"),
		},
		{
			name: "正常系: 非機密キー(subscription_count)はそのまま出力される",
			args: args{
				groups: nil,
				attr:   slog.Int("subscription_count", 100),
			},
			want: slog.Int("subscription_count", 100),
		},
		{
			name: "正常系: 非機密キー(mailing_list_id)はそのまま出力される",
			args: args{
				groups: nil,
				attr:   slog.String("mailing_list_id", "list-12345"),
			},
			want: slog.String("mailing_list_id", "list-12345"),
		},
		{
			name: "正常系: 非機密キー(subject)はそのまま出力される",
			args: args{
				groups: nil,
				attr:   slog.String("subject", "Meeting Request"),
			},
			want: slog.String("subject", "Meeting Request"),
		},
		{
			name: "正常系: 機密キー(oauth_state)がマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("oauth_state", "random-oauth-state-value"),
			},
			want: slog.String("oauth_state", "[REDACTED]"),
		},
		{
			name: "正常系: 機密キー(oidc_state)がマスクされる",
			args: args{
				groups: nil,
				attr:   slog.String("oidc_state", "random-oidc-state-value"),
			},
			want: slog.String("oidc_state", "[REDACTED]"),
		},
		{
			name: "正常系: 非機密キー(state)はマスクされない",
			args: args{
				groups: nil,
				attr:   slog.String("state", "active"),
			},
			want: slog.String("state", "active"),
		},
		{
			name: "正常系: Group内の機密キー(token)がマスクされ、非機密キー(user_id)は保持される",
			args: args{
				groups: nil,
				attr:   slog.Group("auth", slog.String("token", "secret-token"), slog.String("user_id", "12345")),
			},
			want: slog.Group("auth", slog.String("token", "[REDACTED]"), slog.String("user_id", "12345")),
		},
		{
			name: "正常系: ネストしたGroup内の機密キー(authorization)がマスクされ、非機密キー(content-type)は保持される",
			args: args{
				groups: nil,
				attr:   slog.Group("request", slog.Group("headers", slog.String("authorization", "Bearer xyz"), slog.String("content-type", "application/json"))),
			},
			want: slog.Group("request", slog.Group("headers", slog.String("authorization", "[REDACTED]"), slog.String("content-type", "application/json"))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logging.MaskSensitiveAttrs(tt.args.groups, tt.args.attr)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MaskSensitiveAttrs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
