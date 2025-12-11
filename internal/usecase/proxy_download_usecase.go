//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_proxy_download_usecase.go -package=usecase
package usecase

import (
	"context"
	"errors"
	"io"

	"github.com/na2na-p/cargohold/internal/domain"
)

type ProxyDownloadUseCase interface {
	Execute(ctx context.Context, owner, repo string, oid domain.OID) (io.ReadCloser, int64, error)
}

type proxyDownloadUseCaseImpl struct {
	repo          domain.LFSObjectRepository
	objectStorage ObjectStorage
	authService   domain.AccessAuthorizationService
}

func NewProxyDownloadUseCase(
	repo domain.LFSObjectRepository,
	objectStorage ObjectStorage,
	authService domain.AccessAuthorizationService,
) ProxyDownloadUseCase {
	return &proxyDownloadUseCaseImpl{
		repo:          repo,
		objectStorage: objectStorage,
		authService:   authService,
	}
}

func (u *proxyDownloadUseCaseImpl) Execute(ctx context.Context, owner, repo string, oid domain.OID) (io.ReadCloser, int64, error) {
	repoIdentifier, err := domain.NewRepositoryIdentifier(owner + "/" + repo)
	if err != nil {
		return nil, 0, ErrAccessDenied
	}

	authResult, err := CheckAuthorization(ctx, u.authService, domain.OperationDownload, repoIdentifier, oid)
	if err != nil {
		return nil, 0, err
	}
	if !authResult.Allowed {
		return nil, 0, ErrAccessDenied
	}

	obj, err := u.repo.FindByOID(ctx, oid)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, 0, ErrObjectNotFound
		}
		return nil, 0, err
	}

	if !obj.IsUploaded() {
		return nil, 0, ErrNotUploaded
	}

	storageKey := obj.GetStorageKey()
	size := obj.Size().Int64()

	stream, err := u.objectStorage.GetObject(ctx, storageKey)
	if err != nil {
		return nil, 0, err
	}

	return stream, size, nil
}
