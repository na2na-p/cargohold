package domain_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewProviderType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    domain.ProviderType
		wantErr error
	}{
		{
			name:    "正常系: githubを指定した場合、ProviderTypeが返却される",
			input:   "github",
			want:    domain.ProviderTypeGitHub,
			wantErr: nil,
		},
		{
			name:    "異常系: 空文字を指定した場合、ErrInvalidProviderTypeが返却される",
			input:   "",
			want:    domain.ProviderType{},
			wantErr: domain.ErrInvalidProviderType,
		},
		{
			name:    "異常系: 不明なプロバイダ種別を指定した場合、ErrInvalidProviderTypeが返却される",
			input:   "unknown",
			want:    domain.ProviderType{},
			wantErr: domain.ErrInvalidProviderType,
		},
		{
			name:    "異常系: 大文字のGITHUBを指定した場合、ErrInvalidProviderTypeが返却される",
			input:   "GITHUB",
			want:    domain.ProviderType{},
			wantErr: domain.ErrInvalidProviderType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewProviderType(tt.input)

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

			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(domain.ProviderType{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProviderType_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		want       domain.ProviderType
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:    "正常系: githubを指定した場合、ProviderTypeが返却される",
			input:   []byte(`"github"`),
			want:    domain.ProviderTypeGitHub,
			wantErr: nil,
		},
		{
			name:       "異常系: 無効なJSON形式の場合、エラーが返却される",
			input:      []byte(`invalid`),
			want:       domain.ProviderType{},
			wantAnyErr: true,
		},
		{
			name:    "異常系: 空文字を指定した場合、ErrInvalidProviderTypeが返却される",
			input:   []byte(`""`),
			want:    domain.ProviderType{},
			wantErr: domain.ErrInvalidProviderType,
		},
		{
			name:    "異常系: 不明なプロバイダ種別を指定した場合、ErrInvalidProviderTypeが返却される",
			input:   []byte(`"unknown"`),
			want:    domain.ProviderType{},
			wantErr: domain.ErrInvalidProviderType,
		},
		{
			name:    "異常系: 大文字のGITHUBを指定した場合、ErrInvalidProviderTypeが返却される",
			input:   []byte(`"GITHUB"`),
			want:    domain.ProviderType{},
			wantErr: domain.ErrInvalidProviderType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got domain.ProviderType
			err := got.UnmarshalJSON(tt.input)

			if tt.wantAnyErr {
				if err == nil {
					t.Fatalf("want any error, but got nil")
				}
				return
			}

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

			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(domain.ProviderType{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProviderType_String(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "正常系: githubの場合、github文字列が返却される",
			input: "github",
			want:  "github",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt, err := domain.NewProviderType(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := pt.String()
			if got != tt.want {
				t.Errorf("want %q, but got %q", tt.want, got)
			}
		})
	}
}
