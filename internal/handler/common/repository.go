package common

import (
	"fmt"

	"github.com/labstack/echo/v5"
	"github.com/na2na-p/cargohold/internal/domain"
)

func ExtractRepositoryIdentifier(c *echo.Context) (*domain.RepositoryIdentifier, error) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	fullName := fmt.Sprintf("%s/%s", owner, repo)
	return domain.NewRepositoryIdentifier(fullName)
}
