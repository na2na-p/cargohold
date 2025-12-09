//go:build e2e

// Package e2e_test はE2Eテストを提供します
package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/e2e"
)

// TestUploadFlow はアップロードフローのE2Eテストを実施します
func TestUploadFlow(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		fileSize       int64
		uploadEndpoint string
		verifyEndpoint string
		batchEndpoint  string
		repository     string
		ref            string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "正常系: Upload成功時にオブジェクトが正しく保存される",
			args: args{
				fileSize:       1024 * 1024, // 1MB
				uploadEndpoint: e2e.GetBaseEndpoint(),
				verifyEndpoint: e2e.GetVerifyEndpoint("na2na-p/test-repo"),
				batchEndpoint:  e2e.GetBatchEndpoint("na2na-p/test-repo"),
				repository:     "na2na-p/test-repo",
				ref:            "refs/heads/main",
			},
			wantErr: false,
		},
		{
			name: "正常系: 大容量ファイル(10MB)のアップロードが成功する",
			args: args{
				fileSize:       10 * 1024 * 1024, // 10MB
				uploadEndpoint: e2e.GetBaseEndpoint(),
				verifyEndpoint: e2e.GetVerifyEndpoint("na2na-p/test-repo"),
				batchEndpoint:  e2e.GetBatchEndpoint("na2na-p/test-repo"),
				repository:     "na2na-p/test-repo",
				ref:            "refs/heads/main",
			},
			wantErr: false,
		},
		{
			name: "正常系: 小サイズファイル(1KB)のアップロードが成功する",
			args: args{
				fileSize:       1024, // 1KB
				uploadEndpoint: e2e.GetBaseEndpoint(),
				verifyEndpoint: e2e.GetVerifyEndpoint("na2na-p/test-repo"),
				batchEndpoint:  e2e.GetBatchEndpoint("na2na-p/test-repo"),
				repository:     "na2na-p/test-repo",
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

			// ステップ1: Batch APIを呼び出してアップロードURLを取得
			uploadURL, err := requestBatchAPI(
				tt.args.batchEndpoint,
				token,
				"upload",
				oid,
				size,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("requestBatchAPI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// ステップ2: S3にファイルをアップロード
			err = uploadFileToS3(uploadURL, testFile)
			if err != nil {
				t.Errorf("uploadFileToS3() error = %v", err)
				return
			}

			// ステップ3: Verify Endpointを呼び出してアップロードを検証
			err = verifyUpload(
				tt.args.verifyEndpoint,
				token,
				oid,
				size,
			)
			if err != nil {
				t.Errorf("verifyUpload() error = %v", err)
				return
			}

			// ステップ4: PostgreSQLのメタデータを確認（Batch APIを再度呼び出す）
			err = verifyMetadata(
				tt.args.batchEndpoint,
				token,
				oid,
				size,
			)
			if err != nil {
				t.Errorf("verifyMetadata() error = %v", err)
				return
			}
		})
	}
}

// requestBatchAPI はBatch APIを呼び出してアップロード/ダウンロードURLを取得します
func requestBatchAPI(endpoint, token, operation, oid string, size int64) (string, error) {
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
	// OIDCトークンをAuthorizationヘッダーに設定
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

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
		Transfer string `json:"transfer"`
		Objects  []struct {
			OID     string `json:"oid"`
			Size    int64  `json:"size"`
			Actions *struct {
				Upload   *struct{ Href string } `json:"upload,omitempty"`
				Download *struct{ Href string } `json:"download,omitempty"`
			} `json:"actions,omitempty"`
			Error *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error,omitempty"`
		} `json:"objects"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return "", fmt.Errorf("レスポンスのパースに失敗しました: %w", err)
	}

	if len(batchResp.Objects) == 0 {
		return "", fmt.Errorf("レスポンスにオブジェクトが含まれていません")
	}

	obj := batchResp.Objects[0]
	if obj.Error != nil {
		return "", fmt.Errorf("オブジェクトエラー: code=%d, message=%s", obj.Error.Code, obj.Error.Message)
	}

	if obj.Actions == nil {
		return "", fmt.Errorf("アクションが含まれていません")
	}

	if operation == "upload" {
		if obj.Actions.Upload == nil {
			return "", fmt.Errorf("uploadアクションが含まれていません")
		}
		return obj.Actions.Upload.Href, nil
	}

	if obj.Actions.Download == nil {
		return "", fmt.Errorf("downloadアクションが含まれていません")
	}
	return obj.Actions.Download.Href, nil
}

// uploadFileToS3 は署名付きURLを使用してS3にファイルをアップロードします
func uploadFileToS3(uploadURL, filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("ファイルを開けませんでした: %w", err)
	}
	defer func() { _ = file.Close() }()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("ファイル情報の取得に失敗しました: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, uploadURL, file)
	if err != nil {
		return fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = fileInfo.Size()

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("アップロードリクエストの送信に失敗しました: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("アップロードに失敗しました: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// verifyUpload はVerify Endpointを呼び出してアップロードを検証します
func verifyUpload(endpoint, token, oid string, size int64) error {
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Verifyがエラーを返しました: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// verifyMetadata はBatch APIを再度呼び出してメタデータを検証します
func verifyMetadata(endpoint, token, oid string, size int64) error {
	reqBody := map[string]interface{}{
		"operation": "download",
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Batch APIがエラーを返しました: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var batchResp struct {
		Objects []struct {
			OID   string `json:"oid"`
			Size  int64  `json:"size"`
			Error *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error,omitempty"`
		} `json:"objects"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return fmt.Errorf("レスポンスのパースに失敗しました: %w", err)
	}

	if len(batchResp.Objects) == 0 {
		return fmt.Errorf("レスポンスにオブジェクトが含まれていません")
	}

	obj := batchResp.Objects[0]
	if obj.Error != nil {
		return fmt.Errorf("オブジェクトエラー: code=%d, message=%s", obj.Error.Code, obj.Error.Message)
	}

	// OIDとサイズの確認
	if diff := cmp.Diff(oid, obj.OID); diff != "" {
		return fmt.Errorf("OIDが一致しません (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(size, obj.Size); diff != "" {
		return fmt.Errorf("Sizeが一致しません (-want +got):\n%s", diff)
	}

	return nil
}
