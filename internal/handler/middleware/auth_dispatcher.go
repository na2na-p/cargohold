//go:generate mockgen -source=$GOFILE -destination=../../../tests/handler/middleware/mock_auth_dispatcher.go -package=middleware
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/handler/common"
	"github.com/na2na-p/cargohold/internal/handler/response"
)

var (
	ErrRepositoryMismatch    = errors.New("repository mismatch")
	ErrInvalidRepositoryPath = errors.New("invalid repository path")
)

type AuthUseCaseInterface interface {
	AuthenticateSession(ctx context.Context, sessionID string) (*domain.UserInfo, error)
	AuthenticateGitHubOIDC(ctx context.Context, token string) (*domain.UserInfo, error)
}

func AuthDispatcher(authUC AuthUseCaseInterface) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			authHeader := c.Request().Header.Get("Authorization")

			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				userInfo, err := authUC.AuthenticateGitHubOIDC(ctx, token)
				if err != nil {
					return response.SendLFSError(c, http.StatusUnauthorized, "Unauthorized")
				}
				if err := validateRepository(c, userInfo); err != nil {
					return err
				}
				c.Set(UserInfoContextKey, userInfo)
				return next(c)
			}

			cookie, err := c.Cookie(SessionCookieName)
			if err == nil && cookie.Value != "" {
				userInfo, err := authUC.AuthenticateSession(ctx, cookie.Value)
				if err == nil {
					if err := validateRepository(c, userInfo); err != nil {
						return err
					}
					c.Set(UserInfoContextKey, userInfo)
					return next(c)
				}
			}

			return response.SendLFSError(c, http.StatusUnauthorized, "Unauthorized")
		}
	}
}

func validateRepository(c echo.Context, userInfo *domain.UserInfo) error {
	urlRepoIdentifier, err := common.ExtractRepositoryIdentifier(c)
	if err != nil {
		sendErr := response.SendLFSError(c, http.StatusBadRequest, "Invalid repository path")
		if sendErr != nil {
			return sendErr
		}
		return ErrInvalidRepositoryPath
	}

	userRepoIdentifier := userInfo.Repository()
	if userRepoIdentifier == nil {
		sendErr := response.SendLFSError(c, http.StatusForbidden, "Forbidden")
		if sendErr != nil {
			return sendErr
		}
		return ErrRepositoryMismatch
	}

	if !urlRepoIdentifier.EqualsFold(userRepoIdentifier) {
		sendErr := response.SendLFSError(c, http.StatusForbidden, "Forbidden")
		if sendErr != nil {
			return sendErr
		}
		return ErrRepositoryMismatch
	}

	return nil
}

const (
	SessionCookieName  = "session_id"
	UserInfoContextKey = "user_info"
)
