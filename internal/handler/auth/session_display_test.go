package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/handler/auth"
	"github.com/na2na-p/cargohold/internal/handler/middleware"
)

func TestSessionDisplayHandler(t *testing.T) {
	type args struct {
		sessionID string
		host      string
	}
	tests := []struct {
		name             string
		args             args
		expectedStatus   int
		wantAppError     bool
		expectedContains []string
	}{
		{
			name: "正常系: セッションIDとホストが指定された場合、HTMLが返却される",
			args: args{
				sessionID: "test-session-id-12345",
				host:      "cargohold.example.com",
			},
			expectedStatus: http.StatusOK,
			wantAppError:   false,
			expectedContains: []string{
				"test-session-id-12345",
				"cargohold.example.com",
				"git credential approve",
				"protocol=https",
				"username=x-session",
				"24",
			},
		},
		{
			name: "異常系: session_idパラメータが空の場合はBadRequestを返す",
			args: args{
				sessionID: "",
				host:      "cargohold.example.com",
			},
			expectedStatus: http.StatusBadRequest,
			wantAppError:   true,
		},
		{
			name: "異常系: hostパラメータが空の場合はBadRequestを返す",
			args: args{
				sessionID: "test-session-id-12345",
				host:      "",
			},
			expectedStatus: http.StatusBadRequest,
			wantAppError:   true,
		},
		{
			name: "異常系: session_idとhostの両方が空の場合はBadRequestを返す",
			args: args{
				sessionID: "",
				host:      "",
			},
			expectedStatus: http.StatusBadRequest,
			wantAppError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			url := "/auth/session"
			queryParams := ""
			if tt.args.sessionID != "" {
				queryParams += "session_id=" + tt.args.sessionID
			}
			if tt.args.host != "" {
				if queryParams != "" {
					queryParams += "&"
				}
				queryParams += "host=" + tt.args.host
			}
			if queryParams != "" {
				url += "?" + queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := auth.SessionDisplayHandler()

			err := handler(c)

			if tt.wantAppError {
				if err == nil {
					t.Fatal("expected AppError, got nil")
				}
				appErr, ok := err.(*middleware.AppError)
				if !ok {
					t.Fatalf("expected *middleware.AppError, got %T", err)
				}
				if appErr.StatusCode != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, appErr.StatusCode)
				}
			} else {
				if err != nil {
					t.Fatalf("handler returned error: %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
				}

				contentType := rec.Header().Get("Content-Type")
				if !strings.Contains(contentType, "text/html") {
					t.Errorf("expected Content-Type text/html, got %s", contentType)
				}

				body := rec.Body.String()
				for _, expected := range tt.expectedContains {
					if !strings.Contains(body, expected) {
						t.Errorf("expected body to contain %q, but it didn't.\nBody: %s", expected, body)
					}
				}
			}
		})
	}
}
