package domain

import "errors"

var (
	ErrInvalidHashAlgorithm = errors.New("invalid hash algorithm")

	allowedHashAlgorithms = map[string]bool{
		"sha256": true,
	}

	defaultHashAlgorithm = "sha256"
)

type HashAlgorithm struct {
	value string
}

func NewHashAlgorithm(value string) (HashAlgorithm, error) {
	if value == "" {
		return HashAlgorithm{value: defaultHashAlgorithm}, nil
	}

	if !allowedHashAlgorithms[value] {
		return HashAlgorithm{}, ErrInvalidHashAlgorithm
	}

	return HashAlgorithm{value: value}, nil
}

func (h HashAlgorithm) String() string {
	return h.value
}
