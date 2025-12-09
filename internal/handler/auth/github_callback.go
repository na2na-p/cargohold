package auth

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/usecase"
)

const (
	LFSSessionCookieName = "lfs_session"
	LFSSessionMaxAge     = 86400
)

func GitHubCallbackHandler(githubOAuthUC GitHubOAuthUseCaseInterface) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		code := c.QueryParam("code")
		if code == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "codeパラメータが指定されていません",
			})
		}

		state := c.QueryParam("state")
		if state == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "stateパラメータが指定されていません",
			})
		}

		sessionID, err := githubOAuthUC.HandleCallback(ctx, code, state)
		if err != nil {
			return handleCallbackError(c, err)
		}

		cookie := &http.Cookie{
			Name:     LFSSessionCookieName,
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   LFSSessionMaxAge,
		}
		c.SetCookie(cookie)

		return c.Redirect(http.StatusFound, "/")
	}
}

func handleCallbackError(c echo.Context, err error) error {
	if errors.Is(err, usecase.ErrInvalidState) {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "認証セッションが無効または期限切れです",
		})
	}

	if errors.Is(err, usecase.ErrRepositoryAccessDenied) {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "リポジトリへのアクセス権がありません",
		})
	}

	if errors.Is(err, usecase.ErrCodeExchangeFailed) {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "認証に失敗しました",
		})
	}

	if errors.Is(err, usecase.ErrUserInfoFailed) {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "ユーザー情報の取得に失敗しました",
		})
	}

	return c.JSON(http.StatusInternalServerError, map[string]string{
		"error": "認証処理に失敗しました",
	})
}
