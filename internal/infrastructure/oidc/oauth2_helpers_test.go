package oidc_test

import (
	"encoding/base64"
	"testing"

	"github.com/na2na-p/cargohold/internal/infrastructure/oidc"
)

func TestGenerateState(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "正常系: エラーなくstateが生成される",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := oidc.GenerateState()

			// エラーがないこと
			if err != nil {
				t.Fatalf("GenerateState() でエラーが発生しました: %v", err)
			}

			// 返り値が空文字列でないこと
			if got == "" {
				t.Fatal("GenerateState() が空文字列を返しました")
			}

			// 返り値がbase64 URL encodingでデコード可能であること
			decoded, decodeErr := base64.URLEncoding.DecodeString(got)
			if decodeErr != nil {
				t.Fatalf("GenerateState() の返り値がbase64 URL encodingでデコードできません: %v", decodeErr)
			}

			// デコード後のバイト長が32バイトであること（実装の仕様）
			if len(decoded) != 32 {
				t.Errorf("デコード後のバイト長が期待値と異なります: got %d, want 32", len(decoded))
			}
		})
	}
}

func TestGenerateState_Uniqueness(t *testing.T) {
	tests := []struct {
		name       string
		iterations int
	}{
		{
			name:       "正常系: 複数回呼び出しで異なる値が返る",
			iterations: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen := make(map[string]struct{})

			for i := 0; i < tt.iterations; i++ {
				got, err := oidc.GenerateState()
				if err != nil {
					t.Fatalf("GenerateState() でエラーが発生しました（%d回目）: %v", i+1, err)
				}

				if _, exists := seen[got]; exists {
					t.Fatalf("GenerateState() が重複した値を返しました（%d回目）: %s", i+1, got)
				}
				seen[got] = struct{}{}
			}
		})
	}
}
