//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_proxy_upload_usecase.go -package=usecase
package usecase

import (
	"context"
	"errors"
	"io"

	"github.com/na2na-p/cargohold/internal/domain"
)

type ProxyUploadUseCase interface {
	Execute(ctx context.Context, owner, repo string, oid domain.OID, body io.Reader) error
}

type proxyUploadUseCaseImpl struct {
	repo          domain.LFSObjectRepository
	objectStorage ObjectStorage
	authService   domain.AccessAuthorizationService
}

func NewProxyUploadUseCase(
	repo domain.LFSObjectRepository,
	objectStorage ObjectStorage,
	authService domain.AccessAuthorizationService,
) ProxyUploadUseCase {
	return &proxyUploadUseCaseImpl{
		repo:          repo,
		objectStorage: objectStorage,
		authService:   authService,
	}
}

func (u *proxyUploadUseCaseImpl) Execute(ctx context.Context, owner, repo string, oid domain.OID, body io.Reader) error {
	repoIdentifier, err := domain.NewRepositoryIdentifier(owner + "/" + repo)
	if err != nil {
		return ErrAccessDenied
	}

	authResult, err := CheckAuthorization(ctx, u.authService, domain.OperationUpload, repoIdentifier, oid)
	if err != nil {
		return err
	}
	if !authResult.Allowed {
		return ErrAccessDenied
	}

	lfsObject, err := u.repo.FindByOID(ctx, oid)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrObjectNotFound
		}
		return err
	}

	storageKey := lfsObject.GetStorageKey()
	if err := u.objectStorage.PutObject(ctx, storageKey, body, lfsObject.Size().Int64()); err != nil {
		return err
	}

	lfsObject.MarkAsUploaded(ctx)
	if err := u.repo.Update(ctx, lfsObject); err != nil {
		return err
	}

	return nil
}
