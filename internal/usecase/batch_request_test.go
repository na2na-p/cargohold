package usecase_test

import (
	"errors"
	"testing"

	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
)

func TestBatchRequest_Validate(t *testing.T) {
	validOID := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"

	tests := []struct {
		name    string
		request usecase.BatchRequest
		wantErr error
	}{
		{
			name: "正常系: hash_algoが空文字列の場合はエラーなし",
			request: usecase.NewBatchRequest(
				domain.OperationUpload,
				[]usecase.RequestObject{usecase.NewRequestObject(validOID, 100)},
				nil,
				nil,
				"",
				nil,
			),
			wantErr: nil,
		},
		{
			name: "正常系: hash_algoがsha256の場合はエラーなし",
			request: usecase.NewBatchRequest(
				domain.OperationDownload,
				[]usecase.RequestObject{usecase.NewRequestObject(validOID, 100)},
				nil,
				nil,
				"sha256",
				nil,
			),
			wantErr: nil,
		},
		{
			name: "異常系: hash_algoがmd5の場合はエラー",
			request: usecase.NewBatchRequest(
				domain.OperationUpload,
				[]usecase.RequestObject{usecase.NewRequestObject(validOID, 100)},
				nil,
				nil,
				"md5",
				nil,
			),
			wantErr: usecase.ErrInvalidHashAlgorithm,
		},
		{
			name: "異常系: hash_algoがパストラバーサル攻撃文字列の場合はエラー",
			request: usecase.NewBatchRequest(
				domain.OperationUpload,
				[]usecase.RequestObject{usecase.NewRequestObject(validOID, 100)},
				nil,
				nil,
				"../mal",
				nil,
			),
			wantErr: usecase.ErrInvalidHashAlgorithm,
		},
		{
			name: "異常系: hash_algoがSHA256（大文字）の場合はエラー",
			request: usecase.NewBatchRequest(
				domain.OperationUpload,
				[]usecase.RequestObject{usecase.NewRequestObject(validOID, 100)},
				nil,
				nil,
				"SHA256",
				nil,
			),
			wantErr: usecase.ErrInvalidHashAlgorithm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error mismatch: want %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}
		})
	}
}
