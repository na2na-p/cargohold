//go:build e2e

// Package e2e はE2Eテストを提供します
package e2e

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// TestDownloadFlow はダウンロードフローのE2Eテストを実施します
func TestDownloadFlow(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		fileSize      int64
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
			name: "正常系: ダウンロード成功時にオブジェクトが正しく取得される",
			args: args{
				fileSize:      1024 * 1024, // 1MB
				batchEndpoint: GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
			},
			wantErr: false,
		},
		{
			name: "正常系: 大容量ファイル(10MB)のダウンロードが成功する",
			args: args{
				fileSize:      10 * 1024 * 1024, // 10MB
				batchEndpoint: GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
			},
			wantErr: false,
		},
		{
			name: "正常系: 小サイズファイル(1KB)のダウンロードが成功する",
			args: args{
				fileSize:      1024, // 1KB
				batchEndpoint: GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 準備: テスト用ファイルを作成
			originalFile, err := CreateTestFile(tt.args.fileSize)
			if err != nil {
				t.Fatalf("CreateTestFile() error = %v", err)
			}
			defer func() { _ = CleanupTestFiles(originalFile) }()

			// オリジナルファイルのSHA256ハッシュとサイズを計算
			originalOID, originalSize, err := CalculateFileHash(originalFile)
			if err != nil {
				t.Fatalf("CalculateFileHash() error = %v", err)
			}

			// GitHub OIDC JWTトークンを生成
			token, err := GenerateJWT(map[string]interface{}{
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

			// 前提: アップロードフローを実行してファイルをアップロード
			err = setupUploadedFile(
				tt.args.batchEndpoint,
				token,
				originalFile,
				originalOID,
				originalSize,
			)
			if err != nil {
				t.Fatalf("setupUploadedFile() error = %v", err)
			}

			// ステップ1: Batch APIを呼び出してダウンロードURLを取得
			downloadURL, err := requestBatchAPI(
				tt.args.batchEndpoint,
				token,
				"download",
				originalOID,
				originalSize,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("requestBatchAPI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// ステップ2: Proxyエンドポイントからファイルをダウンロード
			downloadedFile, err := downloadFileFromProxy(downloadURL, token)
			if err != nil {
				t.Errorf("downloadFileFromProxy() error = %v", err)
				return
			}
			defer func() { _ = CleanupTestFiles(downloadedFile) }()

			// ステップ3: ダウンロードしたファイルのハッシュを検証
			downloadedOID, downloadedSize, err := CalculateFileHash(downloadedFile)
			if err != nil {
				t.Errorf("CalculateFileHash() error = %v", err)
				return
			}

			// ステップ4: オリジナルとダウンロード後のハッシュ・サイズを比較
			if diff := cmp.Diff(originalOID, downloadedOID); diff != "" {
				t.Errorf("OIDが一致しません (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(originalSize, downloadedSize); diff != "" {
				t.Errorf("Sizeが一致しません (-want +got):\n%s", diff)
			}
		})
	}
}

// setupUploadedFile はテスト用にファイルをアップロードします
func setupUploadedFile(batchEndpoint, token, filepath, oid string, size int64) error {
	uploadURL, err := requestBatchAPI(batchEndpoint, token, "upload", oid, size)
	if err != nil {
		return fmt.Errorf("requestBatchAPI() error = %w", err)
	}

	err = uploadFileToProxy(uploadURL, filepath, token)
	if err != nil {
		return fmt.Errorf("uploadFileToProxy() error = %w", err)
	}

	verifyEndpoint, err := buildVerifyEndpoint(batchEndpoint)
	if err != nil {
		return fmt.Errorf("buildVerifyEndpoint() error = %w", err)
	}

	err = verifyUpload(verifyEndpoint, token, oid, size)
	if err != nil {
		return fmt.Errorf("verifyUpload() error = %w", err)
	}

	return nil
}

// buildVerifyEndpoint はbatchEndpointからverifyEndpointを構築します
func buildVerifyEndpoint(batchEndpoint string) (string, error) {
	parsedURL, err := url.Parse(batchEndpoint)
	if err != nil {
		return "", fmt.Errorf("URLのパースに失敗しました: %w", err)
	}

	dir := path.Dir(parsedURL.Path)
	parsedURL.Path = path.Join(dir, "verify")

	return parsedURL.String(), nil
}

// downloadFileFromProxy はProxyエンドポイントを使用してファイルをダウンロードします
func downloadFileFromProxy(downloadURL, token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ダウンロードリクエストの送信に失敗しました: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ダウンロードに失敗しました: status=%d, body=%s", resp.StatusCode, string(body))
	}

	tempDir, err := GetTestTempDir()
	if err != nil {
		return "", fmt.Errorf("GetTestTempDir() error = %w", err)
	}

	filename := fmt.Sprintf("downloaded-%d.bin", time.Now().UnixNano())
	filepath := fmt.Sprintf("%s/%s", tempDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("ファイルの作成に失敗しました: %w", err)
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("ファイルへの書き込みに失敗しました: %w", err)
	}

	return filepath, nil
}

// TestDownloadProxy_Unauthorized は認証なしでProxyエンドポイントにアクセスした場合のテストを実施します
func TestDownloadProxy_Unauthorized(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	tests := []struct {
		name       string
		oid        string
		wantStatus int
	}{
		{
			name:       "異常系: 認証なしでダウンロードした場合、401エラーが返る",
			oid:        "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL := GetBaseEndpoint()
			proxyURL := fmt.Sprintf("%s/na2na-p/test-repo/info/lfs/objects/%s", baseURL, tt.oid)

			req, err := http.NewRequest(http.MethodGet, proxyURL, nil)
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}
			req.Header.Set("Accept", "application/octet-stream")

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("client.Do() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if diff := cmp.Diff(tt.wantStatus, resp.StatusCode); diff != "" {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("ステータスコードが期待値と異なります (-want +got):\n%s\nbody: %s", diff, string(body))
			}
		})
	}
}

// TestDownloadProxy_NonExistentOID は存在しないOIDに対してダウンロードした場合のテストを実施します
// アクセスポリシーが存在しないため、403 Forbiddenが返されます
func TestDownloadProxy_NonExistentOID(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	tests := []struct {
		name       string
		oid        string
		repository string
		ref        string
		wantStatus int
	}{
		{
			name:       "異常系: 存在しないOIDに対してダウンロードした場合、403エラーが返る（アクセスポリシーが存在しないため）",
			oid:        "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			repository: "na2na-p/test-repo",
			ref:        "refs/heads/main",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", tt.repository, tt.ref),
				"aud":        "cargohold",
				"repository": tt.repository,
				"ref":        tt.ref,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			baseURL := GetBaseEndpoint()
			proxyURL := fmt.Sprintf("%s/%s/info/lfs/objects/%s", baseURL, tt.repository, tt.oid)

			req, err := http.NewRequest(http.MethodGet, proxyURL, nil)
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}
			req.Header.Set("Accept", "application/octet-stream")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("client.Do() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if diff := cmp.Diff(tt.wantStatus, resp.StatusCode); diff != "" {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("ステータスコードが期待値と異なります (-want +got):\n%s\nbody: %s", diff, string(body))
			}
		})
	}
}

// TestDownloadFlow_FileNotFound は存在しないファイルのダウンロード時のエラーをテストします
func TestDownloadFlow_FileNotFound(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		batchEndpoint string
		oid           string
		size          int64
		repository    string
		ref           string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "異常系: 存在しないOIDの場合、エラーが返る",
			args: args{
				batchEndpoint: GetBatchEndpoint("na2na-p/test-repo"),
				oid:           "0000000000000000000000000000000000000000000000000000000000000000",
				size:          1024,
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GitHub OIDC JWTトークンを生成
			token, err := GenerateJWT(map[string]interface{}{
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

			// ステップ1: Batch APIを呼び出す（存在しないOID）
			_, err = requestBatchAPI(
				tt.args.batchEndpoint,
				token,
				"download",
				tt.args.oid,
				tt.args.size,
			)

			// 検証: エラーが返されることを確認
			if (err != nil) != tt.wantErr {
				t.Errorf("requestBatchAPI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDownloadProxy_CrossRepositoryAccessDenied は別のリポジトリのオブジェクトへのアクセスが拒否されることを検証します
func TestDownloadProxy_CrossRepositoryAccessDenied(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		fileSize         int64
		uploadRepository string
		uploadRef        string
		accessRepository string
		accessRef        string
	}
	tests := []struct {
		name       string
		args       args
		wantStatus int
	}{
		{
			name: "異常系: 別リポジトリのトークンでダウンロードした場合、403エラーが返る",
			args: args{
				fileSize:         1024,
				uploadRepository: "na2na-p/test-repo",
				uploadRef:        "refs/heads/main",
				accessRepository: "na2na-p/na2na-platform",
				accessRef:        "refs/heads/main",
			},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile, err := CreateTestFile(tt.args.fileSize)
			if err != nil {
				t.Fatalf("CreateTestFile() error = %v", err)
			}
			defer func() { _ = CleanupTestFiles(testFile) }()

			oid, size, err := CalculateFileHash(testFile)
			if err != nil {
				t.Fatalf("CalculateFileHash() error = %v", err)
			}

			uploadToken, err := GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", tt.args.uploadRepository, tt.args.uploadRef),
				"aud":        "cargohold",
				"repository": tt.args.uploadRepository,
				"ref":        tt.args.uploadRef,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			err = setupUploadedFile(
				GetBatchEndpoint(tt.args.uploadRepository),
				uploadToken,
				testFile,
				oid,
				size,
			)
			if err != nil {
				t.Fatalf("setupUploadedFile() error = %v", err)
			}

			accessToken, err := GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", tt.args.accessRepository, tt.args.accessRef),
				"aud":        "cargohold",
				"repository": tt.args.accessRepository,
				"ref":        tt.args.accessRef,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			baseURL := GetBaseEndpoint()
			proxyURL := fmt.Sprintf("%s/%s/info/lfs/objects/%s", baseURL, tt.args.accessRepository, oid)

			req, err := http.NewRequest(http.MethodGet, proxyURL, nil)
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}
			req.Header.Set("Accept", "application/octet-stream")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("client.Do() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if diff := cmp.Diff(tt.wantStatus, resp.StatusCode); diff != "" {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("ステータスコードが期待値と異なります (-want +got):\n%s\nbody: %s", diff, string(body))
			}
		})
	}
}

// TestDownloadFlow_ContentVerification はダウンロードしたファイルの内容を検証します
func TestDownloadFlow_ContentVerification(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		fileSize      int64
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
			name: "正常系: ダウンロードしたファイルの内容が元のファイルと完全に一致する",
			args: args{
				fileSize:      1024 * 1024, // 1MB
				batchEndpoint: GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 準備: テスト用ファイルを作成
			originalFile, err := CreateTestFile(tt.args.fileSize)
			if err != nil {
				t.Fatalf("CreateTestFile() error = %v", err)
			}
			defer func() { _ = CleanupTestFiles(originalFile) }()

			// オリジナルファイルの内容を読み込み
			originalContent, err := os.ReadFile(originalFile)
			if err != nil {
				t.Fatalf("os.ReadFile() error = %v", err)
			}

			// ハッシュ計算
			originalOID, originalSize, err := CalculateFileHash(originalFile)
			if err != nil {
				t.Fatalf("CalculateFileHash() error = %v", err)
			}

			// GitHub OIDC JWTトークンを生成
			token, err := GenerateJWT(map[string]interface{}{
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

			// アップロード
			err = setupUploadedFile(
				tt.args.batchEndpoint,
				token,
				originalFile,
				originalOID,
				originalSize,
			)
			if err != nil {
				t.Fatalf("setupUploadedFile() error = %v", err)
			}

			// ダウンロード
			downloadURL, err := requestBatchAPI(
				tt.args.batchEndpoint,
				token,
				"download",
				originalOID,
				originalSize,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("requestBatchAPI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			downloadedFile, err := downloadFileFromProxy(downloadURL, token)
			if err != nil {
				t.Errorf("downloadFileFromProxy() error = %v", err)
				return
			}
			defer func() { _ = CleanupTestFiles(downloadedFile) }()

			// ダウンロードしたファイルの内容を読み込み
			downloadedContent, err := os.ReadFile(downloadedFile)
			if err != nil {
				t.Errorf("os.ReadFile() error = %v", err)
				return
			}

			// バイト単位で内容を比較
			if diff := cmp.Diff(originalContent, downloadedContent); diff != "" {
				t.Errorf("ファイルの内容が一致しません (-want +got):\n%s", diff)
			}
		})
	}
}
