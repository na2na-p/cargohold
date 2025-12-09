package domain

import (
	"errors"
	"regexp"
)

type OID struct {
	value string
}

var (
	ErrInvalidOIDFormat = errors.New("OID must be a 64-character hexadecimal string")
	oidPattern          = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
)

func NewOID(value string) (OID, error) {
	if !oidPattern.MatchString(value) {
		return OID{}, ErrInvalidOIDFormat
	}

	return OID{value: value}, nil
}

func (o OID) String() string {
	return o.value
}
