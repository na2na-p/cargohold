//go:generate mockgen -source=$GOFILE -destination=../../../tests/handler/auth/mock_github_login.go -package=auth
package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/domain"
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
		ctx := c.Request().Context()

		host := c.Request().Host
		if !IsHostAllowed(host, cfg.AllowedHosts) {
			slog.WarnContext(ctx, "Host header validation failed",
				slog.String("host", host),
				slog.Any("allowed_hosts", cfg.AllowedHosts),
			)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "許可されていないホストからのリクエストです",
			})
		}

		repositoryParam := c.QueryParam("repository")
		if repositoryParam == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "repositoryパラメータが指定されていません",
			})
		}

		repository, err := domain.NewRepositoryIdentifier(repositoryParam)
		if err != nil {
			slog.WarnContext(ctx, "Repository validation failed",
				slog.String("repository_param", repositoryParam),
				slog.String("error", err.Error()),
			)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "repositoryパラメータの形式が不正です",
			})
		}

		scheme := ResolveScheme(c, cfg.TrustProxy)
		redirectURI := scheme + "://" + host + "/auth/github/callback"

		authURL, err := githubOAuthUC.StartAuthentication(ctx, repository, redirectURI)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "認証URLの生成に失敗しました",
			})
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
