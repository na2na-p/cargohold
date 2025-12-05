package domain

import "errors"

type StorageKey struct {
	value string
}

var ErrInvalidStorageKey = errors.New("invalid storage key: must not be empty")

func NewStorageKey(value string) (StorageKey, error) {
	if value == "" {
		return StorageKey{}, ErrInvalidStorageKey
	}

	return StorageKey{value: value}, nil
}

func (s StorageKey) String() string {
	return s.value
}
