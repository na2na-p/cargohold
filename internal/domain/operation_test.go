package domain_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestParseOperation(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    domain.Operation
		wantErr error
	}{
		{
			name:    "正常系: downloadが指定された場合、OperationDownloadが返る",
			args:    args{s: "download"},
			want:    domain.OperationDownload,
			wantErr: nil,
		},
		{
			name:    "正常系: uploadが指定された場合、OperationUploadが返る",
			args:    args{s: "upload"},
			want:    domain.OperationUpload,
			wantErr: nil,
		},
		{
			name:    "異常系: 不正な値が指定された場合、ErrInvalidOperationが返る",
			args:    args{s: "invalid"},
			want:    domain.Operation{},
			wantErr: domain.ErrInvalidOperation,
		},
		{
			name:    "異常系: 空文字が指定された場合、ErrInvalidOperationが返る",
			args:    args{s: ""},
			want:    domain.Operation{},
			wantErr: domain.ErrInvalidOperation,
		},
		{
			name:    "異常系: 大文字のDOWNLOADが指定された場合、ErrInvalidOperationが返る",
			args:    args{s: "DOWNLOAD"},
			want:    domain.Operation{},
			wantErr: domain.ErrInvalidOperation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ParseOperation(tt.args.s)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseOperation() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}

			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(domain.Operation{})); diff != "" {
				t.Errorf("ParseOperation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestOperation_String(t *testing.T) {
	tests := []struct {
		name string
		op   domain.Operation
		want string
	}{
		{
			name: "正常系: OperationDownloadの場合、downloadが返る",
			op:   domain.OperationDownload,
			want: "download",
		},
		{
			name: "正常系: OperationUploadの場合、uploadが返る",
			op:   domain.OperationUpload,
			want: "upload",
		},
		{
			name: "正常系: ゼロ値の場合、空文字が返る",
			op:   domain.Operation{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.op.String()
			if got != tt.want {
				t.Errorf("Operation.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOperation_IsZero(t *testing.T) {
	tests := []struct {
		name string
		op   domain.Operation
		want bool
	}{
		{
			name: "正常系: OperationDownloadの場合、falseが返る",
			op:   domain.OperationDownload,
			want: false,
		},
		{
			name: "正常系: OperationUploadの場合、falseが返る",
			op:   domain.OperationUpload,
			want: false,
		},
		{
			name: "正常系: ゼロ値の場合、trueが返る",
			op:   domain.Operation{},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.op.IsZero()
			if got != tt.want {
				t.Errorf("Operation.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}
