package domain

import "errors"

var ErrInvalidAllowedRepositoryFormat = errors.New("allowed repository must be in 'owner/repo' format")

type AllowedRepository struct {
	identifier *RepositoryIdentifier
}

func NewAllowedRepository(owner, repo string) (*AllowedRepository, error) {
	if owner == "" || repo == "" {
		return nil, ErrInvalidAllowedRepositoryFormat
	}

	identifier, err := NewRepositoryIdentifier(owner + "/" + repo)
	if err != nil {
		return nil, ErrInvalidAllowedRepositoryFormat
	}

	return &AllowedRepository{
		identifier: identifier,
	}, nil
}

func NewAllowedRepositoryFromString(fullName string) (*AllowedRepository, error) {
	identifier, err := NewRepositoryIdentifier(fullName)
	if err != nil {
		return nil, ErrInvalidAllowedRepositoryFormat
	}

	return &AllowedRepository{
		identifier: identifier,
	}, nil
}

func (ar *AllowedRepository) Owner() string {
	return ar.identifier.Owner()
}

func (ar *AllowedRepository) Repo() string {
	return ar.identifier.Name()
}

func (ar *AllowedRepository) String() string {
	return ar.identifier.FullName()
}

func (ar *AllowedRepository) Equals(other *AllowedRepository) bool {
	if ar == nil {
		return other == nil
	}
	if other == nil {
		return false
	}
	return ar.identifier.Equals(other.identifier)
}
