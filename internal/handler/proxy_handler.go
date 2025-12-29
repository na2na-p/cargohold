package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/usecase"
)

type ProxyHandler struct {
	proxyUploadUseCase   usecase.ProxyUploadUseCase
	proxyDownloadUseCase usecase.ProxyDownloadUseCase
	storageErrorChecker  usecase.StorageErrorChecker
	proxyTimeout         time.Duration
}

func NewProxyHandler(uploadUC usecase.ProxyUploadUseCase, downloadUC usecase.ProxyDownloadUseCase, storageErrorChecker usecase.StorageErrorChecker, proxyTimeout time.Duration) *ProxyHandler {
	return &ProxyHandler{
		proxyUploadUseCase:   uploadUC,
		proxyDownloadUseCase: downloadUC,
		storageErrorChecker:  storageErrorChecker,
		proxyTimeout:         proxyTimeout,
	}
}

func (h *ProxyHandler) HandleUpload(c echo.Context) error {
	owner := c.Param("owner")
	repo := c.Param("repo")
	oidStr := c.Param("oid")
	oid, err := domain.NewOID(oidStr)
	if err != nil {
		return SendLFSError(c, http.StatusUnprocessableEntity, "不正なOIDです")
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.proxyTimeout)
	defer cancel()

	if err := h.proxyUploadUseCase.Execute(ctx, owner, repo, oid, c.Request().Body); err != nil {
		return h.handleProxyError(c, err)
	}

	return c.NoContent(http.StatusOK)
}

func (h *ProxyHandler) HandleDownload(c echo.Context) error {
	owner := c.Param("owner")
	repo := c.Param("repo")
	oidStr := c.Param("oid")
	oid, err := domain.NewOID(oidStr)
	if err != nil {
		return SendLFSError(c, http.StatusUnprocessableEntity, "不正なOIDです")
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.proxyTimeout)
	defer cancel()

	stream, size, err := h.proxyDownloadUseCase.Execute(ctx, owner, repo, oid)
	if err != nil {
		return h.handleProxyError(c, err)
	}
	defer func() { _ = stream.Close() }()

	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(size, 10))
	c.Response().Header().Set(echo.HeaderContentType, "application/octet-stream")

	return c.Stream(http.StatusOK, "application/octet-stream", stream)
}

func (h *ProxyHandler) handleProxyError(c echo.Context, err error) error {
	if errors.Is(err, usecase.ErrAccessDenied) {
		return SendLFSError(c, http.StatusForbidden, "アクセスが拒否されました")
	}

	if errors.Is(err, usecase.ErrObjectNotFound) {
		return SendLFSError(c, http.StatusNotFound, "オブジェクトが存在しません")
	}

	if errors.Is(err, usecase.ErrNotUploaded) {
		return SendLFSError(c, http.StatusNotFound, "オブジェクトがまだアップロードされていません")
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return SendLFSError(c, http.StatusGatewayTimeout, "リクエストがタイムアウトしました")
	}

	if h.isStorageError(err) {
		return SendLFSError(c, http.StatusBadGateway, "ストレージサーバーでエラーが発生しました")
	}

	return SendLFSError(c, http.StatusInternalServerError, "サーバー内部エラーが発生しました")
}

func (h *ProxyHandler) isStorageError(err error) bool {
	return h.storageErrorChecker.IsStorageError(err)
}
