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
	s3Client S3Client,
	policyRepo domain.AccessPolicyRepository,
	storageKeyGenerator StorageKeyGenerator,
) *BatchUseCase {
	downloadUseCase := NewDownloadUseCase(repo, s3Client)
	uploadUseCase := NewUploadUseCase(repo, s3Client, storageKeyGenerator)
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

func (uc *BatchUseCase) HandleBatchRequest(ctx context.Context, req BatchRequest) (BatchResponse, error) {
	if req.Operation() == domain.OperationDownload {
		return uc.batchDownloadUseCase.HandleBatchDownload(ctx, req)
	}
	return uc.batchUploadUseCase.HandleBatchUpload(ctx, req)
}
