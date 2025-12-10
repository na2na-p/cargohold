//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_batch_usecase.go -package=usecase
package usecase

import (
	"context"
	"time"

	"github.com/na2na-p/cargohold/internal/domain"
)

const (
	DefaultTransferType  = "basic"
	DefaultHashAlgorithm = "sha256"
	PresignedURLTTL      = 15 * time.Minute
)

type BatchUseCase struct {
	batchDownloadUseCase BatchDownloadUseCase
	batchUploadUseCase   BatchUploadUseCase
}

func NewBatchUseCase(
	repo domain.LFSObjectRepository,
	actionURLGenerator ActionURLGenerator,
	policyRepo domain.AccessPolicyRepository,
	storageKeyGenerator StorageKeyGenerator,
) *BatchUseCase {
	downloadUseCase := NewDownloadUseCase(repo, actionURLGenerator)
	uploadUseCase := NewUploadUseCase(repo, actionURLGenerator, storageKeyGenerator)
	authService := domain.NewAccessAuthorizationService(policyRepo)

	batchDownloadUseCase := NewBatchDownloadUseCase(downloadUseCase, authService)
	batchUploadUseCase := NewBatchUploadUseCase(uploadUseCase, authService, policyRepo)

	return &BatchUseCase{
		batchDownloadUseCase: batchDownloadUseCase,
		batchUploadUseCase:   batchUploadUseCase,
	}
}

func NewBatchUseCaseWithDependencies(
	batchDownloadUseCase BatchDownloadUseCase,
	batchUploadUseCase BatchUploadUseCase,
) *BatchUseCase {
	return &BatchUseCase{
		batchDownloadUseCase: batchDownloadUseCase,
		batchUploadUseCase:   batchUploadUseCase,
	}
}

func (uc *BatchUseCase) HandleBatchRequest(ctx context.Context, baseURL, owner, repo string, req BatchRequest) (BatchResponse, error) {
	if req.Operation() == domain.OperationDownload {
		return uc.batchDownloadUseCase.HandleBatchDownload(ctx, baseURL, owner, repo, req)
	}
	return uc.batchUploadUseCase.HandleBatchUpload(ctx, baseURL, owner, repo, req)
}
