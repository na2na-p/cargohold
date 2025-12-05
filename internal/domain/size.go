package domain

import "errors"

type Size struct {
	value int64
}

var ErrInvalidSize = errors.New("size must be non-negative")

func NewSize(value int64) (Size, error) {
	if value < 0 {
		return Size{}, ErrInvalidSize
	}

	return Size{value: value}, nil
}

func (s Size) Int64() int64 {
	return s.value
}
