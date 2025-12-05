package domain_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewHashAlgorithm(t *testing.T) {
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
			name: "正常系: sha256",
			args: args{
				value: "sha256",
			},
			want:       "sha256",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "正常系: 空文字列はデフォルト値（sha256）を使用",
			args: args{
				value: "",
			},
			want:       "sha256",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "異常系: md5は許可されない",
			args: args{
				value: "md5",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidHashAlgorithm,
		},
		{
			name: "異常系: SHA256（大文字）は許可されない",
			args: args{
				value: "SHA256",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidHashAlgorithm,
		},
		{
			name: "異常系: sha512は許可されない",
			args: args{
				value: "sha512",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidHashAlgorithm,
		},
		{
			name: "異常系: パストラバーサル攻撃文字列",
			args: args{
				value: "../mal",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidHashAlgorithm,
		},
		{
			name: "異常系: 任意の文字列",
			args: args{
				value: "invalid-algorithm",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidHashAlgorithm,
		},
		{
			name: "異常系: スペースを含む文字列",
			args: args{
				value: "sha 256",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidHashAlgorithm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashAlgo, err := domain.NewHashAlgorithm(tt.args.value)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewHashAlgorithm() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErrVal) {
					t.Errorf("NewHashAlgorithm() error = %v, want %v", err, tt.wantErrVal)
				}
			} else {
				if err != nil {
					t.Errorf("NewHashAlgorithm() error = %v, wantErr %v", err, tt.wantErr)
				}
				if diff := cmp.Diff(tt.want, hashAlgo.String()); diff != "" {
					t.Errorf("HashAlgorithm.String() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestHashAlgorithm_String(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "正常系: sha256",
			args: args{
				value: "sha256",
			},
			want: "sha256",
		},
		{
			name: "正常系: 空文字列からのデフォルト値",
			args: args{
				value: "",
			},
			want: "sha256",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashAlgo, err := domain.NewHashAlgorithm(tt.args.value)
			if err != nil {
				t.Fatalf("NewHashAlgorithm() failed: %v", err)
			}

			got := hashAlgo.String()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("HashAlgorithm.String() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHashAlgorithm_Immutability(t *testing.T) {
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
				value: "sha256",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashAlgo1, err := domain.NewHashAlgorithm(tt.args.value)
			if err != nil {
				t.Fatalf("NewHashAlgorithm() failed: %v", err)
			}

			hashAlgo2 := hashAlgo1

			if diff := cmp.Diff(hashAlgo1.String(), hashAlgo2.String()); diff != "" {
				t.Errorf("コピー後の値が異なる mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.value, hashAlgo1.String()); diff != "" {
				t.Errorf("元の値が変更された mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func BenchmarkNewHashAlgorithm(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = domain.NewHashAlgorithm("sha256")
	}
}

func BenchmarkNewHashAlgorithm_Empty(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = domain.NewHashAlgorithm("")
	}
}
