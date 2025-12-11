//go:build e2e

// Package e2e はE2Eテストを提供します
package e2e

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
)

// TestUploadFlow はアップロードフローのE2Eテストを実施します
func TestUploadFlow(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
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
				uploadEndpoint: GetBaseEndpoint(),
				verifyEndpoint: GetVerifyEndpoint("na2na-p/test-repo"),
				batchEndpoint:  GetBatchEndpoint("na2na-p/test-repo"),
				repository:     "na2na-p/test-repo",
				ref:            "refs/heads/main",
			},
			wantErr: false,
		},
		{
			name: "正常系: 大容量ファイル(10MB)のアップロードが成功する",
			args: args{
				fileSize:       10 * 1024 * 1024, // 10MB
				uploadEndpoint: GetBaseEndpoint(),
				verifyEndpoint: GetVerifyEndpoint("na2na-p/test-repo"),
				batchEndpoint:  GetBatchEndpoint("na2na-p/test-repo"),
				repository:     "na2na-p/test-repo",
				ref:            "refs/heads/main",
			},
			wantErr: false,
		},
		{
			name: "正常系: 小サイズファイル(1KB)のアップロードが成功する",
			args: args{
				fileSize:       1024, // 1KB
				uploadEndpoint: GetBaseEndpoint(),
				verifyEndpoint: GetVerifyEndpoint("na2na-p/test-repo"),
				batchEndpoint:  GetBatchEndpoint("na2na-p/test-repo"),
				repository:     "na2na-p/test-repo",
				ref:            "refs/heads/main",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 準備: テスト用ファイルを作成
			testFile, err := CreateTestFile(tt.args.fileSize)
			if err != nil {
				t.Fatalf("CreateTestFile() error = %v", err)
			}
			defer func() { _ = CleanupTestFiles(testFile) }()

			// ファイルのSHA256ハッシュとサイズを計算
			oid, size, err := CalculateFileHash(testFile)
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

			// ステップ2: Proxyエンドポイントにファイルをアップロード
			err = uploadFileToProxy(uploadURL, testFile, token)
			if err != nil {
				t.Errorf("uploadFileToProxy() error = %v", err)
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

// uploadFileToProxy はProxyエンドポイントを使用してファイルをアップロードします
func uploadFileToProxy(uploadURL, filepath, token string) error {
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

	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	req.Header.Set("Content-Type", "application/octet-stream")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
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

// TestUploadProxy_Unauthorized は認証なしでProxyエンドポイントにアクセスした場合のテストを実施します
func TestUploadProxy_Unauthorized(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	tests := []struct {
		name       string
		oid        string
		wantStatus int
	}{
		{
			name:       "異常系: 認証なしでアップロードした場合、401エラーが返る",
			oid:        "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL := GetBaseEndpoint()
			proxyURL := fmt.Sprintf("%s/na2na-p/test-repo/info/lfs/objects/%s", baseURL, tt.oid)

			req, err := http.NewRequest(http.MethodPut, proxyURL, bytes.NewBuffer([]byte("test data")))
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}
			req.Header.Set("Content-Type", "application/octet-stream")

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

// TestUploadProxy_NotFound は存在しないOIDに対してアップロードした場合のテストを実施します
func TestUploadProxy_NotFound(t *testing.T) {
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
			name:       "異常系: Batch APIで登録されていないOIDにアップロードした場合、404エラーが返る",
			oid:        "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			repository: "na2na-p/test-repo",
			ref:        "refs/heads/main",
			wantStatus: http.StatusNotFound,
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

			req, err := http.NewRequest(http.MethodPut, proxyURL, bytes.NewBuffer([]byte("test data")))
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}
			req.Header.Set("Content-Type", "application/octet-stream")
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

// TestUploadProxy_CrossRepositoryAccessDenied は別のリポジトリで作成されたOIDへのアップロードが拒否されることを検証します
func TestUploadProxy_CrossRepositoryAccessDenied(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		fileSize           int64
		registerRepository string
		registerRef        string
		accessRepository   string
		accessRef          string
	}
	tests := []struct {
		name       string
		args       args
		wantStatus int
	}{
		{
			name: "異常系: 別リポジトリのトークンでアップロードした場合、403エラーが返る",
			args: args{
				fileSize:           1024,
				registerRepository: "na2na-p/test-repo",
				registerRef:        "refs/heads/main",
				accessRepository:   "na2na-p/na2na-platform",
				accessRef:          "refs/heads/main",
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

			registerToken, err := GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", tt.args.registerRepository, tt.args.registerRef),
				"aud":        "cargohold",
				"repository": tt.args.registerRepository,
				"ref":        tt.args.registerRef,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			_, err = requestBatchAPI(
				GetBatchEndpoint(tt.args.registerRepository),
				registerToken,
				"upload",
				oid,
				size,
			)
			if err != nil {
				t.Fatalf("requestBatchAPI() error = %v", err)
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

			file, err := os.Open(testFile)
			if err != nil {
				t.Fatalf("os.Open() error = %v", err)
			}
			defer func() { _ = file.Close() }()

			req, err := http.NewRequest(http.MethodPut, proxyURL, file)
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}
			req.Header.Set("Content-Type", "application/octet-stream")
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
