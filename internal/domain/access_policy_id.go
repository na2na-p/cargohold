package domain

import "errors"

var ErrInvalidAccessPolicyID = errors.New("AccessPolicyID must be a non-negative integer")

type AccessPolicyID struct {
	value int64
}

func NewAccessPolicyID(value int64) (AccessPolicyID, error) {
	if value < 0 {
		return AccessPolicyID{}, ErrInvalidAccessPolicyID
	}
	return AccessPolicyID{value: value}, nil
}

func (id AccessPolicyID) Int64() int64 {
	return id.value
}
