//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_batch_download_usecase.go -package=usecase
package usecase

import (
	"context"
	"fmt"

	"github.com/na2na-p/cargohold/internal/domain"
)

type BatchDownloadUseCase interface {
	HandleBatchDownload(ctx context.Context, req BatchRequest) (BatchResponse, error)
}

type batchDownloadUseCaseImpl struct {
	downloadUseCase DownloadUseCase
	authService     domain.AccessAuthorizationService
}

func NewBatchDownloadUseCase(
	downloadUseCase DownloadUseCase,
	authService domain.AccessAuthorizationService,
) BatchDownloadUseCase {
	return &batchDownloadUseCaseImpl{
		downloadUseCase: downloadUseCase,
		authService:     authService,
	}
}

func (uc *batchDownloadUseCaseImpl) HandleBatchDownload(ctx context.Context, req BatchRequest) (BatchResponse, error) {
	if err := req.Validate(); err != nil {
		return BatchResponse{}, err
	}

	if req.Operation() != domain.OperationDownload {
		return BatchResponse{}, ErrInvalidOperation
	}

	hashAlgo := req.HashAlgo()
	if hashAlgo == "" {
		hashAlgo = DefaultHashAlgorithm
	}

	objects := make([]ResponseObject, 0, len(req.Objects()))

	for _, reqObj := range req.Objects() {
		oid, err := domain.NewOID(reqObj.OID())
		if err != nil {
			return BatchResponse{}, fmt.Errorf("無効なOID: %w", err)
		}
		size, err := domain.NewSize(reqObj.Size())
		if err != nil {
			return BatchResponse{}, fmt.Errorf("無効なサイズ: %w", err)
		}

		authResult, err := CheckAuthorization(ctx, uc.authService, domain.OperationDownload, req.Repository(), oid)
		if err != nil {
			return BatchResponse{}, err
		}

		if !authResult.Allowed {
			return BatchResponse{}, ErrAccessDenied
		}

		respObj := uc.downloadUseCase.HandleDownloadObject(ctx, oid, size)
		objects = append(objects, respObj)
	}

	return NewBatchResponse(DefaultTransferType, objects, hashAlgo), nil
}
