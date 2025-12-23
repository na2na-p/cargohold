package s3

import (
	"errors"
	"fmt"
)

type StorageOperation string

const (
	OperationPut  StorageOperation = "put"
	OperationGet  StorageOperation = "get"
	OperationHead StorageOperation = "head"
)

type StorageError struct {
	Operation StorageOperation
	Err       error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage %s error: %v", e.Operation, e.Err)
}

func (e *StorageError) Unwrap() error {
	return e.Err
}

func (e *StorageError) Is(target error) bool {
	var t *StorageError
	if errors.As(target, &t) {
		return e.Operation == t.Operation
	}
	return false
}

func NewStorageError(operation StorageOperation, err error) *StorageError {
	return &StorageError{
		Operation: operation,
		Err:       err,
	}
}

func IsStorageError(err error) bool {
	if err == nil {
		return false
	}
	var storageErr *StorageError
	return errors.As(err, &storageErr)
}
