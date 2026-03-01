package domain_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestParseShellType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    domain.ShellType
		wantErr error
	}{
		{
			name:    "正常系: bashが指定された場合はShellTypeBashを返す",
			input:   "bash",
			want:    domain.ShellTypeBash,
			wantErr: nil,
		},
		{
			name:    "正常系: 空文字の場合はShellTypeBashをデフォルトとして返す",
			input:   "",
			want:    domain.ShellTypeBash,
			wantErr: nil,
		},
		{
			name:    "正常系: zshが指定された場合はShellTypeZshを返す",
			input:   "zsh",
			want:    domain.ShellTypeZsh,
			wantErr: nil,
		},
		{
			name:    "正常系: powershellが指定された場合はShellTypePowerShellを返す",
			input:   "powershell",
			want:    domain.ShellTypePowerShell,
			wantErr: nil,
		},
		{
			name:    "異常系: 不正なシェル名の場合はエラーを返す",
			input:   "fish",
			want:    domain.ShellType{},
			wantErr: domain.ErrInvalidShellType,
		},
		{
			name:    "異常系: 大文字のBASHは不正な値として扱われる",
			input:   "BASH",
			want:    domain.ShellType{},
			wantErr: domain.ErrInvalidShellType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ParseShellType(tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ParseShellType(%q) error = nil, wantErr %v", tt.input, tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseShellType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("ParseShellType(%q) unexpected error: %v", tt.input, err)
				}
				if diff := cmp.Diff(tt.want.String(), got.String()); diff != "" {
					t.Errorf("ParseShellType(%q) mismatch (-want +got):\n%s", tt.input, diff)
				}
			}
		})
	}
}

func TestShellType_String(t *testing.T) {
	tests := []struct {
		name      string
		shellType domain.ShellType
		want      string
	}{
		{name: "bash", shellType: domain.ShellTypeBash, want: "bash"},
		{name: "zsh", shellType: domain.ShellTypeZsh, want: "zsh"},
		{name: "powershell", shellType: domain.ShellTypePowerShell, want: "powershell"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, tt.shellType.String()); diff != "" {
				t.Errorf("String() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
