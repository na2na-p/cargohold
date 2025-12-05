package usecase_test

import (
	"errors"
	"testing"

	"github.com/na2na-p/cargohold/internal/usecase"
)

func TestVerifyRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     usecase.VerifyRequest
		wantErr error
	}{
		{
			name: "正常系: 有効なOIDとSizeの場合、エラーなし",
			req: usecase.VerifyRequest{
				OID:  "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
				Size: 1024,
			},
			wantErr: nil,
		},
		{
			name: "異常系: OIDが空文字の場合、ErrInvalidOID",
			req: usecase.VerifyRequest{
				OID:  "",
				Size: 1024,
			},
			wantErr: usecase.ErrInvalidOID,
		},
		{
			name: "異常系: OID形式が不正な場合、ErrInvalidOID",
			req: usecase.VerifyRequest{
				OID:  "invalid-oid",
				Size: 1024,
			},
			wantErr: usecase.ErrInvalidOID,
		},
		{
			name: "異常系: OIDが短すぎる場合、ErrInvalidOID",
			req: usecase.VerifyRequest{
				OID:  "a1b2c3d4",
				Size: 1024,
			},
			wantErr: usecase.ErrInvalidOID,
		},
		{
			name: "異常系: Sizeが0の場合、ErrInvalidSize",
			req: usecase.VerifyRequest{
				OID:  "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
				Size: 0,
			},
			wantErr: usecase.ErrInvalidSize,
		},
		{
			name: "異常系: Sizeが負の値の場合、ErrInvalidSize",
			req: usecase.VerifyRequest{
				OID:  "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
				Size: -1,
			},
			wantErr: usecase.ErrInvalidSize,
		},
		{
			name: "正常系: Sizeが1の場合、エラーなし",
			req: usecase.VerifyRequest{
				OID:  "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
				Size: 1,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("VerifyRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("VerifyRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}
