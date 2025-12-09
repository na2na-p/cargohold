//go:build e2e

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

func TestVerifyAPI_BadRequest(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		reqBody    map[string]interface{}
		repository string
		ref        string
	}
	tests := []struct {
		name        string
		args        args
		wantStatus  int
		wantMessage string
	}{
		{
			name: "異常系: OIDが未指定の場合、400エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"size": 1024,
				},
				repository: "na2na-p/test-repo",
				ref:        "refs/heads/main",
			},
			wantStatus:  http.StatusBadRequest,
			wantMessage: "oidフィールドは必須です",
		},
		{
			name: "異常系: sizeが未指定の場合、400エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"oid": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				},
				repository: "na2na-p/test-repo",
				ref:        "refs/heads/main",
			},
			wantStatus:  http.StatusBadRequest,
			wantMessage: "sizeフィールドは正の整数である必要があります",
		},
		{
			name: "異常系: sizeが0の場合、400エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"oid":  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
					"size": 0,
				},
				repository: "na2na-p/test-repo",
				ref:        "refs/heads/main",
			},
			wantStatus:  http.StatusBadRequest,
			wantMessage: "sizeフィールドは正の整数である必要があります",
		},
		{
			name: "異常系: sizeが負の値の場合、400エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"oid":  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
					"size": -1,
				},
				repository: "na2na-p/test-repo",
				ref:        "refs/heads/main",
			},
			wantStatus:  http.StatusBadRequest,
			wantMessage: "sizeフィールドは正の整数である必要があります",
		},
		{
			name: "異常系: OIDが64文字未満の場合、400エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"oid":  "a1b2c3d4e5f6",
					"size": 1024,
				},
				repository: "na2na-p/test-repo",
				ref:        "refs/heads/main",
			},
			wantStatus:  http.StatusBadRequest,
			wantMessage: "oidフィールドは必須です",
		},
		{
			name: "異常系: OIDが64文字超過の場合、400エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"oid":  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2aaaa",
					"size": 1024,
				},
				repository: "na2na-p/test-repo",
				ref:        "refs/heads/main",
			},
			wantStatus:  http.StatusBadRequest,
			wantMessage: "oidフィールドは必須です",
		},
		{
			name: "異常系: OIDに16進数以外の文字が含まれる場合、400エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"oid":  "g1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
					"size": 1024,
				},
				repository: "na2na-p/test-repo",
				ref:        "refs/heads/main",
			},
			wantStatus:  http.StatusBadRequest,
			wantMessage: "oidフィールドは必須です",
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

			endpoint := e2e.GetVerifyEndpoint(tt.args.repository)
			status, message, err := sendVerifyRequest(endpoint, token, tt.args.reqBody)
			if err != nil {
				t.Fatalf("sendVerifyRequest() error = %v", err)
			}

			if diff := cmp.Diff(tt.wantStatus, status); diff != "" {
				t.Errorf("ステータスコードが一致しません (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantMessage, message); diff != "" {
				t.Errorf("エラーメッセージが一致しません (-want +got):\n%s", diff)
			}
		})
	}
}

func TestVerifyAPI_NotFound(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		reqBody    map[string]interface{}
		repository string
		ref        string
	}
	tests := []struct {
		name        string
		args        args
		wantStatus  int
		wantMessage string
	}{
		{
			name: "異常系: 存在しないOIDを指定した場合、404エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"oid":  "0000000000000000000000000000000000000000000000000000000000000000",
					"size": 1024,
				},
				repository: "na2na-p/test-repo",
				ref:        "refs/heads/main",
			},
			wantStatus:  http.StatusNotFound,
			wantMessage: "オブジェクトが見つかりません",
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

			endpoint := e2e.GetVerifyEndpoint(tt.args.repository)
			status, message, err := sendVerifyRequest(endpoint, token, tt.args.reqBody)
			if err != nil {
				t.Fatalf("sendVerifyRequest() error = %v", err)
			}

			if diff := cmp.Diff(tt.wantStatus, status); diff != "" {
				t.Errorf("ステータスコードが一致しません (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantMessage, message); diff != "" {
				t.Errorf("エラーメッセージが一致しません (-want +got):\n%s", diff)
			}
		})
	}
}

func TestVerifyAPI_UnprocessableEntity(t *testing.T) {
	if err := e2e.SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		fileSize       int64
		verifyWithSize int64
		repository     string
		ref            string
	}
	tests := []struct {
		name        string
		args        args
		wantStatus  int
		wantMessage string
	}{
		{
			name: "異常系: アップロード時と異なるサイズを指定した場合、422エラーが返る",
			args: args{
				fileSize:       1024,
				verifyWithSize: 2048,
				repository:     "na2na-p/test-repo",
				ref:            "refs/heads/main",
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantMessage: "サイズが一致しません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile, err := e2e.CreateTestFile(tt.args.fileSize)
			if err != nil {
				t.Fatalf("CreateTestFile() error = %v", err)
			}
			defer func() { _ = e2e.CleanupTestFiles(testFile) }()

			oid, size, err := e2e.CalculateFileHash(testFile)
			if err != nil {
				t.Fatalf("CalculateFileHash() error = %v", err)
			}

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

			batchEndpoint := e2e.GetBatchEndpoint(tt.args.repository)
			uploadURL, err := requestBatchAPIForVerifyTest(batchEndpoint, token, "upload", oid, size)
			if err != nil {
				t.Fatalf("requestBatchAPIForVerifyTest() error = %v", err)
			}

			err = uploadFileToS3ForVerifyTest(uploadURL, testFile)
			if err != nil {
				t.Fatalf("uploadFileToS3ForVerifyTest() error = %v", err)
			}

			verifyEndpoint := e2e.GetVerifyEndpoint(tt.args.repository)
			reqBody := map[string]interface{}{
				"oid":  oid,
				"size": tt.args.verifyWithSize,
			}

			status, message, err := sendVerifyRequest(verifyEndpoint, token, reqBody)
			if err != nil {
				t.Fatalf("sendVerifyRequest() error = %v", err)
			}

			if diff := cmp.Diff(tt.wantStatus, status); diff != "" {
				t.Errorf("ステータスコードが一致しません (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantMessage, message); diff != "" {
				t.Errorf("エラーメッセージが一致しません (-want +got):\n%s", diff)
			}
		})
	}
}

func sendVerifyRequest(endpoint, token string, reqBody map[string]interface{}) (int, string, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return 0, "", fmt.Errorf("JSONのマーシャルに失敗しました: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, "", fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("リクエストの送信に失敗しました: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("レスポンスボディの読み込みに失敗しました: %w", err)
	}

	var respBody struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &respBody); err != nil {
		return resp.StatusCode, "", fmt.Errorf("レスポンスのパースに失敗しました: %w", err)
	}

	return resp.StatusCode, respBody.Message, nil
}

func requestBatchAPIForVerifyTest(endpoint, token, operation, oid string, size int64) (string, error) {
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
		Objects []struct {
			OID     string `json:"oid"`
			Size    int64  `json:"size"`
			Actions *struct {
				Upload *struct {
					Href string `json:"href"`
				} `json:"upload,omitempty"`
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

	if obj.Actions == nil || obj.Actions.Upload == nil {
		return "", fmt.Errorf("uploadアクションが含まれていません")
	}

	return obj.Actions.Upload.Href, nil
}

func uploadFileToS3ForVerifyTest(uploadURL, filepath string) error {
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
