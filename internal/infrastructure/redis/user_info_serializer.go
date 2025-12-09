//go:generate mockgen -source=$GOFILE -destination=../../../tests/infrastructure/redis/mock_user_info_serializer.go -package=redis
package redis

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/na2na-p/cargohold/internal/domain"
)

var (
	ErrNilUserInfo       = errors.New("userInfo is nil")
	ErrNilGitHubUserInfo = errors.New("gitHubUserInfo is nil")
)

type UserInfoSerializer interface {
	Serialize(userInfo *domain.UserInfo) ([]byte, error)
	Deserialize(data []byte) (*domain.UserInfo, error)
}

type GitHubUserInfoSerializer interface {
	Serialize(userInfo *domain.GitHubUserInfo) ([]byte, error)
	Deserialize(data []byte) (*domain.GitHubUserInfo, error)
}

type userInfoDTO struct {
	Sub        string              `json:"sub"`
	Email      string              `json:"email"`
	Name       string              `json:"name"`
	Provider   domain.ProviderType `json:"provider"`
	Repository string              `json:"repository,omitempty"`
	Ref        string              `json:"ref,omitempty"`
}

type gitHubUserInfoDTO struct {
	Sub        string `json:"sub"`
	Repository string `json:"repository"`
	Ref        string `json:"ref"`
	Actor      string `json:"actor"`
}

type userInfoSerializerImpl struct{}

func NewUserInfoSerializer() UserInfoSerializer {
	return &userInfoSerializerImpl{}
}

func (s *userInfoSerializerImpl) Serialize(userInfo *domain.UserInfo) ([]byte, error) {
	if userInfo == nil {
		return nil, ErrNilUserInfo
	}

	var repoStr string
	if userInfo.Repository() != nil {
		repoStr = userInfo.Repository().FullName()
	}

	dto := &userInfoDTO{
		Sub:        userInfo.Sub(),
		Email:      userInfo.Email(),
		Name:       userInfo.Name(),
		Provider:   userInfo.Provider(),
		Repository: repoStr,
		Ref:        userInfo.Ref(),
	}

	data, err := json.Marshal(dto)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal UserInfo: %w", err)
	}
	return data, nil
}

func (s *userInfoSerializerImpl) Deserialize(data []byte) (*domain.UserInfo, error) {
	var dto userInfoDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("failed to unmarshal UserInfo: %w", err)
	}

	var repo *domain.RepositoryIdentifier
	if dto.Repository != "" {
		var err error
		repo, err = domain.NewRepositoryIdentifier(dto.Repository)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repository: %w", err)
		}
	}

	return domain.NewUserInfo(
		dto.Sub,
		dto.Email,
		dto.Name,
		dto.Provider,
		repo,
		dto.Ref,
	)
}

type gitHubUserInfoSerializerImpl struct{}

func NewGitHubUserInfoSerializer() GitHubUserInfoSerializer {
	return &gitHubUserInfoSerializerImpl{}
}

func (s *gitHubUserInfoSerializerImpl) Serialize(userInfo *domain.GitHubUserInfo) ([]byte, error) {
	if userInfo == nil {
		return nil, ErrNilGitHubUserInfo
	}

	dto := &gitHubUserInfoDTO{
		Sub:        userInfo.Sub(),
		Repository: userInfo.Repository(),
		Ref:        userInfo.Ref(),
		Actor:      userInfo.Actor(),
	}

	data, err := json.Marshal(dto)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GitHubUserInfo: %w", err)
	}
	return data, nil
}

func (s *gitHubUserInfoSerializerImpl) Deserialize(data []byte) (*domain.GitHubUserInfo, error) {
	var dto gitHubUserInfoDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GitHubUserInfo: %w", err)
	}

	return domain.NewGitHubUserInfo(
		dto.Sub,
		dto.Repository,
		dto.Ref,
		dto.Actor,
	), nil
}
