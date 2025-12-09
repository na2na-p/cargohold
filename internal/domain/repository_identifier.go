package domain

import (
	"errors"
	"strings"
)

type RepositoryIdentifier struct {
	owner string
	name  string
}

var ErrInvalidRepositoryIdentifierFormat = errors.New("repository identifier must be in 'owner/repo' format")

func NewRepositoryIdentifier(fullName string) (*RepositoryIdentifier, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 {
		return nil, ErrInvalidRepositoryIdentifierFormat
	}

	owner := parts[0]
	name := parts[1]

	if owner == "" || name == "" {
		return nil, ErrInvalidRepositoryIdentifierFormat
	}

	return &RepositoryIdentifier{
		owner: owner,
		name:  name,
	}, nil
}

func (ri *RepositoryIdentifier) FullName() string {
	return ri.owner + "/" + ri.name
}

func (ri *RepositoryIdentifier) Owner() string {
	return ri.owner
}

func (ri *RepositoryIdentifier) Name() string {
	return ri.name
}

func (ri *RepositoryIdentifier) Equals(other *RepositoryIdentifier) bool {
	if ri == nil {
		return other == nil
	}
	if other == nil {
		return false
	}
	return ri.owner == other.owner && ri.name == other.name
}

func (ri *RepositoryIdentifier) EqualsFold(other *RepositoryIdentifier) bool {
	if ri == nil {
		return other == nil
	}
	if other == nil {
		return false
	}
	return strings.EqualFold(ri.owner, other.owner) && strings.EqualFold(ri.name, other.name)
}
