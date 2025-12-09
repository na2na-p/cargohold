//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_download_usecase.go -package=usecase
package usecase

import (
	"context"
	"errors"

	"github.com/na2na-p/cargohold/internal/domain"
)

type DownloadUseCase interface {
	HandleDownloadObject(ctx context.Context, oid domain.OID, size domain.Size) ResponseObject
}

type downloadUseCaseImpl struct {
	repo     domain.LFSObjectRepository
	s3Client S3Client
}

func NewDownloadUseCase(
	repo domain.LFSObjectRepository,
	s3Client S3Client,
) DownloadUseCase {
	return &downloadUseCaseImpl{
		repo:     repo,
		s3Client: s3Client,
	}
}

func (uc *downloadUseCaseImpl) HandleDownloadObject(ctx context.Context, oid domain.OID, size domain.Size) ResponseObject {
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

	downloadURL, err := uc.s3Client.GenerateGetURL(ctx, obj.GetStorageKey(), PresignedURLTTL)
	if err != nil {
		objectError := NewObjectError(500, "ダウンロードURLの生成に失敗しました")
		return NewResponseObject(oid.String(), size.Int64(), false, nil, &objectError)
	}

	downloadAction := NewAction(downloadURL, nil, int(PresignedURLTTL.Seconds()))
	actions := NewActions(nil, &downloadAction)
	return NewResponseObject(oid.String(), size.Int64(), true, &actions, nil)
}
