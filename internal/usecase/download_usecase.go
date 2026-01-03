//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_download_usecase.go -package=usecase
package usecase

import (
	"context"
	"errors"

	"github.com/na2na-p/cargohold/internal/domain"
)

type DownloadUseCase interface {
	HandleDownloadObject(ctx context.Context, baseURL, owner, repo string, oid domain.OID, size domain.Size, authHeader string) ResponseObject
}

type downloadUseCaseImpl struct {
	repo               domain.LFSObjectRepository
	actionURLGenerator ActionURLGenerator
}

func NewDownloadUseCase(
	repo domain.LFSObjectRepository,
	actionURLGenerator ActionURLGenerator,
) DownloadUseCase {
	return &downloadUseCaseImpl{
		repo:               repo,
		actionURLGenerator: actionURLGenerator,
	}
}

func (uc *downloadUseCaseImpl) HandleDownloadObject(ctx context.Context, baseURL, owner, repo string, oid domain.OID, size domain.Size, authHeader string) ResponseObject {
	obj, err := uc.repo.FindByOID(ctx, oid)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			objectError := NewObjectError(404, "オブジェクトが存在しません")
			return NewResponseObject(oid.String(), size.Int64(), false, nil, &objectError)
		}
		objectError := NewObjectError(500, "メタデータの取得に失敗しました")
		return NewResponseObject(oid.String(), size.Int64(), false, nil, &objectError)
	}

	if !obj.IsUploaded() {
		objectError := NewObjectError(404, "オブジェクトがまだアップロードされていません")
		return NewResponseObject(oid.String(), size.Int64(), false, nil, &objectError)
	}

	downloadURL := uc.actionURLGenerator.GenerateDownloadURL(baseURL, owner, repo, oid.String())

	header := map[string]string{}
	if authHeader != "" {
		header["Authorization"] = authHeader
	}
	downloadAction := NewAction(downloadURL, header, int(PresignedURLTTL.Seconds()))
	actions := NewActions(nil, &downloadAction)
	return NewResponseObject(oid.String(), size.Int64(), true, &actions, nil)
}
