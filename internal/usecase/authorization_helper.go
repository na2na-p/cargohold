package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/na2na-p/cargohold/internal/domain"
)

func CheckAuthorization(
	ctx context.Context,
	authService domain.AccessAuthorizationService,
	op domain.Operation,
	userRepo *domain.RepositoryIdentifier,
	oid domain.OID,
) (domain.AuthorizationResult, error) {
	result, err := authService.Authorize(ctx, op, userRepo, oid)
	if err != nil {
		if errors.Is(err, domain.ErrAuthorizationDenied) || errors.Is(err, domain.ErrInvalidRepositoryIdentifier) {
			return result, ErrAccessDenied
		}
		return result, fmt.Errorf("認可判定に失敗しました: %w", err)
	}
	return result, nil
}
