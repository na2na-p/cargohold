package middleware_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/handler/middleware"
)

func TestNewAppError(t *testing.T) {
	type args struct {
		statusCode int
		message    string
		err        error
	}
	tests := []struct {
		name string
		args args
		want *middleware.AppError
	}{
		{
			name: "正常系: NewAppErrorでAppErrorが生成される",
			args: args{
				statusCode: http.StatusBadRequest,
				message:    "バリデーションエラー",
				err:        errors.New("invalid input"),
			},
			want: &middleware.AppError{
				StatusCode: http.StatusBadRequest,
				Message:    "バリデーションエラー",
				Err:        errors.New("invalid input"),
			},
		},
		{
			name: "正常系: errがnilでもAppErrorが生成される",
			args: args{
				statusCode: http.StatusNotFound,
				message:    "リソースが見つかりません",
				err:        nil,
			},
			want: &middleware.AppError{
				StatusCode: http.StatusNotFound,
				Message:    "リソースが見つかりません",
				Err:        nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := middleware.NewAppError(tt.args.statusCode, tt.args.message, tt.args.err)

			if got.StatusCode != tt.want.StatusCode {
				t.Errorf("NewAppError() StatusCode = %v, want %v", got.StatusCode, tt.want.StatusCode)
			}
			if got.Message != tt.want.Message {
				t.Errorf("NewAppError() Message = %v, want %v", got.Message, tt.want.Message)
			}
			if tt.want.Err != nil {
				if got.Err == nil || got.Err.Error() != tt.want.Err.Error() {
					t.Errorf("NewAppError() Err = %v, want %v", got.Err, tt.want.Err)
				}
			} else {
				if got.Err != nil {
					t.Errorf("NewAppError() Err = %v, want nil", got.Err)
				}
			}
		})
	}
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name    string
		appErr  *middleware.AppError
		wantMsg string
	}{
		{
			name: "正常系: Error()メソッドがMessageを返す",
			appErr: &middleware.AppError{
				StatusCode: http.StatusBadRequest,
				Message:    "バリデーションエラー",
				Err:        errors.New("some error"),
			},
			wantMsg: "バリデーションエラー",
		},
		{
			name: "正常系: 空のMessageでも空文字を返す",
			appErr: &middleware.AppError{
				StatusCode: http.StatusInternalServerError,
				Message:    "",
				Err:        nil,
			},
			wantMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.appErr.Error()
			if got != tt.wantMsg {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

type logCapture struct {
	buf    *bytes.Buffer
	level  slog.Level
	attrs  []slog.Attr
	msg    string
	called bool
}

type captureHandler struct {
	capture *logCapture
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.capture.called = true
	h.capture.level = r.Level
	h.capture.msg = r.Message
	r.Attrs(func(a slog.Attr) bool {
		h.capture.attrs = append(h.capture.attrs, a)
		return true
	})
	return nil
}

func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *captureHandler) WithGroup(_ string) slog.Handler {
	return h
}

func newLogCapture() (*logCapture, func()) {
	capture := &logCapture{
		buf: &bytes.Buffer{},
	}
	handler := &captureHandler{capture: capture}
	logger := slog.New(handler)
	oldLogger := slog.Default()
	slog.SetDefault(logger)
	return capture, func() {
		slog.SetDefault(oldLogger)
	}
}

func TestCustomHTTPErrorHandler(t *testing.T) {
	type args struct {
		err       error
		committed bool
		requestID string
		method    string
		path      string
	}
	type want struct {
		statusCode  int
		bodyContain string
		contentType string
		logLevel    slog.Level
		logCalled   bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "正常系: AppErrorの場合、指定されたステータスコードとメッセージが返される",
			args: args{
				err:       middleware.NewAppError(http.StatusBadRequest, "バリデーションエラー", errors.New("validation failed")),
				committed: false,
				requestID: "req-123",
				method:    http.MethodPost,
				path:      "/api/test",
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				bodyContain: "バリデーションエラー",
				contentType: "application/vnd.git-lfs+json",
				logLevel:    slog.LevelWarn,
				logCalled:   true,
			},
		},
		{
			name: "正常系: echo.HTTPErrorの場合、HTTPエラーのコードとメッセージが返される",
			args: args{
				err:       echo.NewHTTPError(http.StatusNotFound, "Not Found"),
				committed: false,
				requestID: "req-456",
				method:    http.MethodGet,
				path:      "/api/resource",
			},
			want: want{
				statusCode:  http.StatusNotFound,
				bodyContain: "Not Found",
				contentType: "application/vnd.git-lfs+json",
				logLevel:    slog.LevelWarn,
				logCalled:   true,
			},
		},
		{
			name: "正常系: 通常のerrorの場合、500エラーが返される",
			args: args{
				err:       errors.New("unexpected error"),
				committed: false,
				requestID: "req-789",
				method:    http.MethodPut,
				path:      "/api/update",
			},
			want: want{
				statusCode:  http.StatusInternalServerError,
				bodyContain: "サーバー内部エラーが発生しました",
				contentType: "application/vnd.git-lfs+json",
				logLevel:    slog.LevelError,
				logCalled:   true,
			},
		},
		{
			name: "正常系: 5xxエラーの場合、slog.Errorでログ出力される",
			args: args{
				err:       middleware.NewAppError(http.StatusServiceUnavailable, "サービス利用不可", errors.New("service down")),
				committed: false,
				requestID: "req-service",
				method:    http.MethodGet,
				path:      "/api/health",
			},
			want: want{
				statusCode:  http.StatusServiceUnavailable,
				bodyContain: "サービス利用不可",
				contentType: "application/vnd.git-lfs+json",
				logLevel:    slog.LevelError,
				logCalled:   true,
			},
		},
		{
			name: "正常系: 4xxエラーの場合、slog.Warnでログ出力される",
			args: args{
				err:       middleware.NewAppError(http.StatusUnauthorized, "認証が必要です", errors.New("auth required")),
				committed: false,
				requestID: "req-auth",
				method:    http.MethodPost,
				path:      "/api/protected",
			},
			want: want{
				statusCode:  http.StatusUnauthorized,
				bodyContain: "認証が必要です",
				contentType: "application/vnd.git-lfs+json",
				logLevel:    slog.LevelWarn,
				logCalled:   true,
			},
		},
		{
			name: "正常系: レスポンスがコミット済みの場合、何もしない",
			args: args{
				err:       errors.New("some error"),
				committed: true,
				requestID: "req-committed",
				method:    http.MethodGet,
				path:      "/api/test",
			},
			want: want{
				statusCode:  http.StatusOK,
				bodyContain: "",
				contentType: "",
				logLevel:    0,
				logCalled:   false,
			},
		},
		{
			name: "正常系: echo.HTTPErrorのMessageが文字列以外の場合、デフォルトメッセージが返される",
			args: args{
				err:       echo.NewHTTPError(http.StatusBadRequest, 12345),
				committed: false,
				requestID: "req-non-string",
				method:    http.MethodPost,
				path:      "/api/test",
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				bodyContain: "サーバー内部エラーが発生しました",
				contentType: "application/vnd.git-lfs+json",
				logLevel:    slog.LevelWarn,
				logCalled:   true,
			},
		},
		{
			name: "正常系: /auth/パスでAppErrorの場合、JSON形式でエラーが返される",
			args: args{
				err:       middleware.NewAppError(http.StatusUnauthorized, "認証に失敗しました", errors.New("auth failed")),
				committed: false,
				requestID: "req-auth-json",
				method:    http.MethodGet,
				path:      "/auth/github/callback",
			},
			want: want{
				statusCode:  http.StatusUnauthorized,
				bodyContain: `"error":"認証に失敗しました"`,
				contentType: "application/json",
				logLevel:    slog.LevelWarn,
				logCalled:   true,
			},
		},
		{
			name: "正常系: /auth/パスでecho.HTTPErrorの場合、JSON形式でエラーが返される",
			args: args{
				err:       echo.NewHTTPError(http.StatusForbidden, "アクセスが拒否されました"),
				committed: false,
				requestID: "req-auth-http-err",
				method:    http.MethodGet,
				path:      "/auth/callback",
			},
			want: want{
				statusCode:  http.StatusForbidden,
				bodyContain: `"error":"アクセスが拒否されました"`,
				contentType: "application/json",
				logLevel:    slog.LevelWarn,
				logCalled:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture, cleanup := newLogCapture()
			defer cleanup()

			e := echo.New()
			req := httptest.NewRequest(tt.args.method, tt.args.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.args.requestID != "" {
				c.Response().Header().Set(echo.HeaderXRequestID, tt.args.requestID)
			}

			if tt.args.committed {
				c.Response().WriteHeader(http.StatusOK)
			}

			middleware.CustomHTTPErrorHandler(tt.args.err, c)

			if tt.args.committed {
				if capture.called {
					t.Error("CustomHTTPErrorHandler() log should not be called when response is committed")
				}
				return
			}

			if rec.Code != tt.want.statusCode {
				t.Errorf("CustomHTTPErrorHandler() status code = %v, want %v", rec.Code, tt.want.statusCode)
			}

			body := rec.Body.String()
			if !strings.Contains(body, tt.want.bodyContain) {
				t.Errorf("CustomHTTPErrorHandler() body = %v, want to contain %v", body, tt.want.bodyContain)
			}

			contentType := rec.Header().Get(echo.HeaderContentType)
			if tt.want.contentType != "" && contentType != tt.want.contentType {
				t.Errorf("CustomHTTPErrorHandler() Content-Type = %v, want %v", contentType, tt.want.contentType)
			}

			if tt.want.logCalled {
				if !capture.called {
					t.Error("CustomHTTPErrorHandler() log should be called")
				}
				if capture.level != tt.want.logLevel {
					t.Errorf("CustomHTTPErrorHandler() log level = %v, want %v", capture.level, tt.want.logLevel)
				}
			}
		})
	}
}

func TestCustomHTTPErrorHandler_LogAttributes(t *testing.T) {
	capture, cleanup := newLogCapture()
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Response().Header().Set(echo.HeaderXRequestID, "test-request-id")

	testErr := errors.New("test error")
	middleware.CustomHTTPErrorHandler(testErr, c)

	if !capture.called {
		t.Fatal("CustomHTTPErrorHandler() log should be called")
	}

	attrMap := make(map[string]any)
	for _, attr := range capture.attrs {
		attrMap[attr.Key] = attr.Value.Any()
	}

	wantAttrs := map[string]any{
		"request_id": "test-request-id",
		"method":     http.MethodPost,
		"path":       "/api/test",
		"status":     int64(http.StatusInternalServerError),
	}

	for key, wantValue := range wantAttrs {
		gotValue, ok := attrMap[key]
		if !ok {
			t.Errorf("CustomHTTPErrorHandler() log missing attribute %q", key)
			continue
		}
		if diff := cmp.Diff(wantValue, gotValue); diff != "" {
			t.Errorf("CustomHTTPErrorHandler() log attribute %q mismatch (-want +got):\n%s", key, diff)
		}
	}

	if _, ok := attrMap["error"]; !ok {
		t.Error("CustomHTTPErrorHandler() log should contain error attribute")
	}
}
