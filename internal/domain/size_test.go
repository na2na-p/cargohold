package domain_test

import (
	"errors"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewSize(t *testing.T) {
	type args struct {
		value int64
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "正の値",
			args: args{
				value: 1024,
			},
			want:    1024,
			wantErr: false,
		},
		{
			name: "ゼロ",
			args: args{
				value: 0,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "負の値",
			args: args{
				value: -1,
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "大きな正の値",
			args: args{
				value: math.MaxInt64,
			},
			want:    math.MaxInt64,
			wantErr: false,
		},
		{
			name: "最小の負の値",
			args: args{
				value: math.MinInt64,
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "1バイト",
			args: args{
				value: 1,
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "1MB",
			args: args{
				value: 1024 * 1024,
			},
			want:    1024 * 1024,
			wantErr: false,
		},
		{
			name: "1GB",
			args: args{
				value: 1024 * 1024 * 1024,
			},
			want:    1024 * 1024 * 1024,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, err := domain.NewSize(tt.args.value)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewSize() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, domain.ErrInvalidSize) {
					t.Errorf("NewSize() error = %v, want %v", err, domain.ErrInvalidSize)
				}
			} else {
				if err != nil {
					t.Errorf("NewSize() error = %v, wantErr %v", err, tt.wantErr)
				}
				if diff := cmp.Diff(tt.want, size.Int64()); diff != "" {
					t.Errorf("Size.Int64() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestSize_Int64(t *testing.T) {
	type args struct {
		value int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "2048バイト",
			args: args{
				value: 2048,
			},
			want: 2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, err := domain.NewSize(tt.args.value)
			if err != nil {
				t.Fatalf("NewSize() failed: %v", err)
			}

			if diff := cmp.Diff(tt.want, size.Int64()); diff != "" {
				t.Errorf("Size.Int64() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSize_Immutability(t *testing.T) {
	type args struct {
		value int64
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "4096バイトの不変性",
			args: args{
				value: 4096,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ValueObjectの不変性を確認
			// Goの構造体は値渡しなので、コピーしても元の値は変わらない
			size1, err := domain.NewSize(tt.args.value)
			if err != nil {
				t.Fatalf("NewSize() failed: %v", err)
			}

			size2 := size1

			if diff := cmp.Diff(size1.Int64(), size2.Int64()); diff != "" {
				t.Errorf("コピー後の値が異なる (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.value, size1.Int64()); diff != "" {
				t.Errorf("元の値が変更された (-want +got):\n%s", diff)
			}
		})
	}
}

func BenchmarkNewSize(b *testing.B) {
	testValue := int64(1024 * 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = domain.NewSize(testValue)
	}
}
