//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_batch_upload_usecase.go -package=usecase
package usecase

import (
	"context"
	"fmt"

	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/newmo-oss/ctxtime"
)

type BatchUploadUseCase interface {
	HandleBatchUpload(ctx context.Context, baseURL, owner, repo string, req BatchRequest, authHeader string) (BatchResponse, error)
}

type batchUploadUseCaseImpl struct {
	uploadUseCase UploadUseCase
	authService   domain.AccessAuthorizationService
	policyRepo    domain.AccessPolicyRepository
}

func NewBatchUploadUseCase(
	uploadUseCase UploadUseCase,
	authService domain.AccessAuthorizationService,
	policyRepo domain.AccessPolicyRepository,
) BatchUploadUseCase {
	return &batchUploadUseCaseImpl{
		uploadUseCase: uploadUseCase,
		authService:   authService,
		policyRepo:    policyRepo,
	}
}

func (uc *batchUploadUseCaseImpl) HandleBatchUpload(ctx context.Context, baseURL, owner, repo string, req BatchRequest, authHeader string) (BatchResponse, error) {
	if err := req.Validate(); err != nil {
		return BatchResponse{}, err
	}

	if req.Operation() != domain.OperationUpload {
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

		authResult, err := CheckAuthorization(ctx, uc.authService, domain.OperationUpload, req.Repository(), oid)
		if err != nil {
			return BatchResponse{}, err
		}

		if !authResult.Allowed {
			return BatchResponse{}, ErrAccessDenied
		}

		respObj := uc.uploadUseCase.HandleUploadObject(ctx, baseURL, owner, repo, oid, size, hashAlgo, authHeader)
		if authResult.IsNewObject && respObj.Error() == nil {
			if err := uc.createAccessPolicy(ctx, oid, req.Repository()); err != nil {
				return BatchResponse{}, fmt.Errorf("アクセスポリシーの作成に失敗しました: %w", err)
			}
		}

		objects = append(objects, respObj)
	}

	return NewBatchResponse(DefaultTransferType, objects, hashAlgo), nil
}

func (uc *batchUploadUseCaseImpl) createAccessPolicy(ctx context.Context, oid domain.OID, repo *domain.RepositoryIdentifier) error {
	policyID, err := domain.NewAccessPolicyID(0)
	if err != nil {
		return err
	}
	policy := domain.NewAccessPolicy(policyID, oid, repo, ctxtime.Now(ctx))
	return uc.policyRepo.Save(ctx, policy)
}
