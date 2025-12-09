package domain

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrEmptySub            = errors.New("sub cannot be empty")
	ErrInvalidProviderType = errors.New("invalid provider type")
)
