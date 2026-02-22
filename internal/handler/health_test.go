package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/labstack/echo/v5"

	"github.com/na2na-p/cargohold/internal/handler"
)

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name           string
		wantStatusCode int
		wantBody       map[string]string
	}{
		{
			name:           "正常系: ステータス200とhealthyレスポンスが返却される",
			wantStatusCode: http.StatusOK,
			wantBody: map[string]string{
				"status": "healthy",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.HealthHandler(c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if rec.Code != tt.wantStatusCode {
				t.Errorf("status code = %v, want %v", rec.Code, tt.wantStatusCode)
			}

			var gotBody map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &gotBody); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}

			if diff := cmp.Diff(tt.wantBody, gotBody); diff != "" {
				t.Errorf("response body mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
