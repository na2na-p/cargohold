//go:generate mockgen -source=$GOFILE -destination=../../../tests/handler/auth/mock_github_login.go -package=auth
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/handler/middleware"
)

type GitHubOAuthUseCaseInterface interface {
	StartAuthentication(
		ctx context.Context,
		repository *domain.RepositoryIdentifier,
		redirectURI string,
	) (string, error)
	HandleCallback(
		ctx context.Context,
		code string,
		state string,
	) (string, error)
}

type GitHubLoginHandlerConfig struct {
	TrustProxy   bool
	AllowedHosts []string
}

func GitHubLoginHandler(githubOAuthUC GitHubOAuthUseCaseInterface, cfg GitHubLoginHandlerConfig) echo.HandlerFunc {
	return func(c echo.Context) error {
		host := c.Request().Host
		if !IsHostAllowed(host, cfg.AllowedHosts) {
			return middleware.NewAppError(
				http.StatusBadRequest,
				"許可されていないホストからのリクエストです",
				fmt.Errorf("host %q is not in allowed list %v", host, cfg.AllowedHosts),
			)
		}

		repositoryParam := c.QueryParam("repository")
		if repositoryParam == "" {
			return middleware.NewAppError(
				http.StatusBadRequest,
				"repositoryパラメータが指定されていません",
				errors.New("repository parameter is empty"),
			)
		}

		repository, err := domain.NewRepositoryIdentifier(repositoryParam)
		if err != nil {
			return middleware.NewAppError(
				http.StatusBadRequest,
				"repositoryパラメータの形式が不正です",
				fmt.Errorf("invalid repository %q: %w", repositoryParam, err),
			)
		}

		scheme := ResolveScheme(c, cfg.TrustProxy)
		redirectURI := scheme + "://" + host + "/auth/github/callback"

		authURL, err := githubOAuthUC.StartAuthentication(c.Request().Context(), repository, redirectURI)
		if err != nil {
			return middleware.NewAppError(
				http.StatusInternalServerError,
				"認証URLの生成に失敗しました",
				err,
			)
		}

		return c.Redirect(http.StatusFound, authURL)
	}
}

func IsHostAllowed(host string, allowedHosts []string) bool {
	if len(allowedHosts) == 0 {
		return true
	}
	if host == "" {
		return false
	}
	for _, allowed := range allowedHosts {
		if host == allowed {
			return true
		}
	}
	return false
}

func ResolveScheme(c echo.Context, trustProxy bool) string {
	if trustProxy {
		forwarded := strings.ToLower(c.Request().Header.Get("X-Forwarded-Proto"))
		if forwarded == "http" || forwarded == "https" {
			return forwarded
		}
	}
	if c.Request().TLS != nil {
		return "https"
	}
	return "http"
}
