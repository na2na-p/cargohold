package handler

import (
	"github.com/labstack/echo/v5"
	"github.com/na2na-p/cargohold/internal/handler/response"
)

type LFSErrorResponse = response.LFSErrorResponse

func SendLFSError(c *echo.Context, statusCode int, message string) error {
	return response.SendLFSError(c, statusCode, message)
}
