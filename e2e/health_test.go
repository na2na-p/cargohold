//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func GetHealthzEndpoint() string {
	return fmt.Sprintf("%s/healthz", GetBaseEndpoint())
}

func GetReadyzEndpoint() string {
	return fmt.Sprintf("%s/readyz", GetBaseEndpoint())
}

func TestHealthzEndpoint_Get(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type want struct {
		statusCode int
		body       map[string]string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "正常系: ヘルスチェックが成功し、healthy状態が返る",
			want: want{
				statusCode: http.StatusOK,
				body: map[string]string{
					"status": "healthy",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Timeout: 30 * time.Second}

			resp, err := client.Get(GetHealthzEndpoint())
			if err != nil {
				t.Fatalf("HTTPリクエストに失敗しました: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if diff := cmp.Diff(tt.want.statusCode, resp.StatusCode); diff != "" {
				t.Errorf("ステータスコードが一致しません (-want +got):\n%s", diff)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("レスポンスボディの読み取りに失敗しました: %v", err)
			}

			var got map[string]string
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("JSONのパースに失敗しました: %v", err)
			}

			if diff := cmp.Diff(tt.want.body, got); diff != "" {
				t.Errorf("レスポンスボディが一致しません (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadyzEndpoint_Get(t *testing.T) {
	if err := SetupE2EEnvironment(); err != nil {
		t.Fatalf("E2E環境のセットアップに失敗: %v", err)
	}

	type serviceDetail struct {
		Name    string `json:"name"`
		Healthy bool   `json:"healthy"`
	}
	type readyzResponse struct {
		Status  string          `json:"status"`
		Details []serviceDetail `json:"details"`
	}
	type want struct {
		statusCode       int
		status           string
		requiredServices []string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "正常系: 全サービスが正常でready状態が返る",
			want: want{
				statusCode:       http.StatusOK,
				status:           "ready",
				requiredServices: []string{"postgres", "redis", "s3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Timeout: 30 * time.Second}

			resp, err := client.Get(GetReadyzEndpoint())
			if err != nil {
				t.Fatalf("HTTPリクエストに失敗しました: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if diff := cmp.Diff(tt.want.statusCode, resp.StatusCode); diff != "" {
				t.Errorf("ステータスコードが一致しません (-want +got):\n%s", diff)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("レスポンスボディの読み取りに失敗しました: %v", err)
			}

			var got readyzResponse
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("JSONのパースに失敗しました: %v", err)
			}

			if diff := cmp.Diff(tt.want.status, got.Status); diff != "" {
				t.Errorf("statusフィールドが一致しません (-want +got):\n%s", diff)
			}

			serviceMap := make(map[string]bool)
			for _, detail := range got.Details {
				serviceMap[detail.Name] = detail.Healthy
			}

			for _, requiredService := range tt.want.requiredServices {
				healthy, exists := serviceMap[requiredService]
				if !exists {
					t.Errorf("必須サービス %q がdetailsに含まれていません", requiredService)
					continue
				}
				if !healthy {
					t.Errorf("サービス %q がhealthyではありません", requiredService)
				}
			}
		})
	}
}
