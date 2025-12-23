package s3_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/infrastructure/s3"
)

func TestStorageError_Error(t *testing.T) {
	type fields struct {
		Operation s3.StorageOperation
		Err       error
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "正常系: Put操作のエラーメッセージが正しく生成される",
			fields: fields{
				Operation: s3.OperationPut,
				Err:       errors.New("connection refused"),
			},
			want: "storage put error: connection refused",
		},
		{
			name: "正常系: Get操作のエラーメッセージが正しく生成される",
			fields: fields{
				Operation: s3.OperationGet,
				Err:       errors.New("timeout"),
			},
			want: "storage get error: timeout",
		},
		{
			name: "正常系: Head操作のエラーメッセージが正しく生成される",
			fields: fields{
				Operation: s3.OperationHead,
				Err:       errors.New("access denied"),
			},
			want: "storage head error: access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &s3.StorageError{
				Operation: tt.fields.Operation,
				Err:       tt.fields.Err,
			}
			got := e.Error()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStorageError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	storageErr := &s3.StorageError{
		Operation: s3.OperationPut,
		Err:       originalErr,
	}

	got := storageErr.Unwrap()
	if got != originalErr {
		t.Errorf("Unwrap() = %v, want %v", got, originalErr)
	}
}

func TestStorageError_Is(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name: "正常系: 同じ操作タイプのStorageErrorはtrueを返す",
			err: &s3.StorageError{
				Operation: s3.OperationPut,
				Err:       errors.New("some error"),
			},
			target: &s3.StorageError{Operation: s3.OperationPut},
			want:   true,
		},
		{
			name: "正常系: 異なる操作タイプのStorageErrorはfalseを返す",
			err: &s3.StorageError{
				Operation: s3.OperationPut,
				Err:       errors.New("some error"),
			},
			target: &s3.StorageError{Operation: s3.OperationGet},
			want:   false,
		},
		{
			name: "正常系: wrapされたStorageErrorもerrors.Isで検出可能",
			err: errors.Join(
				errors.New("wrapper"),
				&s3.StorageError{
					Operation: s3.OperationGet,
					Err:       errors.New("nested"),
				},
			),
			target: &s3.StorageError{Operation: s3.OperationGet},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errors.Is(tt.err, tt.target)
			if got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsStorageError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "正常系: StorageErrorはtrueを返す",
			err: &s3.StorageError{
				Operation: s3.OperationPut,
				Err:       errors.New("some error"),
			},
			want: true,
		},
		{
			name: "正常系: wrapされたStorageErrorもtrueを返す",
			err: errors.Join(
				errors.New("wrapper"),
				&s3.StorageError{
					Operation: s3.OperationGet,
					Err:       errors.New("nested"),
				},
			),
			want: true,
		},
		{
			name: "正常系: 通常のエラーはfalseを返す",
			err:  errors.New("normal error"),
			want: false,
		},
		{
			name: "正常系: nilはfalseを返す",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s3.IsStorageError(tt.err)
			if got != tt.want {
				t.Errorf("IsStorageError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewStorageError(t *testing.T) {
	type args struct {
		operation s3.StorageOperation
		err       error
	}
	tests := []struct {
		name string
		args args
		want *s3.StorageError
	}{
		{
			name: "正常系: Put操作のStorageErrorが生成される",
			args: args{
				operation: s3.OperationPut,
				err:       errors.New("put error"),
			},
			want: &s3.StorageError{
				Operation: s3.OperationPut,
				Err:       errors.New("put error"),
			},
		},
		{
			name: "正常系: Get操作のStorageErrorが生成される",
			args: args{
				operation: s3.OperationGet,
				err:       errors.New("get error"),
			},
			want: &s3.StorageError{
				Operation: s3.OperationGet,
				Err:       errors.New("get error"),
			},
		},
		{
			name: "正常系: Head操作のStorageErrorが生成される",
			args: args{
				operation: s3.OperationHead,
				err:       errors.New("head error"),
			},
			want: &s3.StorageError{
				Operation: s3.OperationHead,
				Err:       errors.New("head error"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s3.NewStorageError(tt.args.operation, tt.args.err)
			if got.Operation != tt.want.Operation {
				t.Errorf("NewStorageError().Operation = %v, want %v", got.Operation, tt.want.Operation)
			}
			if got.Err.Error() != tt.want.Err.Error() {
				t.Errorf("NewStorageError().Err = %v, want %v", got.Err, tt.want.Err)
			}
		})
	}
}
