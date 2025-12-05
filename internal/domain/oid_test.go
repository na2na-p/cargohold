package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewOID(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name       string
		args       args
		want       string
		wantErr    bool
		wantErrVal error
	}{
		{
			name: "正常なOID（小文字）",
			args: args{
				value: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			},
			want:       "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "正常なOID（大文字）",
			args: args{
				value: "A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2",
			},
			want:       "A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "正常なOID（混在）",
			args: args{
				value: "a1B2c3D4e5F6a1B2c3D4e5F6a1B2c3D4e5F6a1B2c3D4e5F6a1B2c3D4e5F6a1B2",
			},
			want:       "a1B2c3D4e5F6a1B2c3D4e5F6a1B2c3D4e5F6a1B2c3D4e5F6a1B2c3D4e5F6a1B2",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "空文字列",
			args: args{
				value: "",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidOIDFormat,
		},
		{
			name: "63文字（短すぎる）",
			args: args{
				value: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidOIDFormat,
		},
		{
			name: "65文字（長すぎる）",
			args: args{
				value: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidOIDFormat,
		},
		{
			name: "16進数以外の文字を含む",
			args: args{
				value: "g1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidOIDFormat,
		},
		{
			name: "特殊文字を含む",
			args: args{
				value: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1-2",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidOIDFormat,
		},
		{
			name: "スペースを含む",
			args: args{
				value: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1 2",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidOIDFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, err := domain.NewOID(tt.args.value)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewOID() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErrVal) {
					t.Errorf("NewOID() error = %v, want %v", err, tt.wantErrVal)
				}
			} else {
				if err != nil {
					t.Errorf("NewOID() error = %v, wantErr %v", err, tt.wantErr)
				}
				if diff := cmp.Diff(tt.want, oid.String()); diff != "" {
					t.Errorf("OID.String() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestOID_String(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "正常なOID",
			args: args{
				value: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			},
			want: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, err := domain.NewOID(tt.args.value)
			if err != nil {
				t.Fatalf("NewOID() failed: %v", err)
			}

			got := oid.String()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("OID.String() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestOID_Immutability(t *testing.T) {
	// ValueObjectの不変性を確認
	// Goの構造体は値渡しなので、コピーしても元の値は変わらない
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "不変性の確認",
			args: args{
				value: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid1, err := domain.NewOID(tt.args.value)
			if err != nil {
				t.Fatalf("NewOID() failed: %v", err)
			}

			oid2 := oid1

			if diff := cmp.Diff(oid1.String(), oid2.String()); diff != "" {
				t.Errorf("コピー後の値が異なる mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.value, oid1.String()); diff != "" {
				t.Errorf("元の値が変更された mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func BenchmarkNewOID(b *testing.B) {
	validOID := strings.Repeat("a1b2c3d4", 8)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = domain.NewOID(validOID)
	}
}
