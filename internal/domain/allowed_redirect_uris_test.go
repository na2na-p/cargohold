package domain_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewAllowedRedirectURIs(t *testing.T) {
	tests := []struct {
		name    string
		uris    []string
		wantErr error
	}{
		{
			name:    "正常系: 単一の有効なURIで作成できる",
			uris:    []string{"https://example.com/callback"},
			wantErr: nil,
		},
		{
			name:    "正常系: 複数の有効なURIで作成できる",
			uris:    []string{"https://example.com/callback", "https://localhost:3000/oauth"},
			wantErr: nil,
		},
		{
			name:    "異常系: 空のスライスの場合、エラーが返る",
			uris:    []string{},
			wantErr: domain.ErrEmptyAllowedRedirectURIs,
		},
		{
			name:    "異常系: nilの場合、エラーが返る",
			uris:    nil,
			wantErr: domain.ErrEmptyAllowedRedirectURIs,
		},
		{
			name:    "異常系: 空文字列を含む場合、エラーが返る",
			uris:    []string{"https://example.com/callback", ""},
			wantErr: domain.ErrInvalidRedirectURIFormat,
		},
		{
			name:    "異常系: スキームがないURIの場合、エラーが返る",
			uris:    []string{"example.com/callback"},
			wantErr: domain.ErrInvalidRedirectURIFormat,
		},
		{
			name:    "異常系: ホストがないURIの場合、エラーが返る",
			uris:    []string{"https:///callback"},
			wantErr: domain.ErrInvalidRedirectURIFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewAllowedRedirectURIs(tt.uris)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("want error %v, but got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("want no error, but got %v", err)
			}

			if got == nil {
				t.Fatal("want non-nil AllowedRedirectURIs, but got nil")
			}
		})
	}
}

func TestAllowedRedirectURIs_Contains(t *testing.T) {
	tests := []struct {
		name      string
		uris      []string
		targetURI string
		want      bool
	}{
		{
			name:      "正常系: 許可リストに含まれるURIの場合、trueを返す",
			uris:      []string{"https://example.com/callback", "https://localhost:3000/oauth"},
			targetURI: "https://example.com/callback",
			want:      true,
		},
		{
			name:      "正常系: 許可リストに含まれない URIの場合、falseを返す",
			uris:      []string{"https://example.com/callback"},
			targetURI: "https://other.com/callback",
			want:      false,
		},
		{
			name:      "正常系: 空文字列の場合、falseを返す",
			uris:      []string{"https://example.com/callback"},
			targetURI: "",
			want:      false,
		},
		{
			name:      "正常系: 部分一致しないURIの場合、falseを返す",
			uris:      []string{"https://example.com/callback"},
			targetURI: "https://example.com/callback/extra",
			want:      false,
		},
		{
			name:      "正常系: 完全一致するURIの場合、trueを返す",
			uris:      []string{"https://example.com/callback?param=value"},
			targetURI: "https://example.com/callback?param=value",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowedURIs, err := domain.NewAllowedRedirectURIs(tt.uris)
			if err != nil {
				t.Fatalf("failed to create AllowedRedirectURIs: %v", err)
			}

			got := allowedURIs.Contains(tt.targetURI)

			if got != tt.want {
				t.Errorf("Contains(%q) = %v, want %v", tt.targetURI, got, tt.want)
			}
		})
	}
}

func TestAllowedRedirectURIs_Values(t *testing.T) {
	tests := []struct {
		name string
		uris []string
		want []string
	}{
		{
			name: "正常系: 格納されたURIのコピーを返す",
			uris: []string{"https://example.com/callback", "https://localhost:3000/oauth"},
			want: []string{"https://example.com/callback", "https://localhost:3000/oauth"},
		},
		{
			name: "正常系: 単一のURIの場合も正しく返す",
			uris: []string{"https://example.com/callback"},
			want: []string{"https://example.com/callback"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowedURIs, err := domain.NewAllowedRedirectURIs(tt.uris)
			if err != nil {
				t.Fatalf("failed to create AllowedRedirectURIs: %v", err)
			}

			got := allowedURIs.Values()

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Values() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
