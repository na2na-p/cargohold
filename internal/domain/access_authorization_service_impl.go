package domain

import (
	"context"
	"errors"
)

type accessAuthorizationServiceImpl struct {
	policyRepo AccessPolicyRepository
}

func NewAccessAuthorizationService(policyRepo AccessPolicyRepository) AccessAuthorizationService {
	return &accessAuthorizationServiceImpl{
		policyRepo: policyRepo,
	}
}

func (s *accessAuthorizationServiceImpl) CanAccess(ctx context.Context, userRepo *RepositoryIdentifier, oid OID) (bool, error) {
	if userRepo == nil {
		return false, ErrInvalidRepositoryIdentifier
	}

	policy, err := s.policyRepo.FindByOID(ctx, oid)
	if err != nil {
		return false, err
	}

	if policy == nil {
		return false, ErrAccessPolicyNotFound
	}

	return userRepo.Equals(policy.Repository()), nil
}

func (s *accessAuthorizationServiceImpl) Authorize(ctx context.Context, operation Operation, userRepo *RepositoryIdentifier, oid OID) (AuthorizationResult, error) {
	if userRepo == nil {
		return AuthorizationResult{Allowed: false, IsNewObject: false}, ErrInvalidRepositoryIdentifier
	}

	policy, err := s.policyRepo.FindByOID(ctx, oid)
	if err != nil && !errors.Is(err, ErrAccessPolicyNotFound) {
		return AuthorizationResult{Allowed: false, IsNewObject: false}, err
	}

	if policy == nil || errors.Is(err, ErrAccessPolicyNotFound) {
		if operation == OperationUpload {
			return AuthorizationResult{Allowed: true, IsNewObject: true}, nil
		}
		return AuthorizationResult{Allowed: false, IsNewObject: false}, ErrAuthorizationDenied
	}

	if !userRepo.Equals(policy.Repository()) {
		return AuthorizationResult{Allowed: false, IsNewObject: false}, ErrAuthorizationDenied
	}

	return AuthorizationResult{Allowed: true, IsNewObject: false}, nil
}
