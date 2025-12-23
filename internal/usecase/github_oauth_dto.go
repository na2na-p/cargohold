package usecase

// OAuthTokenResult はOAuthトークン情報を表すDTO
type OAuthTokenResult struct {
	AccessToken string
	TokenType   string
	Scope       string
}

// GitHubUserResult はGitHubユーザー情報を表すDTO
type GitHubUserResult struct {
	ID    int64
	Login string
	Name  string
}
