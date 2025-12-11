//go:build e2e

// Package e2e はE2Eテストを提供します
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestBatchAPI_ValidationError(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	validOID := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	shortOID := "abc123"
	longOID := strings.Repeat("a", 100)
	nonHexOID := "ghijkl1234567890ghijkl1234567890ghijkl1234567890ghijkl1234567890"

	type args struct {
		reqBody map[string]interface{}
	}
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
	}{
		{
			name: "異常系: operationが不正な場合、422エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"operation": "invalid-operation",
					"objects": []map[string]interface{}{
						{
							"oid":  validOID,
							"size": 1024,
						},
					},
					"transfers": []string{"basic"},
					"hash_algo": "sha256",
				},
			},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name: "異常系: OID形式が不正な場合（短すぎる）、422エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  shortOID,
							"size": 1024,
						},
					},
					"transfers": []string{"basic"},
					"hash_algo": "sha256",
				},
			},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name: "異常系: OID形式が不正な場合（長すぎる）、422エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  longOID,
							"size": 1024,
						},
					},
					"transfers": []string{"basic"},
					"hash_algo": "sha256",
				},
			},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name: "異常系: OID形式が不正な場合（非16進数文字）、422エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  nonHexOID,
							"size": 1024,
						},
					},
					"transfers": []string{"basic"},
					"hash_algo": "sha256",
				},
			},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name: "正常系: sizeが0の場合、リクエストが受け入れられる",
			args: args{
				reqBody: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  validOID,
							"size": 0,
						},
					},
					"transfers": []string{"basic"},
					"hash_algo": "sha256",
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "異常系: sizeが負の値の場合、422エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  validOID,
							"size": -1,
						},
					},
					"transfers": []string{"basic"},
					"hash_algo": "sha256",
				},
			},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name: "正常系: sizeがnullの場合、デフォルト値(0)として処理される",
			args: args{
				reqBody: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  validOID,
							"size": nil,
						},
					},
					"transfers": []string{"basic"},
					"hash_algo": "sha256",
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "異常系: objectsが空配列の場合、422エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"operation": "upload",
					"objects":   []map[string]interface{}{},
					"transfers": []string{"basic"},
					"hash_algo": "sha256",
				},
			},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name: "異常系: hash_algoがsha256以外の場合、422エラーが返る",
			args: args{
				reqBody: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  validOID,
							"size": 1024,
						},
					},
					"transfers": []string{"basic"},
					"hash_algo": "md5",
				},
			},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name: "正常系: transfersがbasic以外の場合、サーバーは無視してリクエストを処理する",
			args: args{
				reqBody: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  validOID,
							"size": 1024,
						},
					},
					"transfers": []string{"multipart"},
					"hash_algo": "sha256",
				},
			},
			wantStatusCode: http.StatusOK,
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

			jsonData, err := json.Marshal(tt.args.reqBody)
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

			if diff := cmp.Diff(tt.wantStatusCode, resp.StatusCode); diff != "" {
				t.Errorf("ステータスコードが期待値と異なります (-want +got):\n%s", diff)
			}
		})
	}
}
