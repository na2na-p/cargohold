package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type gitHubUserInfoProvider struct {
	httpClient       *http.Client
	userInfoEndpoint string
}

func NewGitHubUserInfoProvider() *gitHubUserInfoProvider {
	return &gitHubUserInfoProvider{
		httpClient:       &http.Client{Timeout: 10 * time.Second},
		userInfoEndpoint: GitHubAPIUserURL,
	}
}

func (p *gitHubUserInfoProvider) SetUserInfoEndpoint(endpoint string) {
	p.userInfoEndpoint = endpoint
}

func (p *gitHubUserInfoProvider) GetUserInfo(ctx context.Context, token *OAuthToken) (*GitHubUser, error) {
	if token == nil {
		return nil, fmt.Errorf("token is required")
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ユーザー情報取得リクエストに失敗しました: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ユーザー情報取得に失敗しました: status=%d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンスの読み取りに失敗しました: %w", err)
	}

	var userResp struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
	}

	if err := json.Unmarshal(body, &userResp); err != nil {
		return nil, fmt.Errorf("レスポンスのパースに失敗しました: %w", err)
	}

	return &GitHubUser{
		ID:    userResp.ID,
		Login: userResp.Login,
		Name:  userResp.Name,
	}, nil
}
