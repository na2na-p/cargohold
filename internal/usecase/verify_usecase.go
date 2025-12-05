package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/na2na-p/cargohold/internal/domain"
)

type VerifyUseCase struct {
	repo            domain.LFSObjectRepository
	cacheKeyManager CacheKeyManager
}

func NewVerifyUseCase(
	repo domain.LFSObjectRepository,
	cacheKeyManager CacheKeyManager,
) *VerifyUseCase {
	return &VerifyUseCase{
		repo:            repo,
		cacheKeyManager: cacheKeyManager,
	}
}

func (uc *VerifyUseCase) VerifyUpload(ctx context.Context, oid string, size int64) error {
	domainOID, err := domain.NewOID(oid)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidOIDFormat) {
			return ErrInvalidOID
		}
		return fmt.Errorf("OIDの検証に失敗しました: %w", err)
	}

	domainSize, err := domain.NewSize(size)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidSize) {
			return ErrInvalidSize
		}
		return fmt.Errorf("サイズの検証に失敗しました: %w", err)
	}

	obj, err := uc.repo.FindByOID(ctx, domainOID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrObjectNotFound
		}
		return fmt.Errorf("メタデータの取得に失敗しました: %w", err)
	}

	if obj.Size().Int64() != domainSize.Int64() {
		return ErrSizeMismatch
	}

	obj.MarkAsUploaded(ctx)

	if err := uc.repo.Update(ctx, obj); err != nil {
		return fmt.Errorf("メタデータの更新に失敗しました: %w", err)
	}

	if err := uc.cacheKeyManager.DeleteBatchUploadKey(ctx, oid); err != nil {
		slog.Warn("BatchUploadKeyの削除に失敗しました", "oid", oid, "error", err)
	}

	return nil
}
