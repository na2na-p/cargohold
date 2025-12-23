package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxResponseSize = 1 << 20

type gitHubTokenExchanger struct {
	clientID      string
	clientSecret  string
	redirectURI   string
	httpClient    *http.Client
	tokenEndpoint string
}

func NewGitHubTokenExchanger(clientID, clientSecret, redirectURI string) (*gitHubTokenExchanger, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, fmt.Errorf("clientID is required")
	}

	clientSecret = strings.TrimSpace(clientSecret)
	if clientSecret == "" {
		return nil, fmt.Errorf("clientSecret is required")
	}

	return &gitHubTokenExchanger{
		clientID:      clientID,
		clientSecret:  clientSecret,
		redirectURI:   strings.TrimSpace(redirectURI),
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		tokenEndpoint: GitHubOAuthTokenURL,
	}, nil
}

func (e *gitHubTokenExchanger) SetRedirectURI(redirectURI string) {
	e.redirectURI = strings.TrimSpace(redirectURI)
}

func (e *gitHubTokenExchanger) SetTokenEndpoint(endpoint string) {
	e.tokenEndpoint = endpoint
}

func (e *gitHubTokenExchanger) GetAuthorizationURL(state string, scopes []string) string {
	params := url.Values{}
	params.Set("client_id", e.clientID)
	params.Set("redirect_uri", e.redirectURI)
	params.Set("state", state)

	if len(scopes) > 0 {
		params.Set("scope", strings.Join(scopes, " "))
	}

	return fmt.Sprintf("%s?%s", GitHubOAuthAuthorizeURL, params.Encode())
}

func (e *gitHubTokenExchanger) ExchangeCode(ctx context.Context, code string) (*OAuthToken, error) {
	data := url.Values{}
	data.Set("client_id", e.clientID)
	data.Set("client_secret", e.clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", e.redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("トークン取得リクエストに失敗しました: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("トークン取得に失敗しました: status=%d", resp.StatusCode)
	}

	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("レスポンスの読み取りに失敗しました: %w", err)
	}
	if len(body) > maxResponseSize {
		return nil, fmt.Errorf("レスポンスが大きすぎます: %d bytes (最大: %d bytes)", len(body), maxResponseSize)
	}

	var tokenResp struct {
		AccessToken      string `json:"access_token"`
		TokenType        string `json:"token_type"`
		Scope            string `json:"scope"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("レスポンスのパースに失敗しました: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("トークン取得エラー: %s - %s", tokenResp.Error, tokenResp.ErrorDescription)
	}

	return &OAuthToken{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		Scope:       tokenResp.Scope,
	}, nil
}
