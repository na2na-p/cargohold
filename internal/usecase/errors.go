package usecase

import "errors"

var (
	// ErrInvalidOperation は不正なオペレーション種別の場合のエラー
	ErrInvalidOperation = errors.New("operation must be 'upload' or 'download'")

	// ErrNoObjects はオブジェクトが空の場合のエラー
	ErrNoObjects = errors.New("objects must not be empty")

	// ErrInvalidOID はOID形式が不正な場合のエラー
	ErrInvalidOID = errors.New("invalid OID format")

	// ErrInvalidSize はサイズが不正な場合のエラー
	ErrInvalidSize = errors.New("invalid size")

	// ErrObjectNotFound はオブジェクトが見つからない場合のエラー
	ErrObjectNotFound = errors.New("object not found")

	// ErrSizeMismatch はサイズが一致しない場合のエラー
	ErrSizeMismatch = errors.New("size mismatch")

	// ErrS3URLGeneration はS3署名付きURL生成に失敗した場合のエラー
	ErrS3URLGeneration = errors.New("failed to generate S3 presigned URL")

	// ErrMetadataCache はメタデータキャッシュの操作に失敗した場合のエラー
	ErrMetadataCache = errors.New("failed to operate metadata cache")

	// ErrAuthenticationFailed は認証失敗を表すエラーです
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrInvalidState はstateパラメータが無効な場合のエラーです
	ErrInvalidState = errors.New("invalid state parameter")

	// ErrSessionCreationFailed はセッション作成に失敗した場合のエラーです
	ErrSessionCreationFailed = errors.New("session creation failed")

	// ErrSessionNotFound はセッションが見つからない場合のエラーです
	ErrSessionNotFound = errors.New("session not found")

	// ErrInvalidSessionData はセッションデータの形式が不正な場合のエラーです
	ErrInvalidSessionData = errors.New("invalid session data format")

	// ErrInvalidRepository はリポジトリ識別子が不正な場合のエラーです
	ErrInvalidRepository = errors.New("invalid repository identifier")

	// ErrInvalidRedirectURI はリダイレクトURIが不正な場合のエラーです
	ErrInvalidRedirectURI = errors.New("invalid redirect URI")

	// ErrStateSaveFailed はstate保存に失敗した場合のエラーです
	ErrStateSaveFailed = errors.New("failed to save state parameter")

	// ErrCodeExchangeFailed はコード交換に失敗した場合のエラーです
	ErrCodeExchangeFailed = errors.New("failed to exchange authorization code")

	// ErrUserInfoFailed はユーザー情報取得に失敗した場合のエラーです
	ErrUserInfoFailed = errors.New("failed to get user info")

	// ErrUserInfoCreationFailed はユーザー情報の生成に失敗した場合のエラーです
	ErrUserInfoCreationFailed = errors.New("failed to create user info")

	// ErrRepositoryAccessDenied はリポジトリへのアクセス権がない場合のエラーです
	ErrRepositoryAccessDenied = errors.New("repository access denied")

	// ErrRepositoryAccessCheckFailed はリポジトリアクセス権の検証に失敗した場合のエラーです
	ErrRepositoryAccessCheckFailed = errors.New("failed to check repository access")

	// ErrAccessDenied は認可判定に失敗した場合のエラーです（LFSオブジェクトへのアクセス権がない）
	ErrAccessDenied = errors.New("access denied to LFS object")

	// ErrInvalidHashAlgorithm は無効なハッシュアルゴリズムの場合のエラー
	ErrInvalidHashAlgorithm = errors.New("invalid hash algorithm")

	// ErrInvalidCode は認証コードが無効な場合のエラーです
	ErrInvalidCode = errors.New("invalid authorization code")

	// ErrGitHubOIDCNotConfigured はGitHub OIDC認証が設定されていない場合のエラーです
	ErrGitHubOIDCNotConfigured = errors.New("GitHub OIDC authentication is not configured")

	// ErrCacheMiss はキャッシュにデータが存在しない場合のエラーです
	ErrCacheMiss = errors.New("cache miss")

	// ErrNotUploaded はオブジェクトがまだアップロードされていない場合のエラーです
	ErrNotUploaded = errors.New("object not uploaded yet")
)
