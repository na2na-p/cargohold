package auth

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v5"
	"github.com/na2na-p/cargohold/internal/handler/common"
	"github.com/na2na-p/cargohold/internal/handler/middleware"
	"github.com/na2na-p/cargohold/internal/usecase"
)

func GitHubCallbackHandler(githubOAuthUC GitHubOAuthUseCaseInterface) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ctx := c.Request().Context()

		code := c.QueryParam("code")
		if code == "" {
			return middleware.NewAppError(
				http.StatusBadRequest,
				"codeパラメータが指定されていません",
				errors.New("code parameter is empty"),
			)
		}

		state := c.QueryParam("state")
		if state == "" {
			return middleware.NewAppError(
				http.StatusBadRequest,
				"stateパラメータが指定されていません",
				errors.New("state parameter is empty"),
			)
		}

		sessionID, err := githubOAuthUC.HandleCallback(ctx, code, state)
		if err != nil {
			return handleCallbackError(err)
		}

		cookie := &http.Cookie{
			Name:     common.LFSSessionCookieName,
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   common.LFSSessionMaxAge,
		}
		c.SetCookie(cookie)

		host := c.Request().Host
		redirectURL := "/auth/session?session_id=" + sessionID + "&host=" + url.QueryEscape(host)
		return c.Redirect(http.StatusFound, redirectURL)
	}
}

func handleCallbackError(err error) error {
	if errors.Is(err, usecase.ErrInvalidState) {
		return middleware.NewAppError(
			http.StatusUnauthorized,
			"認証セッションが無効または期限切れです",
			err,
		)
	}

	if errors.Is(err, usecase.ErrRepositoryAccessDenied) {
		return middleware.NewAppError(
			http.StatusForbidden,
			"リポジトリへのアクセス権がありません",
			err,
		)
	}

	if errors.Is(err, usecase.ErrCodeExchangeFailed) {
		return middleware.NewAppError(
			http.StatusUnauthorized,
			"認証に失敗しました",
			err,
		)
	}

	if errors.Is(err, usecase.ErrUserInfoFailed) {
		return middleware.NewAppError(
			http.StatusUnauthorized,
			"ユーザー情報の取得に失敗しました",
			err,
		)
	}

	return middleware.NewAppError(
		http.StatusInternalServerError,
		"認証処理に失敗しました",
		err,
	)
}
