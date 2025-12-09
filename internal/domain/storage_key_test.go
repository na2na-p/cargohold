package domain_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewStorageKey(t *testing.T) {
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
			name: "正常系: 有効なストレージキー",
			args: args{
				value: "objects/sha256/12/34/1234567890abcdef",
			},
			want:       "objects/sha256/12/34/1234567890abcdef",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "正常系: スラッシュを含むパス",
			args: args{
				value: "test/storage/key",
			},
			want:       "test/storage/key",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "正常系: 単一の文字",
			args: args{
				value: "a",
			},
			want:       "a",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "正常系: 長いパス",
			args: args{
				value: "objects/sha256/ab/cd/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
			want:       "objects/sha256/ab/cd/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "異常系: 空文字列",
			args: args{
				value: "",
			},
			want:       "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidStorageKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageKey, err := domain.NewStorageKey(tt.args.value)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewStorageKey() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErrVal) {
					t.Errorf("NewStorageKey() error = %v, want %v", err, tt.wantErrVal)
				}
			} else {
				if err != nil {
					t.Errorf("NewStorageKey() error = %v, wantErr %v", err, tt.wantErr)
				}
				if diff := cmp.Diff(tt.want, storageKey.String()); diff != "" {
					t.Errorf("StorageKey.String() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestStorageKey_String(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "正常系: String()メソッドが値を返す",
			args: args{
				value: "objects/sha256/12/34/1234567890abcdef",
			},
			want: "objects/sha256/12/34/1234567890abcdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageKey, err := domain.NewStorageKey(tt.args.value)
			if err != nil {
				t.Fatalf("NewStorageKey() failed: %v", err)
			}

			got := storageKey.String()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("StorageKey.String() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStorageKey_Immutability(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常系: Value Objectの不変性確認",
			args: args{
				value: "objects/sha256/12/34/1234567890abcdef",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageKey1, err := domain.NewStorageKey(tt.args.value)
			if err != nil {
				t.Fatalf("NewStorageKey() failed: %v", err)
			}

			storageKey2 := storageKey1

			if diff := cmp.Diff(storageKey1.String(), storageKey2.String()); diff != "" {
				t.Errorf("コピー後の値が異なる mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.args.value, storageKey1.String()); diff != "" {
				t.Errorf("元の値が変更された mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func BenchmarkNewStorageKey(b *testing.B) {
	validKey := "objects/sha256/12/34/1234567890abcdef"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = domain.NewStorageKey(validKey)
	}
}
