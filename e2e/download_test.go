//go:build e2e

// Package e2e_test はE2Eテストを提供します
package e2e_test

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
	"github.com/na2na-p/cargohold/e2e"
)

// TestDownloadFlow はダウンロードフローのE2Eテストを実施します
func TestDownloadFlow(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
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
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
			},
			wantErr: false,
		},
		{
			name: "正常系: 大容量ファイル(10MB)のダウンロードが成功する",
			args: args{
				fileSize:      10 * 1024 * 1024, // 10MB
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
			},
			wantErr: false,
		},
		{
			name: "正常系: 小サイズファイル(1KB)のダウンロードが成功する",
			args: args{
				fileSize:      1024, // 1KB
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 準備: テスト用ファイルを作成
			originalFile, err := e2e.CreateTestFile(tt.args.fileSize)
			if err != nil {
				t.Fatalf("CreateTestFile() error = %v", err)
			}
			defer func() { _ = e2e.CleanupTestFiles(originalFile) }()

			// オリジナルファイルのSHA256ハッシュとサイズを計算
			originalOID, originalSize, err := e2e.CalculateFileHash(originalFile)
			if err != nil {
				t.Fatalf("e2e.CalculateFileHash() error = %v", err)
			}

			// GitHub OIDC JWTトークンを生成
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

			// ステップ2: S3からファイルをダウンロード
			downloadedFile, err := downloadFileFromS3(downloadURL)
			if err != nil {
				t.Errorf("downloadFileFromS3() error = %v", err)
				return
			}
			defer func() { _ = e2e.CleanupTestFiles(downloadedFile) }()

			// ステップ3: ダウンロードしたファイルのハッシュを検証
			downloadedOID, downloadedSize, err := e2e.CalculateFileHash(downloadedFile)
			if err != nil {
				t.Errorf("e2e.CalculateFileHash() error = %v", err)
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
	// Batch APIを呼び出してアップロードURLを取得
	uploadURL, err := requestBatchAPI(batchEndpoint, token, "upload", oid, size)
	if err != nil {
		return fmt.Errorf("requestBatchAPI() error = %w", err)
	}

	// S3にファイルをアップロード
	err = uploadFileToS3(uploadURL, filepath)
	if err != nil {
		return fmt.Errorf("uploadFileToS3() error = %w", err)
	}

	// Verify EndpointのURLを構築
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

// downloadFileFromS3 は署名付きURLを使用してS3からファイルをダウンロードします
func downloadFileFromS3(downloadURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("リクエストの作成に失敗しました: %w", err)
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

	// 一時ファイルにダウンロード
	tempDir, err := e2e.GetTestTempDir()
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

// TestDownloadFlow_FileNotFound は存在しないファイルのダウンロード時のエラーをテストします
func TestDownloadFlow_FileNotFound(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
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
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
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

// TestDownloadFlow_ContentVerification はダウンロードしたファイルの内容を検証します
func TestDownloadFlow_ContentVerification(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
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
				batchEndpoint: e2e.GetBatchEndpoint("na2na-p/test-repo"),
				repository:    "na2na-p/test-repo",
				ref:           "refs/heads/main",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 準備: テスト用ファイルを作成
			originalFile, err := e2e.CreateTestFile(tt.args.fileSize)
			if err != nil {
				t.Fatalf("CreateTestFile() error = %v", err)
			}
			defer func() { _ = e2e.CleanupTestFiles(originalFile) }()

			// オリジナルファイルの内容を読み込み
			originalContent, err := os.ReadFile(originalFile)
			if err != nil {
				t.Fatalf("os.ReadFile() error = %v", err)
			}

			// ハッシュ計算
			originalOID, originalSize, err := e2e.CalculateFileHash(originalFile)
			if err != nil {
				t.Fatalf("e2e.CalculateFileHash() error = %v", err)
			}

			// GitHub OIDC JWTトークンを生成
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

			downloadedFile, err := downloadFileFromS3(downloadURL)
			if err != nil {
				t.Errorf("downloadFileFromS3() error = %v", err)
				return
			}
			defer func() { _ = e2e.CleanupTestFiles(downloadedFile) }()

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
