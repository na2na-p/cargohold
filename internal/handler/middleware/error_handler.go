package middleware

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/handler/response"
)

type AppError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(statusCode int, message string, err error) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}

func CustomHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	requestID := c.Response().Header().Get(echo.HeaderXRequestID)

	var statusCode int
	var message string
	var originalErr error

	var appErr *AppError
	if errors.As(err, &appErr) {
		statusCode = appErr.StatusCode
		message = appErr.Message
		originalErr = appErr.Err
	} else {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			statusCode = httpErr.Code
			if msg, ok := httpErr.Message.(string); ok {
				message = msg
			} else {
				message = "サーバー内部エラーが発生しました"
			}
		} else {
			statusCode = http.StatusInternalServerError
			message = "サーバー内部エラーが発生しました"
		}
		originalErr = err
	}

	logAttrs := []any{
		"request_id", requestID,
		"method", c.Request().Method,
		"path", c.Request().URL.Path,
		"status", statusCode,
	}
	if originalErr != nil {
		logAttrs = append(logAttrs, "error", originalErr)
	}

	if statusCode >= 500 {
		slog.Error("サーバーエラー", logAttrs...)
	} else if statusCode >= 400 {
		slog.Warn("クライアントエラー", logAttrs...)
	}

	c.Response().Header().Set(echo.HeaderContentType, response.GitLFSContentType)
	data, marshalErr := json.Marshal(response.LFSErrorResponse{Message: message})
	if marshalErr != nil {
		slog.Error("JSONマーシャルに失敗しました",
			"request_id", requestID,
			"error", marshalErr,
		)
		return
	}
	if blobErr := c.Blob(statusCode, response.GitLFSContentType, data); blobErr != nil {
		slog.Error("レスポンスの送信に失敗しました",
			"request_id", requestID,
			"status_code", statusCode,
			"message", message,
			"error", blobErr,
		)
	}
}
