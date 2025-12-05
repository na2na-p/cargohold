package usecase

import "github.com/na2na-p/cargohold/internal/domain"

// VerifyRequest はVerifyエンドポイントのリクエストを表す
type VerifyRequest struct {
	OID  string `json:"oid"`
	Size int64  `json:"size"`
}

// Validate はリクエストのバリデーションを行う
func (r *VerifyRequest) Validate() error {
	if r.OID == "" {
		return ErrInvalidOID
	}

	if _, err := domain.NewOID(r.OID); err != nil {
		return ErrInvalidOID
	}

	if r.Size <= 0 {
		return ErrInvalidSize
	}

	return nil
}
