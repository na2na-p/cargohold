package dto

import (
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
)

type BatchRequestDTO struct {
	Operation string             `json:"operation"`
	Objects   []RequestObjectDTO `json:"objects"`
	Transfers []string           `json:"transfers,omitempty"`
	Ref       *RefInfoDTO        `json:"ref,omitempty"`
	HashAlgo  string             `json:"hash_algo,omitempty"`
}

type RequestObjectDTO struct {
	OID  string `json:"oid"`
	Size int64  `json:"size"`
}

type RefInfoDTO struct {
	Name string `json:"name"`
}

func (d *BatchRequestDTO) ToBatchRequest(repo *domain.RepositoryIdentifier) (usecase.BatchRequest, error) {
	op, err := domain.ParseOperation(d.Operation)
	if err != nil {
		return usecase.BatchRequest{}, err
	}

	objects := make([]usecase.RequestObject, len(d.Objects))
	for i, obj := range d.Objects {
		objects[i] = usecase.NewRequestObject(obj.OID, obj.Size)
	}

	var refInfo *usecase.RefInfo
	if d.Ref != nil {
		r := usecase.NewRefInfo(d.Ref.Name)
		refInfo = &r
	}

	return usecase.NewBatchRequest(op, objects, d.Transfers, refInfo, d.HashAlgo, repo), nil
}
