package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/handler/dto"
	"github.com/na2na-p/cargohold/internal/handler/middleware"
	"github.com/na2na-p/cargohold/internal/usecase"
)

const maxBodySize = 10 << 20

type BatchHandler struct {
	batchUseCase usecase.BatchUseCaseInterface
}

func NewBatchHandler(batchUseCase usecase.BatchUseCaseInterface) *BatchHandler {
	return &BatchHandler{
		batchUseCase: batchUseCase,
	}
}

func (h *BatchHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	if err := ValidateLFSHeaders(c); err != nil {
		return SendLFSError(c, http.StatusBadRequest, err.Error())
	}

	repoID, err := ExtractRepositoryIdentifier(c)
	if err != nil {
		return SendLFSError(c, http.StatusBadRequest, "リポジトリ識別子の形式が不正です")
	}

	var reqDTO dto.BatchRequestDTO
	bodyBytes, readErr := io.ReadAll(io.LimitReader(c.Request().Body, maxBodySize+1))
	if readErr != nil {
		return SendLFSError(c, http.StatusUnprocessableEntity, "リクエストボディのパースに失敗しました")
	}
	if len(bodyBytes) > maxBodySize {
		return SendLFSError(c, http.StatusRequestEntityTooLarge, "リクエストボディが大きすぎます")
	}
	if err := json.Unmarshal(bodyBytes, &reqDTO); err != nil {
		return SendLFSError(c, http.StatusUnprocessableEntity, "リクエストボディのパースに失敗しました")
	}

	req, err := reqDTO.ToBatchRequest(repoID)
	if err != nil {
		return SendLFSError(c, http.StatusUnprocessableEntity, "リクエストボディのパースに失敗しました")
	}

	if err := h.checkPermissions(c, reqDTO.Operation); err != nil {
		return err
	}

	baseURL := getBaseURL(c)
	owner := repoID.Owner()
	repo := repoID.Name()

	resp, err := h.batchUseCase.HandleBatchRequest(ctx, baseURL, owner, repo, req)
	if err != nil {
		return h.handleUseCaseError(c, err)
	}

	c.Response().Header().Set(echo.HeaderContentType, GitLFSContentType)
	return c.JSON(http.StatusOK, resp)
}

func (h *BatchHandler) checkPermissions(c echo.Context, operation string) error {
	userInfoRaw := c.Get(middleware.UserInfoContextKey)
	if userInfoRaw == nil {
		return middleware.NewAppError(http.StatusForbidden, "認証情報が見つかりません", nil)
	}

	userInfo, ok := userInfoRaw.(*domain.UserInfo)
	if !ok {
		return middleware.NewAppError(http.StatusForbidden, "認証情報が見つかりません", nil)
	}

	permissions := userInfo.Permissions()
	if permissions == nil {
		return middleware.NewAppError(http.StatusForbidden, "このオペレーションを実行する権限がありません", nil)
	}

	switch operation {
	case "upload":
		if !permissions.CanUpload() {
			return middleware.NewAppError(http.StatusForbidden, "このオペレーションを実行する権限がありません", nil)
		}
	case "download":
		if !permissions.CanDownload() {
			return middleware.NewAppError(http.StatusForbidden, "このオペレーションを実行する権限がありません", nil)
		}
	}

	return nil
}

func getBaseURL(c echo.Context) string {
	scheme := "http"
	if c.Request().TLS != nil {
		scheme = "https"
	}
	if proto := c.Request().Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return scheme + "://" + c.Request().Host
}

func (h *BatchHandler) handleUseCaseError(_ echo.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrInvalidOperation):
		return middleware.NewAppError(http.StatusUnprocessableEntity, "不正なオペレーションです", err)
	case errors.Is(err, usecase.ErrNoObjects):
		return middleware.NewAppError(http.StatusUnprocessableEntity, "オブジェクトが指定されていません", err)
	case errors.Is(err, usecase.ErrInvalidOID):
		return middleware.NewAppError(http.StatusUnprocessableEntity, "不正なOIDです", err)
	case errors.Is(err, usecase.ErrInvalidSize):
		return middleware.NewAppError(http.StatusUnprocessableEntity, "不正なサイズです", err)
	case errors.Is(err, usecase.ErrInvalidHashAlgorithm):
		return middleware.NewAppError(http.StatusUnprocessableEntity, "不正なハッシュアルゴリズムです", err)
	case errors.Is(err, usecase.ErrAccessDenied):
		return middleware.NewAppError(http.StatusForbidden, "アクセスが拒否されました", err)
	default:
		return middleware.NewAppError(http.StatusInternalServerError, "サーバー内部エラーが発生しました", err)
	}
}
