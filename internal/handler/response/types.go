package response

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

const GitLFSContentType = "application/vnd.git-lfs+json"

type LFSErrorResponse struct {
	Message string `json:"message"`
}

func SendLFSError(c echo.Context, statusCode int, message string) error {
	c.Response().Header().Set(echo.HeaderContentType, GitLFSContentType)

	if statusCode == http.StatusUnauthorized {
		c.Response().Header().Set("LFS-Authenticate", `Basic realm="Git LFS"`)
	}

	return c.JSON(statusCode, LFSErrorResponse{
		Message: message,
	})
}
