//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type BatchResponse struct {
	Transfer string        `json:"transfer"`
	HashAlgo string        `json:"hash_algo"`
	Objects  []BatchObject `json:"objects"`
}

type BatchObject struct {
	OID           string       `json:"oid"`
	Size          int64        `json:"size"`
	Authenticated bool         `json:"authenticated"`
	Actions       *Actions     `json:"actions,omitempty"`
	Error         *ObjectError `json:"error,omitempty"`
}

type Actions struct {
	Upload   *Action `json:"upload,omitempty"`
	Download *Action `json:"download,omitempty"`
}

type Action struct {
	Href      string `json:"href"`
	ExpiresIn int    `json:"expires_in"`
}

type ObjectError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func TestBatchAPI_ResponseFieldValidation(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	validOID := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	type args struct {
		operation string
		oid       string
		size      int64
	}
	type want struct {
		transfer      string
		hashAlgo      string
		authenticated bool
		hasExpiresIn  bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "正常系: upload時にtransferフィールドがbasicであること",
			args: args{
				operation: "upload",
				oid:       validOID,
				size:      1024,
			},
			want: want{
				transfer:      "basic",
				hashAlgo:      "sha256",
				authenticated: true,
				hasExpiresIn:  true,
			},
		},
		{
			name: "正常系: upload時にhash_algoフィールドがsha256であること",
			args: args{
				operation: "upload",
				oid:       validOID,
				size:      2048,
			},
			want: want{
				transfer:      "basic",
				hashAlgo:      "sha256",
				authenticated: true,
				hasExpiresIn:  true,
			},
		},
		{
			name: "正常系: upload時にauthenticatedフィールドがtrueであること",
			args: args{
				operation: "upload",
				oid:       validOID,
				size:      4096,
			},
			want: want{
				transfer:      "basic",
				hashAlgo:      "sha256",
				authenticated: true,
				hasExpiresIn:  true,
			},
		},
		{
			name: "正常系: upload時にexpires_inフィールドが含まれること",
			args: args{
				operation: "upload",
				oid:       validOID,
				size:      8192,
			},
			want: want{
				transfer:      "basic",
				hashAlgo:      "sha256",
				authenticated: true,
				hasExpiresIn:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := "na2na-p/test-repo"
			ref := "refs/heads/main"
			endpoint := GetBatchEndpoint(repository)

			token, err := GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", repository, ref),
				"aud":        "cargohold",
				"repository": repository,
				"ref":        ref,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			reqBody := map[string]interface{}{
				"operation": tt.args.operation,
				"objects": []map[string]interface{}{
					{
						"oid":  tt.args.oid,
						"size": tt.args.size,
					},
				},
				"transfers": []string{"basic"},
				"hash_algo": "sha256",
			}

			jsonData, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}

			req.Header.Set("Accept", "application/vnd.git-lfs+json")
			req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("client.Do() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("Batch APIがエラーを返しました: status=%d, body=%s", resp.StatusCode, string(body))
			}

			var batchResp BatchResponse
			if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
				t.Fatalf("json.Decode() error = %v", err)
			}

			if diff := cmp.Diff(tt.want.transfer, batchResp.Transfer); diff != "" {
				t.Errorf("transferフィールドが期待値と異なります (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.want.hashAlgo, batchResp.HashAlgo); diff != "" {
				t.Errorf("hash_algoフィールドが期待値と異なります (-want +got):\n%s", diff)
			}

			if len(batchResp.Objects) == 0 {
				t.Fatal("レスポンスにオブジェクトが含まれていません")
			}

			obj := batchResp.Objects[0]

			if diff := cmp.Diff(tt.want.authenticated, obj.Authenticated); diff != "" {
				t.Errorf("authenticatedフィールドが期待値と異なります (-want +got):\n%s", diff)
			}

			if obj.Actions == nil {
				t.Fatal("actionsフィールドが含まれていません")
			}

			if tt.args.operation == "upload" {
				if obj.Actions.Upload == nil {
					t.Fatal("uploadアクションが含まれていません")
				}
				if tt.want.hasExpiresIn && obj.Actions.Upload.ExpiresIn <= 0 {
					t.Errorf("expires_inフィールドが正の値ではありません: got=%d", obj.Actions.Upload.ExpiresIn)
				}
			}
		})
	}
}

func TestBatchAPI_MultipleObjects(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type objectSpec struct {
		oid  string
		size int64
	}
	type args struct {
		operation string
		objects   []objectSpec
	}
	type want struct {
		statusCode       int
		objectCount      int
		allAuthenticated bool
		hasAllTransfer   bool
		hasAllHashAlgo   bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "正常系: 複数オブジェクト同時アップロード",
			args: args{
				operation: "upload",
				objects: []objectSpec{
					{
						oid:  "1111111111111111111111111111111111111111111111111111111111111111",
						size: 1024,
					},
					{
						oid:  "2222222222222222222222222222222222222222222222222222222222222222",
						size: 2048,
					},
					{
						oid:  "3333333333333333333333333333333333333333333333333333333333333333",
						size: 4096,
					},
				},
			},
			want: want{
				statusCode:       http.StatusOK,
				objectCount:      3,
				allAuthenticated: true,
				hasAllTransfer:   true,
				hasAllHashAlgo:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := "na2na-p/test-repo"
			ref := "refs/heads/main"
			endpoint := GetBatchEndpoint(repository)

			token, err := GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", repository, ref),
				"aud":        "cargohold",
				"repository": repository,
				"ref":        ref,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			objects := make([]map[string]interface{}, len(tt.args.objects))
			for i, obj := range tt.args.objects {
				objects[i] = map[string]interface{}{
					"oid":  obj.oid,
					"size": obj.size,
				}
			}

			reqBody := map[string]interface{}{
				"operation": tt.args.operation,
				"objects":   objects,
				"transfers": []string{"basic"},
				"hash_algo": "sha256",
			}

			jsonData, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}

			req.Header.Set("Accept", "application/vnd.git-lfs+json")
			req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("client.Do() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if diff := cmp.Diff(tt.want.statusCode, resp.StatusCode); diff != "" {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("ステータスコードが期待値と異なります (-want +got):\n%s\nbody: %s", diff, string(body))
			}

			var batchResp BatchResponse
			if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
				t.Fatalf("json.Decode() error = %v", err)
			}

			if diff := cmp.Diff(tt.want.objectCount, len(batchResp.Objects)); diff != "" {
				t.Errorf("オブジェクト数が期待値と異なります (-want +got):\n%s", diff)
			}

			if tt.want.hasAllTransfer {
				if diff := cmp.Diff("basic", batchResp.Transfer); diff != "" {
					t.Errorf("transferフィールドが期待値と異なります (-want +got):\n%s", diff)
				}
			}

			if tt.want.hasAllHashAlgo {
				if diff := cmp.Diff("sha256", batchResp.HashAlgo); diff != "" {
					t.Errorf("hash_algoフィールドが期待値と異なります (-want +got):\n%s", diff)
				}
			}

			for i, obj := range batchResp.Objects {
				if tt.want.allAuthenticated && !obj.Authenticated {
					t.Errorf("オブジェクト[%d]のauthenticatedがfalseです", i)
				}

				if obj.Actions == nil && obj.Error == nil {
					t.Errorf("オブジェクト[%d]にactionsもerrorも含まれていません", i)
				}

				if tt.args.operation == "upload" && obj.Actions != nil {
					if obj.Actions.Upload == nil {
						t.Errorf("オブジェクト[%d]にuploadアクションが含まれていません", i)
					}
				}
			}
		})
	}
}

func TestBatchAPI_MultipleObjectsDownload(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type want struct {
		statusCode  int
		objectCount int
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "正常系: 複数オブジェクトを同時にダウンロード",
			want: want{
				statusCode:  http.StatusOK,
				objectCount: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := "na2na-p/test-repo"
			ref := "refs/heads/main"
			endpoint := GetBatchEndpoint(repository)

			token, err := GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", repository, ref),
				"aud":        "cargohold",
				"repository": repository,
				"ref":        ref,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			testFile1, err := CreateTestFile(1024)
			if err != nil {
				t.Fatalf("CreateTestFile() error = %v", err)
			}
			defer func() { _ = CleanupTestFiles(testFile1) }()

			testFile2, err := CreateTestFile(2048)
			if err != nil {
				t.Fatalf("CreateTestFile() error = %v", err)
			}
			defer func() { _ = CleanupTestFiles(testFile2) }()

			oid1, size1, err := CalculateFileHash(testFile1)
			if err != nil {
				t.Fatalf("CalculateFileHash() error = %v", err)
			}

			oid2, size2, err := CalculateFileHash(testFile2)
			if err != nil {
				t.Fatalf("CalculateFileHash() error = %v", err)
			}

			if err := uploadTestObject(endpoint, token, oid1, size1, testFile1); err != nil {
				t.Fatalf("uploadTestObject() error = %v", err)
			}
			if err := uploadTestObject(endpoint, token, oid2, size2, testFile2); err != nil {
				t.Fatalf("uploadTestObject() error = %v", err)
			}

			reqBody := map[string]interface{}{
				"operation": "download",
				"objects": []map[string]interface{}{
					{"oid": oid1, "size": size1},
					{"oid": oid2, "size": size2},
				},
				"transfers": []string{"basic"},
				"hash_algo": "sha256",
			}

			jsonData, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}

			req.Header.Set("Accept", "application/vnd.git-lfs+json")
			req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("client.Do() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if diff := cmp.Diff(tt.want.statusCode, resp.StatusCode); diff != "" {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("ステータスコードが期待値と異なります (-want +got):\n%s\nbody: %s", diff, string(body))
			}

			var batchResp BatchResponse
			if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
				t.Fatalf("json.Decode() error = %v", err)
			}

			if diff := cmp.Diff(tt.want.objectCount, len(batchResp.Objects)); diff != "" {
				t.Errorf("オブジェクト数が期待値と異なります (-want +got):\n%s", diff)
			}

			for i, obj := range batchResp.Objects {
				if obj.Actions == nil || obj.Actions.Download == nil {
					t.Errorf("オブジェクト[%d]にdownloadアクションが含まれていません", i)
					continue
				}
				if obj.Actions.Download.ExpiresIn <= 0 {
					t.Errorf("オブジェクト[%d]のexpires_inが正の値ではありません: got=%d", i, obj.Actions.Download.ExpiresIn)
				}
			}
		})
	}
}

func TestBatchAPI_DownloadUnauthorizedObjects(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type objectSpec struct {
		oid  string
		size int64
	}
	type args struct {
		objects []objectSpec
	}
	type want struct {
		statusCode int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "異常系: アクセスポリシーが存在しないオブジェクトのダウンロードは403を返す",
			args: args{
				objects: []objectSpec{
					{
						oid:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						size: 1024,
					},
					{
						oid:  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
						size: 2048,
					},
				},
			},
			want: want{
				statusCode: http.StatusForbidden,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := "na2na-p/test-repo"
			ref := "refs/heads/main"
			endpoint := GetBatchEndpoint(repository)

			token, err := GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", repository, ref),
				"aud":        "cargohold",
				"repository": repository,
				"ref":        ref,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			objects := make([]map[string]interface{}, len(tt.args.objects))
			for i, obj := range tt.args.objects {
				objects[i] = map[string]interface{}{
					"oid":  obj.oid,
					"size": obj.size,
				}
			}

			reqBody := map[string]interface{}{
				"operation": "download",
				"objects":   objects,
				"transfers": []string{"basic"},
				"hash_algo": "sha256",
			}

			jsonData, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}

			req.Header.Set("Accept", "application/vnd.git-lfs+json")
			req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("client.Do() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if diff := cmp.Diff(tt.want.statusCode, resp.StatusCode); diff != "" {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("ステータスコードが期待値と異なります (-want +got):\n%s\nbody: %s", diff, string(body))
			}
		})
	}
}

func uploadTestObject(batchEndpoint, token, oid string, size int64, filepath string) error {
	uploadURL, err := requestBatchAPIForURL(batchEndpoint, token, "upload", oid, size)
	if err != nil {
		return fmt.Errorf("requestBatchAPIForURL() error = %w", err)
	}

	if err := uploadFileToProxyForTest(uploadURL, filepath, token); err != nil {
		return fmt.Errorf("uploadFileToProxyForTest() error = %w", err)
	}

	verifyEndpoint, err := buildVerifyEndpointForTest(batchEndpoint)
	if err != nil {
		return fmt.Errorf("buildVerifyEndpointForTest() error = %w", err)
	}

	if err := verifyUploadForTest(verifyEndpoint, token, oid, size); err != nil {
		return fmt.Errorf("verifyUploadForTest() error = %w", err)
	}

	return nil
}

func requestBatchAPIForURL(endpoint, token, operation, oid string, size int64) (string, error) {
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
		return "", fmt.Errorf("json.Marshal() error = %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("http.NewRequest() error = %w", err)
	}

	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("client.Do() error = %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Batch APIがエラーを返しました: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var batchResp BatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return "", fmt.Errorf("json.Decode() error = %w", err)
	}

	if len(batchResp.Objects) == 0 {
		return "", fmt.Errorf("レスポンスにオブジェクトが含まれていません")
	}

	obj := batchResp.Objects[0]
	if obj.Error != nil {
		return "", fmt.Errorf("オブジェクトエラー: code=%d, message=%s", obj.Error.Code, obj.Error.Message)
	}

	if obj.Actions == nil {
		return "", fmt.Errorf("actionsがありません")
	}

	if operation == "upload" && obj.Actions.Upload != nil {
		return obj.Actions.Upload.Href, nil
	}
	if operation == "download" && obj.Actions.Download != nil {
		return obj.Actions.Download.Href, nil
	}

	return "", fmt.Errorf("対応するアクションがありません")
}

func uploadFileToProxyForTest(uploadURL, filepath, token string) error {
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("アップロードに失敗しました: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

func buildVerifyEndpointForTest(batchEndpoint string) (string, error) {
	parsedURL, err := url.Parse(batchEndpoint)
	if err != nil {
		return "", fmt.Errorf("URLのパースに失敗しました: %w", err)
	}

	dir := path.Dir(parsedURL.Path)
	parsedURL.Path = path.Join(dir, "verify")

	return parsedURL.String(), nil
}

func verifyUploadForTest(verifyEndpoint, token, oid string, size int64) error {
	reqBody := map[string]interface{}{
		"oid":  oid,
		"size": size,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("json.Marshal() error = %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, verifyEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("http.NewRequest() error = %w", err)
	}

	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("client.Do() error = %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Verify APIがエラーを返しました: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

func TestBatchAPI_ProxyEndpointURLFormat(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	validOID := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	type args struct {
		operation string
		oid       string
		size      int64
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常系: uploadレスポンスURLがProxyエンドポイント形式であること",
			args: args{
				operation: "upload",
				oid:       validOID,
				size:      1024,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := "na2na-p/test-repo"
			ref := "refs/heads/main"
			endpoint := GetBatchEndpoint(repository)

			token, err := GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", repository, ref),
				"aud":        "cargohold",
				"repository": repository,
				"ref":        ref,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			reqBody := map[string]interface{}{
				"operation": tt.args.operation,
				"objects": []map[string]interface{}{
					{
						"oid":  tt.args.oid,
						"size": tt.args.size,
					},
				},
				"transfers": []string{"basic"},
				"hash_algo": "sha256",
			}

			jsonData, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}

			req.Header.Set("Accept", "application/vnd.git-lfs+json")
			req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("client.Do() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("Batch APIがエラーを返しました: status=%d, body=%s", resp.StatusCode, string(body))
			}

			var batchResp BatchResponse
			if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
				t.Fatalf("json.Decode() error = %v", err)
			}

			if len(batchResp.Objects) == 0 {
				t.Fatal("レスポンスにオブジェクトが含まれていません")
			}

			obj := batchResp.Objects[0]
			if obj.Actions == nil {
				t.Fatal("actionsフィールドが含まれていません")
			}

			if obj.Actions.Upload == nil {
				t.Fatal("uploadアクションが含まれていません")
			}

			uploadURL := obj.Actions.Upload.Href

			expectedURLPattern := fmt.Sprintf("/%s/info/lfs/objects/%s", repository, tt.args.oid)
			if !strings.Contains(uploadURL, expectedURLPattern) {
				t.Errorf("URLがProxyエンドポイント形式ではありません: got=%s, want contains=%s", uploadURL, expectedURLPattern)
			}

			s3Keywords := []string{
				"s3.amazonaws.com",
				"s3-",
				".s3.",
				"X-Amz-",
				"amazonaws",
				"minio",
			}
			for _, keyword := range s3Keywords {
				if strings.Contains(uploadURL, keyword) {
					t.Errorf("URLにS3関連の情報が含まれています: url=%s, keyword=%s", uploadURL, keyword)
				}
			}

			baseURL := GetBaseEndpoint()
			if !strings.HasPrefix(uploadURL, baseURL) {
				t.Errorf("URLがベースURLから始まっていません: got=%s, want prefix=%s", uploadURL, baseURL)
			}
		})
	}
}

func TestBatchAPI_DownloadProxyEndpointURLFormat(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	tests := []struct {
		name string
	}{
		{
			name: "正常系: downloadレスポンスURLがProxyエンドポイント形式であること",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := "na2na-p/test-repo"
			ref := "refs/heads/main"
			endpoint := GetBatchEndpoint(repository)

			token, err := GenerateJWT(map[string]interface{}{
				"iss":        "https://token.actions.githubusercontent.com",
				"sub":        fmt.Sprintf("repo:%s:ref:%s", repository, ref),
				"aud":        "cargohold",
				"repository": repository,
				"ref":        ref,
				"actor":      "github-actions[bot]",
			})
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			testFile, err := CreateTestFile(1024)
			if err != nil {
				t.Fatalf("CreateTestFile() error = %v", err)
			}
			defer func() { _ = CleanupTestFiles(testFile) }()

			oid, size, err := CalculateFileHash(testFile)
			if err != nil {
				t.Fatalf("CalculateFileHash() error = %v", err)
			}

			if err := uploadTestObject(endpoint, token, oid, size, testFile); err != nil {
				t.Fatalf("uploadTestObject() error = %v", err)
			}

			reqBody := map[string]interface{}{
				"operation": "download",
				"objects": []map[string]interface{}{
					{"oid": oid, "size": size},
				},
				"transfers": []string{"basic"},
				"hash_algo": "sha256",
			}

			jsonData, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}

			req.Header.Set("Accept", "application/vnd.git-lfs+json")
			req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("client.Do() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("Batch APIがエラーを返しました: status=%d, body=%s", resp.StatusCode, string(body))
			}

			var batchResp BatchResponse
			if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
				t.Fatalf("json.Decode() error = %v", err)
			}

			if len(batchResp.Objects) == 0 {
				t.Fatal("レスポンスにオブジェクトが含まれていません")
			}

			obj := batchResp.Objects[0]
			if obj.Actions == nil || obj.Actions.Download == nil {
				t.Fatal("downloadアクションが含まれていません")
			}

			downloadURL := obj.Actions.Download.Href

			expectedURLPattern := fmt.Sprintf("/%s/info/lfs/objects/%s", repository, oid)
			if !strings.Contains(downloadURL, expectedURLPattern) {
				t.Errorf("URLがProxyエンドポイント形式ではありません: got=%s, want contains=%s", downloadURL, expectedURLPattern)
			}

			s3Keywords := []string{
				"s3.amazonaws.com",
				"s3-",
				".s3.",
				"X-Amz-",
				"amazonaws",
				"minio",
			}
			for _, keyword := range s3Keywords {
				if strings.Contains(downloadURL, keyword) {
					t.Errorf("URLにS3関連の情報が含まれています: url=%s, keyword=%s", downloadURL, keyword)
				}
			}

			baseURL := GetBaseEndpoint()
			if !strings.HasPrefix(downloadURL, baseURL) {
				t.Errorf("URLがベースURLから始まっていません: got=%s, want prefix=%s", downloadURL, baseURL)
			}
		})
	}
}
