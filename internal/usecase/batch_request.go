package usecase

import (
	"github.com/na2na-p/cargohold/internal/domain"
)

type BatchRequest struct {
	operation  domain.Operation
	objects    []RequestObject
	transfers  []string
	ref        *RefInfo
	hashAlgo   string
	repository *domain.RepositoryIdentifier
}

func NewBatchRequest(
	operation domain.Operation,
	objects []RequestObject,
	transfers []string,
	ref *RefInfo,
	hashAlgo string,
	repository *domain.RepositoryIdentifier,
) BatchRequest {
	objectsCopy := make([]RequestObject, len(objects))
	copy(objectsCopy, objects)

	transfersCopy := make([]string, len(transfers))
	copy(transfersCopy, transfers)

	return BatchRequest{
		operation:  operation,
		objects:    objectsCopy,
		transfers:  transfersCopy,
		ref:        ref,
		hashAlgo:   hashAlgo,
		repository: repository,
	}
}

func (r BatchRequest) Operation() domain.Operation {
	return r.operation
}

func (r BatchRequest) Objects() []RequestObject {
	result := make([]RequestObject, len(r.objects))
	copy(result, r.objects)
	return result
}

func (r BatchRequest) Transfers() []string {
	result := make([]string, len(r.transfers))
	copy(result, r.transfers)
	return result
}

func (r BatchRequest) Ref() *RefInfo {
	return r.ref
}

func (r BatchRequest) HashAlgo() string {
	return r.hashAlgo
}

func (r BatchRequest) Repository() *domain.RepositoryIdentifier {
	return r.repository
}

func (r BatchRequest) WithRepository(repo *domain.RepositoryIdentifier) BatchRequest {
	return BatchRequest{
		operation:  r.operation,
		objects:    r.objects,
		transfers:  r.transfers,
		ref:        r.ref,
		hashAlgo:   r.hashAlgo,
		repository: repo,
	}
}

func (r BatchRequest) Validate() error {
	if r.operation != domain.OperationUpload && r.operation != domain.OperationDownload {
		return ErrInvalidOperation
	}

	if len(r.objects) == 0 {
		return ErrNoObjects
	}

	for _, obj := range r.objects {
		if _, err := domain.NewOID(obj.OID()); err != nil {
			return ErrInvalidOID
		}
		if _, err := domain.NewSize(obj.Size()); err != nil {
			return ErrInvalidSize
		}
	}

	if _, err := domain.NewHashAlgorithm(r.hashAlgo); err != nil {
		return ErrInvalidHashAlgorithm
	}

	return nil
}

type RequestObject struct {
	oid  string
	size int64
}

func NewRequestObject(oid string, size int64) RequestObject {
	return RequestObject{
		oid:  oid,
		size: size,
	}
}

func (r RequestObject) OID() string {
	return r.oid
}

func (r RequestObject) Size() int64 {
	return r.size
}

type RefInfo struct {
	name string
}

func NewRefInfo(name string) RefInfo {
	return RefInfo{name: name}
}

func (r RefInfo) Name() string {
	return r.name
}
