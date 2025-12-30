package oidc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/na2na-p/cargohold/internal/domain"
)

type gitHubRepositoryChecker struct {
	httpClient  *http.Client
	apiEndpoint string
}

func NewGitHubRepositoryChecker() *gitHubRepositoryChecker {
	return &gitHubRepositoryChecker{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		apiEndpoint: GitHubAPIBaseURL,
	}
}

func (c *gitHubRepositoryChecker) SetAPIEndpoint(endpoint string) {
	c.apiEndpoint = endpoint
}

func (c *gitHubRepositoryChecker) CanAccessRepository(ctx context.Context, token *oauthToken, repo *domain.RepositoryIdentifier) (bool, error) {
	if token == nil {
		return false, fmt.Errorf("token is required")
	}
	if token.AccessToken == "" {
		return false, fmt.Errorf("access token is required")
	}

	if repo == nil {
		return false, fmt.Errorf("repository is required")
	}

	repoURL := fmt.Sprintf("%s/repos/%s/%s", c.apiEndpoint, repo.Owner(), repo.Name())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, repoURL, nil)
	if err != nil {
		return false, fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("リポジトリアクセス確認リクエストに失敗しました: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound, http.StatusForbidden:
		return false, nil
	default:
		return false, fmt.Errorf("リポジトリアクセス確認に失敗しました: status=%d", resp.StatusCode)
	}
}
