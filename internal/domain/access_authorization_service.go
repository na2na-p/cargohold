//go:generate mockgen -source=$GOFILE -destination=../../tests/domain/mock_access_authorization_service.go -package=domain
package domain

import (
	"context"
	"errors"
)

var (
	ErrAccessPolicyNotFound        = errors.New("access policy not found")
	ErrInvalidRepositoryIdentifier = errors.New("invalid repository identifier")
	ErrAuthorizationDenied         = errors.New("authorization denied")
)

type AuthorizationResult struct {
	Allowed     bool
	IsNewObject bool
}

type AccessAuthorizationService interface {
	CanAccess(ctx context.Context, userRepo *RepositoryIdentifier, oid OID) (bool, error)
	Authorize(ctx context.Context, operation Operation, userRepo *RepositoryIdentifier, oid OID) (AuthorizationResult, error)
}
