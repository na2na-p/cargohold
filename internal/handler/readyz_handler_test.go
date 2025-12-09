package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/handler"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_handler "github.com/na2na-p/cargohold/tests/handler"
	"go.uber.org/mock/gomock"
)

func TestReadyzHandler_Handle(t *testing.T) {
	type fields struct {
		setupMock func(ctrl *gomock.Controller) handler.ReadinessUseCaseInterface
	}
	tests := []struct {
		name           string
		fields         fields
		wantStatusCode int
		wantStatus     string
	}{
		{
			name: "正常系: すべてのヘルスチェックが成功した場合、200 OKが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) handler.ReadinessUseCaseInterface {
					mockUC := mock_handler.NewMockReadinessUseCaseInterface(ctrl)
					mockUC.EXPECT().ExecuteDetails(gomock.Any()).Return([]usecase.HealthCheckResult{
						{Name: "postgres", Healthy: true, Error: nil},
						{Name: "redis", Healthy: true, Error: nil},
						{Name: "s3", Healthy: true, Error: nil},
					}, nil)
					return mockUC
				},
			},
			wantStatusCode: http.StatusOK,
			wantStatus:     "ready",
		},
		{
			name: "異常系: ヘルスチェックが失敗した場合、503 Service Unavailableが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) handler.ReadinessUseCaseInterface {
					mockUC := mock_handler.NewMockReadinessUseCaseInterface(ctrl)
					mockUC.EXPECT().ExecuteDetails(gomock.Any()).Return([]usecase.HealthCheckResult{
						{Name: "postgres", Healthy: false, Error: errors.New("connection refused")},
					}, usecase.ErrHealthCheckFailed)
					return mockUC
				},
			},
			wantStatusCode: http.StatusServiceUnavailable,
			wantStatus:     "not ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			uc := tt.fields.setupMock(ctrl)
			h := handler.NewReadyzHandler(uc)

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.Handle(c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if rec.Code != tt.wantStatusCode {
				t.Errorf("status code = %v, want %v", rec.Code, tt.wantStatusCode)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			status, ok := response["status"].(string)
			if !ok {
				t.Fatal("status field not found or not a string")
			}
			if status != tt.wantStatus {
				t.Errorf("status = %v, want %v", status, tt.wantStatus)
			}
		})
	}
}
