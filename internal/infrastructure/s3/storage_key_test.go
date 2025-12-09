package s3_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/na2na-p/cargohold/internal/infrastructure/s3"
)

func TestGenerateStorageKey(t *testing.T) {
	type args struct {
		oid      string
		hashAlgo string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			name: "正常系: 標準的なSHA256ハッシュ",
			args: args{
				oid:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				hashAlgo: "sha256",
			},
			want:    "objects/sha256/ab/cd/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			wantErr: nil,
		},
		{
			name: "正常系: SHA512ハッシュ",
			args: args{
				oid:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				hashAlgo: "sha512",
			},
			want:    "objects/sha512/ab/cd/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			wantErr: nil,
		},
		{
			name: "正常系: ハイフン付きハッシュアルゴリズム",
			args: args{
				oid:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				hashAlgo: "sha-256",
			},
			want:    "objects/sha-256/ab/cd/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			wantErr: nil,
		},
		{
			name: "異常系: 空のOID",
			args: args{
				oid:      "",
				hashAlgo: "sha256",
			},
			want:    "",
			wantErr: s3.ErrInvalidOID,
		},
		{
			name: "異常系: 非hex文字を含むOID",
			args: args{
				oid:      "ghijkl1234567890ghijkl1234567890ghijkl1234567890ghijkl1234567890",
				hashAlgo: "sha256",
			},
			want:    "",
			wantErr: s3.ErrInvalidOID,
		},
		{
			name: "異常系: 空のhashAlgo",
			args: args{
				oid:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				hashAlgo: "",
			},
			want:    "",
			wantErr: s3.ErrInvalidHashAlgo,
		},
		{
			name: "異常系: スラッシュを含むhashAlgo（パストラバーサル）",
			args: args{
				oid:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				hashAlgo: "../etc",
			},
			want:    "",
			wantErr: s3.ErrInvalidHashAlgo,
		},
		{
			name: "異常系: ドットドットを含むhashAlgo",
			args: args{
				oid:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				hashAlgo: "sha256..",
			},
			want:    "",
			wantErr: s3.ErrInvalidHashAlgo,
		},
		{
			name: "異常系: 不正な文字を含むhashAlgo",
			args: args{
				oid:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				hashAlgo: "sha256!@#",
			},
			want:    "",
			wantErr: s3.ErrInvalidHashAlgo,
		},
		{
			name: "異常系: OIDが短すぎる（4文字未満）",
			args: args{
				oid:      "abc",
				hashAlgo: "sha256",
			},
			want:    "",
			wantErr: s3.ErrInvalidOID,
		},
		{
			name: "正常系: ちょうど4文字のOID",
			args: args{
				oid:      "abcd",
				hashAlgo: "sha256",
			},
			want:    "objects/sha256/ab/cd/abcd",
			wantErr: nil,
		},
		{
			name: "正常系: 大文字hex",
			args: args{
				oid:      "ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890",
				hashAlgo: "SHA256",
			},
			want:    "objects/SHA256/AB/CD/ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s3.GenerateStorageKey(tt.args.oid, tt.args.hashAlgo)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if !cmp.Equal(err, tt.wantErr, cmp.Comparer(func(a, b error) bool {
					return a.Error() == b.Error()
				})) {
					t.Errorf("error mismatch: got %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("want no error, but got %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
