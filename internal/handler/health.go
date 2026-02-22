package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func HealthHandler(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
