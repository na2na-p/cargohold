//go:build e2e

// Package e2e_test はE2Eテストを提供します
package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/e2e"
)

// TestGitHubOIDCAuthentication はGitHub Actions OIDC認証のE2Eテストを実施します
func TestGitHubOIDCAuthentication(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		batchEndpoint string
		repository    string
		ref           string
		sha           string
		actor         string
		oid           string // テストケースごとに一意なOIDを使用
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "正常系: GitHub OIDC認証が成功する",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/na2na-platform"),
				repository:    "na2na-p/na2na-platform",
				ref:           "refs/heads/main",
				sha:           "abc123",
				actor:         "github-actions[bot]",
				oid:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			wantErr: false,
		},
		{
			name: "正常系: 別のリポジトリでの認証が成功する",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/develop",
				sha:           "def456",
				actor:         "github-actions[bot]",
				oid:           "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ステップ1: GitHub OIDC JWTトークンを生成
			token, err := e2e.GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", tt.args.repository, tt.args.ref),
				"aud":        "cargohold",
				"repository": tt.args.repository,
				"ref":        tt.args.ref,
				"sha":        tt.args.sha,
				"actor":      tt.args.actor,
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			// ステップ2: Batch APIにBearerトークンを付けてリクエスト（テストケースごとに一意なOIDを使用）
			err = requestBatchAPIWithOIDC(
				tt.args.batchEndpoint,
				token,
				"upload",
				tt.args.oid,
				123456,
			)

			// 検証: 認証成功を確認
			if (err != nil) != tt.wantErr {
				t.Errorf("requestBatchAPIWithOIDC() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGitHubOIDCAuthentication_InvalidToken は無効なトークンでの認証失敗をテストします
func TestGitHubOIDCAuthentication_InvalidToken(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		batchEndpoint string
		token         string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "異常系: 無効なトークンの場合、認証エラーが返る",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				token:         "invalid.token.here",
			},
			wantErr: true,
		},
		{
			name: "異常系: 空のトークンの場合、認証エラーが返る",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				token:         "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ステップ1: Batch APIにBearerトークンを付けてリクエスト
			err := requestBatchAPIWithOIDC(
				tt.args.batchEndpoint,
				tt.args.token,
				"upload",
				"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				123456,
			)

			// 検証: 認証エラーが返されることを確認
			if (err != nil) != tt.wantErr {
				t.Errorf("requestBatchAPIWithOIDC() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGitHubOIDCAuthentication_ExpiredToken は期限切れトークンでの認証失敗をテストします
func TestGitHubOIDCAuthentication_ExpiredToken(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		batchEndpoint string
		repository    string
		ref           string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "異常系: 期限切れトークンの場合、認証エラーが返る",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/na2na-platform"),
				repository:    "na2na-p/na2na-platform",
				ref:           "refs/heads/main",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ステップ1: 期限切れのJWTトークンを生成
			now := time.Now()
			expiredTime := now.Add(-1 * time.Hour) // 1時間前に期限切れ

			token, err := e2e.GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", tt.args.repository, tt.args.ref),
				"aud":        "cargohold",
				"repository": tt.args.repository,
				"ref":        tt.args.ref,
				"actor":      "github-actions[bot]",
				"iat":        now.Add(-2 * time.Hour).Unix(),
				"exp":        expiredTime.Unix(),
				"nbf":        now.Add(-2 * time.Hour).Unix(),
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			// ステップ2: Batch APIにBearerトークンを付けてリクエスト
			err = requestBatchAPIWithOIDC(
				tt.args.batchEndpoint,
				token,
				"upload",
				"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				123456,
			)

			// 検証: 認証エラーが返されることを確認
			if (err != nil) != tt.wantErr {
				t.Errorf("requestBatchAPIWithOIDC() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGitHubOIDCAuthentication_DownloadOperation はダウンロード操作でのOIDC認証をテストします
// ダウンロード操作をテストするには、まずオブジェクトをアップロードする必要があります
func TestGitHubOIDCAuthentication_DownloadOperation(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		fileSize       int64
		batchEndpoint  string
		verifyEndpoint string
		repository     string
		ref            string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "正常系: ダウンロード操作でのGitHub OIDC認証が成功する",
			args: args{
				fileSize:       1024,
				batchEndpoint:  e2e.GetBatchEndpoint("na2na-p/na2na-platform"),
				verifyEndpoint: e2e.GetVerifyEndpoint("na2na-p/na2na-platform"),
				repository:     "na2na-p/na2na-platform",
				ref:            "refs/heads/main",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ステップ1: テスト用ファイルを作成
			testFile, err := e2e.CreateTestFile(tt.args.fileSize)
			if err != nil {
				t.Fatalf("CreateTestFile() error = %v", err)
			}
			defer func() { _ = e2e.CleanupTestFiles(testFile) }()

			// ファイルのSHA256ハッシュとサイズを計算
			oid, size, err := e2e.CalculateFileHash(testFile)
			if err != nil {
				t.Fatalf("e2e.CalculateFileHash() error = %v", err)
			}

			// ステップ2: GitHub OIDC JWTトークンを生成
			token, err := e2e.GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", tt.args.repository, tt.args.ref),
				"aud":        "cargohold",
				"repository": tt.args.repository,
				"ref":        tt.args.ref,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			// ステップ3: まずオブジェクトをアップロード
			uploadURL, err := requestBatchAPIWithOIDCAndGetURL(
				tt.args.batchEndpoint,
				token,
				"upload",
				oid,
				size,
			)
			if err != nil {
				t.Fatalf("requestBatchAPIWithOIDCAndGetURL() for upload error = %v", err)
			}

			err = uploadFileToS3(uploadURL, testFile)
			if err != nil {
				t.Fatalf("uploadFileToS3() error = %v", err)
			}

			err = verifyUploadWithOIDC(tt.args.verifyEndpoint, token, oid, size)
			if err != nil {
				t.Fatalf("verifyUploadWithOIDC() error = %v", err)
			}

			// ステップ4: downloadオペレーションでBatch APIにリクエスト
			err = requestBatchAPIWithOIDC(
				tt.args.batchEndpoint,
				token,
				"download",
				oid,
				size,
			)

			// 検証: 認証成功を確認
			if (err != nil) != tt.wantErr {
				t.Errorf("requestBatchAPIWithOIDC() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// requestBatchAPIWithOIDC はOIDCトークンを使用してBatch APIを呼び出します
func requestBatchAPIWithOIDC(endpoint, token, operation, oid string, size int64) error {
	reqBody := map[string]interface{}{
		"operation": operation,
		"objects": []map[string]interface{}{
			{
				"oid":  oid,
				"size": size,
			},
		},
		"transfers": []string{"basic"},
		"hash_algo": "sha256",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("JSONのマーシャルに失敗しました: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	req.Header.Set("Content-Type", "application/vnd.git-lfs+json")

	// OIDCトークンをAuthorizationヘッダーに設定
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("リクエストの送信に失敗しました: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 401エラーの場合は認証失敗
	if resp.StatusCode == http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("認証に失敗しました: status=%d, body=%s", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Batch APIがエラーを返しました: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// TestGitHubOIDCAuthentication_WithUploadFlow はOIDC認証を使用した完全なアップロードフローをテストします
func TestGitHubOIDCAuthentication_WithUploadFlow(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		fileSize       int64
		batchEndpoint  string
		verifyEndpoint string
		repository     string
		ref            string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "正常系: OIDC認証を使用した完全なアップロードフローが成功する",
			args: args{
				fileSize:       1024 * 1024, // 1MB
				batchEndpoint:  e2e.GetBatchEndpoint("na2na-p/na2na-platform"),
				verifyEndpoint: e2e.GetVerifyEndpoint("na2na-p/na2na-platform"),
				repository:     "na2na-p/na2na-platform",
				ref:            "refs/heads/main",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 準備: テスト用ファイルを作成
			testFile, err := e2e.CreateTestFile(tt.args.fileSize)
			if err != nil {
				t.Fatalf("CreateTestFile() error = %v", err)
			}
			defer func() { _ = e2e.CleanupTestFiles(testFile) }()

			// ファイルのSHA256ハッシュとサイズを計算
			oid, size, err := e2e.CalculateFileHash(testFile)
			if err != nil {
				t.Fatalf("e2e.CalculateFileHash() error = %v", err)
			}

			// ステップ1: GitHub OIDC JWTトークンを生成
			token, err := e2e.GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", tt.args.repository, tt.args.ref),
				"aud":        "cargohold",
				"repository": tt.args.repository,
				"ref":        tt.args.ref,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			// ステップ2: OIDC認証でBatch APIを呼び出してアップロードURLを取得
			uploadURL, err := requestBatchAPIWithOIDCAndGetURL(
				tt.args.batchEndpoint,
				token,
				"upload",
				oid,
				size,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("requestBatchAPIWithOIDCAndGetURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// ステップ3: S3にファイルをアップロード
			err = uploadFileToS3(uploadURL, testFile)
			if err != nil {
				t.Errorf("uploadFileToS3() error = %v", err)
				return
			}

			// ステップ4: OIDC認証でVerify Endpointを呼び出す
			err = verifyUploadWithOIDC(tt.args.verifyEndpoint, token, oid, size)
			if err != nil {
				t.Errorf("verifyUploadWithOIDC() error = %v", err)
				return
			}
		})
	}
}

// requestBatchAPIWithOIDCAndGetURL はOIDC認証でBatch APIを呼び出してURLを取得します
func requestBatchAPIWithOIDCAndGetURL(endpoint, token, operation, oid string, size int64) (string, error) {
	reqBody := map[string]interface{}{
		"operation": operation,
		"objects": []map[string]interface{}{
			{
				"oid":  oid,
				"size": size,
			},
		},
		"transfers": []string{"basic"},
		"hash_algo": "sha256",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("JSONのマーシャルに失敗しました: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("リクエストの送信に失敗しました: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Batch APIがエラーを返しました: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var batchResp struct {
		Objects []struct {
			Actions *struct {
				Upload   *struct{ Href string } `json:"upload,omitempty"`
				Download *struct{ Href string } `json:"download,omitempty"`
			} `json:"actions,omitempty"`
		} `json:"objects"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return "", fmt.Errorf("レスポンスのパースに失敗しました: %w", err)
	}

	if len(batchResp.Objects) == 0 || batchResp.Objects[0].Actions == nil {
		return "", fmt.Errorf("アクションが含まれていません")
	}

	if operation == "upload" {
		if batchResp.Objects[0].Actions.Upload == nil {
			return "", fmt.Errorf("uploadアクションが含まれていません")
		}
		return batchResp.Objects[0].Actions.Upload.Href, nil
	}

	if batchResp.Objects[0].Actions.Download == nil {
		return "", fmt.Errorf("downloadアクションが含まれていません")
	}
	return batchResp.Objects[0].Actions.Download.Href, nil
}

// verifyUploadWithOIDC はOIDC認証でVerify Endpointを呼び出します
func verifyUploadWithOIDC(endpoint, token, oid string, size int64) error {
	reqBody := map[string]interface{}{
		"oid":  oid,
		"size": size,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("JSONのマーシャルに失敗しました: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("リクエストの送信に失敗しました: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Verifyがエラーを返しました: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// TestGitHubOIDCAuthentication_LFSAuthenticateHeader は401エラー時のLFS-Authenticateヘッダーを検証します
func TestGitHubOIDCAuthentication_LFSAuthenticateHeader(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		batchEndpoint string
		token         string
	}
	tests := []struct {
		name              string
		args              args
		wantStatusCode    int
		wantLFSAuthHeader string
	}{
		{
			name: "異常系: 無効なトークンの場合、LFS-Authenticateヘッダーが返される",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				token:         "invalid.token.here",
			},
			wantStatusCode:    http.StatusUnauthorized,
			wantLFSAuthHeader: `Basic realm="Git LFS"`,
		},
		{
			name: "異常系: 空のトークンの場合、LFS-Authenticateヘッダーが返される",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				token:         "",
			},
			wantStatusCode:    http.StatusUnauthorized,
			wantLFSAuthHeader: `Basic realm="Git LFS"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := requestBatchAPIWithOIDCAndResponse(
				tt.args.batchEndpoint,
				tt.args.token,
				"upload",
				"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				123456,
			)
			if err != nil {
				t.Fatalf("requestBatchAPIWithOIDCAndResponse() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode = %d, want %d", resp.StatusCode, tt.wantStatusCode)
			}

			lfsAuthHeader := resp.Header.Get("LFS-Authenticate")
			if diff := cmp.Diff(tt.wantLFSAuthHeader, lfsAuthHeader); diff != "" {
				t.Errorf("LFS-Authenticate ヘッダーが一致しません (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGitHubOIDCAuthentication_InvalidAudience は不正なaudienceでの認証失敗をテストします
func TestGitHubOIDCAuthentication_InvalidAudience(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		batchEndpoint string
		repository    string
		ref           string
		audience      string
	}
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
	}{
		{
			name: "異常系: audクレームが不正な場合、認証エラーが返る",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
				audience:      "invalid-audience",
			},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name: "異常系: audクレームが空の場合、認証エラーが返る",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
				audience:      "",
			},
			wantStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := e2e.GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", tt.args.repository, tt.args.ref),
				"aud":        tt.args.audience,
				"repository": tt.args.repository,
				"ref":        tt.args.ref,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			resp, err := requestBatchAPIWithOIDCAndResponse(
				tt.args.batchEndpoint,
				token,
				"upload",
				"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
				123456,
			)
			if err != nil {
				t.Fatalf("requestBatchAPIWithOIDCAndResponse() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.wantStatusCode {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("StatusCode = %d, want %d, body = %s", resp.StatusCode, tt.wantStatusCode, string(body))
			}
		})
	}
}

// TestGitHubOIDCAuthentication_InvalidIssuer は不正なissuerでの認証失敗をテストします
func TestGitHubOIDCAuthentication_InvalidIssuer(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		batchEndpoint string
		repository    string
		ref           string
		issuer        string
	}
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
	}{
		{
			name: "異常系: issクレームがGitHub OIDC issuer以外の場合、認証エラーが返る",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
				issuer:        "https://invalid-issuer.example.com",
			},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name: "異常系: issクレームが空の場合、認証エラーが返る",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
				issuer:        "",
			},
			wantStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := e2e.GenerateJWT(map[string]interface{}{
				"iss":        tt.args.issuer,
				"sub":        fmt.Sprintf("repo:%s:ref:%s", tt.args.repository, tt.args.ref),
				"aud":        "cargohold",
				"repository": tt.args.repository,
				"ref":        tt.args.ref,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			resp, err := requestBatchAPIWithOIDCAndResponse(
				tt.args.batchEndpoint,
				token,
				"upload",
				"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
				123456,
			)
			if err != nil {
				t.Fatalf("requestBatchAPIWithOIDCAndResponse() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.wantStatusCode {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("StatusCode = %d, want %d, body = %s", resp.StatusCode, tt.wantStatusCode, string(body))
			}
		})
	}
}

// TestGitHubOIDCAuthentication_UnauthorizedRepository は許可されていないリポジトリからのアクセスをテストします
func TestGitHubOIDCAuthentication_UnauthorizedRepository(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		batchEndpoint string
		repository    string
		ref           string
	}
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
	}{
		{
			name: "異常系: 許可リストに含まれないリポジトリからのアクセスは拒否される",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("unauthorized/repo"),
				repository:    "unauthorized/repo",
				ref:           "refs/heads/main",
			},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name: "異常系: 存在しないオーナーのリポジトリからのアクセスは拒否される",
			args: args{
				batchEndpoint: e2e.GetBatchEndpoint("unknown-owner/unknown-repo"),
				repository:    "unknown-owner/unknown-repo",
				ref:           "refs/heads/main",
			},
			wantStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := e2e.GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", tt.args.repository, tt.args.ref),
				"aud":        "cargohold",
				"repository": tt.args.repository,
				"ref":        tt.args.ref,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			resp, err := requestBatchAPIWithOIDCAndResponse(
				tt.args.batchEndpoint,
				token,
				"upload",
				"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
				123456,
			)
			if err != nil {
				t.Fatalf("requestBatchAPIWithOIDCAndResponse() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.wantStatusCode {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("StatusCode = %d, want %d, body = %s", resp.StatusCode, tt.wantStatusCode, string(body))
			}
		})
	}
}

// requestBatchAPIWithOIDCAndResponse はOIDCトークンを使用してBatch APIを呼び出し、レスポンスを返します
func requestBatchAPIWithOIDCAndResponse(endpoint, token, operation, oid string, size int64) (*http.Response, error) {
	reqBody := map[string]interface{}{
		"operation": operation,
		"objects": []map[string]interface{}{
			{
				"oid":  oid,
				"size": size,
			},
		},
		"transfers": []string{"basic"},
		"hash_algo": "sha256",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("JSONのマーシャルに失敗しました: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	req.Header.Set("Content-Type", "application/vnd.git-lfs+json")

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}
