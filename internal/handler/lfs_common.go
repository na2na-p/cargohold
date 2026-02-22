package handler

import (
	"errors"
	"mime"

	"github.com/labstack/echo/v5"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/handler/common"
	"github.com/na2na-p/cargohold/internal/handler/response"
)

const GitLFSContentType = response.GitLFSContentType

func ValidateLFSHeaders(c *echo.Context) error {
	accept := c.Request().Header.Get(echo.HeaderAccept)
	contentType := c.Request().Header.Get(echo.HeaderContentType)

	acceptMediaType, _, err := mime.ParseMediaType(accept)
	if err != nil || acceptMediaType != GitLFSContentType {
		return errors.New("accept ヘッダーは application/vnd.git-lfs+json である必要があります")
	}

	contentMediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || contentMediaType != GitLFSContentType {
		return errors.New("Content-Typeは application/vnd.git-lfs+json である必要があります")
	}

	return nil
}

func ExtractRepositoryIdentifier(c *echo.Context) (*domain.RepositoryIdentifier, error) {
	return common.ExtractRepositoryIdentifier(c)
}
