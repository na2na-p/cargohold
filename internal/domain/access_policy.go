package domain

import "time"

type AccessPolicy struct {
	id         AccessPolicyID
	oid        OID
	repository *RepositoryIdentifier
	createdAt  time.Time
}

func NewAccessPolicy(id AccessPolicyID, oid OID, repository *RepositoryIdentifier, createdAt time.Time) *AccessPolicy {
	return &AccessPolicy{
		id:         id,
		oid:        oid,
		repository: repository,
		createdAt:  createdAt,
	}
}

func (ap *AccessPolicy) ID() AccessPolicyID {
	return ap.id
}

func (ap *AccessPolicy) OID() OID {
	return ap.oid
}

func (ap *AccessPolicy) Repository() *RepositoryIdentifier {
	return ap.repository
}

func (ap *AccessPolicy) CreatedAt() time.Time {
	return ap.createdAt
}
