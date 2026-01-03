//go:generate mockgen -source=$GOFILE -destination=../../tests/usecase/mock_upload_usecase.go -package=usecase
package usecase

import (
	"context"
	"errors"

	"github.com/na2na-p/cargohold/internal/domain"
)

type UploadUseCase interface {
	HandleUploadObject(ctx context.Context, baseURL, owner, repo string, oid domain.OID, size domain.Size, hashAlgo string, authHeader string) ResponseObject
}

type uploadUseCaseImpl struct {
	repo                domain.LFSObjectRepository
	actionURLGenerator  ActionURLGenerator
	storageKeyGenerator StorageKeyGenerator
}

func NewUploadUseCase(
	repo domain.LFSObjectRepository,
	actionURLGenerator ActionURLGenerator,
	storageKeyGenerator StorageKeyGenerator,
) UploadUseCase {
	return &uploadUseCaseImpl{
		repo:                repo,
		actionURLGenerator:  actionURLGenerator,
		storageKeyGenerator: storageKeyGenerator,
	}
}

func (uc *uploadUseCaseImpl) HandleUploadObject(ctx context.Context, baseURL, owner, repo string, oid domain.OID, size domain.Size, hashAlgo string, authHeader string) ResponseObject {
	obj, err := uc.repo.FindByOID(ctx, oid)

	var storageKey string
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			objectError := NewObjectError(500, "メタデータの取得に失敗しました")
			return NewResponseObject(oid.String(), size.Int64(), false, nil, &objectError)
		}
		storageKey, err = uc.storageKeyGenerator.GenerateStorageKey(oid.String(), hashAlgo)
		if err != nil {
			objectError := NewObjectError(400, "無効なストレージキーパラメータです")
			return NewResponseObject(oid.String(), size.Int64(), false, nil, &objectError)
		}
		hashAlgorithm, err := domain.NewHashAlgorithm(hashAlgo)
		if err != nil {
			objectError := NewObjectError(400, "無効なハッシュアルゴリズムです")
			return NewResponseObject(oid.String(), size.Int64(), false, nil, &objectError)
		}
		newObj, err := domain.NewLFSObject(ctx, oid, size, hashAlgorithm, storageKey)
		if err != nil {
			objectError := NewObjectError(500, "メタデータの作成に失敗しました")
			return NewResponseObject(oid.String(), size.Int64(), false, nil, &objectError)
		}
		if saveErr := uc.repo.Save(ctx, newObj); saveErr != nil {
			objectError := NewObjectError(500, "メタデータの保存に失敗しました")
			return NewResponseObject(oid.String(), size.Int64(), false, nil, &objectError)
		}
	} else if obj.IsUploaded() {
		if obj.Size().Int64() != size.Int64() {
			objectError := NewObjectError(409, "オブジェクトサイズが一致しません")
			return NewResponseObject(oid.String(), size.Int64(), false, nil, &objectError)
		}
		return NewResponseObject(oid.String(), size.Int64(), true, nil, nil)
	}

	uploadURL := uc.actionURLGenerator.GenerateUploadURL(baseURL, owner, repo, oid.String())

	header := map[string]string{}
	if authHeader != "" {
		header["Authorization"] = authHeader
	}
	uploadAction := NewAction(uploadURL, header, int(PresignedURLTTL.Seconds()))
	actions := NewActions(&uploadAction, nil)
	return NewResponseObject(oid.String(), size.Int64(), true, &actions, nil)
}
