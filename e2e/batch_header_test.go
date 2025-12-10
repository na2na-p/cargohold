//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestBatchAPI_HeaderValidation(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type args struct {
		acceptHeader      string
		contentTypeHeader string
		setAccept         bool
		setContentType    bool
	}
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
		wantErrMessage string
	}{
		{
			name: "異常系: Acceptヘッダーが不正な場合、400エラーが返る",
			args: args{
				acceptHeader:      "application/json",
				contentTypeHeader: "application/vnd.git-lfs+json",
				setAccept:         true,
				setContentType:    true,
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrMessage: "accept ヘッダーは application/vnd.git-lfs+json である必要があります",
		},
		{
			name: "異常系: Content-Typeヘッダーが不正な場合、400エラーが返る",
			args: args{
				acceptHeader:      "application/vnd.git-lfs+json",
				contentTypeHeader: "application/json",
				setAccept:         true,
				setContentType:    true,
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrMessage: "Content-Typeは application/vnd.git-lfs+json である必要があります",
		},
		{
			name: "異常系: Acceptヘッダーが欠落している場合、400エラーが返る",
			args: args{
				acceptHeader:      "",
				contentTypeHeader: "application/vnd.git-lfs+json",
				setAccept:         false,
				setContentType:    true,
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrMessage: "accept ヘッダーは application/vnd.git-lfs+json である必要があります",
		},
		{
			name: "異常系: Content-Typeヘッダーが欠落している場合、400エラーが返る",
			args: args{
				acceptHeader:      "application/vnd.git-lfs+json",
				contentTypeHeader: "",
				setAccept:         true,
				setContentType:    false,
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrMessage: "Content-Typeは application/vnd.git-lfs+json である必要があります",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := "na2na-p/test-repo"
			ref := "refs/heads/main"

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
				"operation": "upload",
				"objects": []map[string]interface{}{
					{
						"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
						"size": 1024,
					},
				},
				"transfers": []string{"basic"},
				"hash_algo": "sha256",
			}

			jsonData, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("JSONのマーシャルに失敗しました: %v", err)
			}

			endpoint := GetBatchEndpoint(repository)
			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
			if err != nil {
				t.Fatalf("リクエストの作成に失敗しました: %v", err)
			}

			if tt.args.setAccept {
				req.Header.Set("Accept", tt.args.acceptHeader)
			}
			if tt.args.setContentType {
				req.Header.Set("Content-Type", tt.args.contentTypeHeader)
			}
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("リクエストの送信に失敗しました: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if diff := cmp.Diff(tt.wantStatusCode, resp.StatusCode); diff != "" {
				t.Errorf("ステータスコードが一致しません (-want +got):\n%s", diff)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("レスポンスボディの読み取りに失敗しました: %v", err)
			}

			var errResp struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(body, &errResp); err != nil {
				t.Fatalf("レスポンスのパースに失敗しました: %v, body=%s", err, string(body))
			}

			if diff := cmp.Diff(tt.wantErrMessage, errResp.Message); diff != "" {
				t.Errorf("エラーメッセージが一致しません (-want +got):\n%s", diff)
			}
		})
	}
}
