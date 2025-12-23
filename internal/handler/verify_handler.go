package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/usecase"
)

type VerifyResponse struct {
	Message string `json:"message"`
}

func VerifyHandler(uc usecase.VerifyUseCaseInterface) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		if err := ValidateLFSHeaders(c); err != nil {
			return SendLFSError(c, http.StatusBadRequest, err.Error())
		}

		_, err := ExtractRepositoryIdentifier(c)
		if err != nil {
			return SendLFSError(c, http.StatusBadRequest, "リポジトリ識別子の形式が不正です")
		}

		var req usecase.VerifyRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return SendLFSError(c, http.StatusBadRequest, "リクエストボディの解析に失敗しました")
		}

		if err := req.Validate(); err != nil {
			switch {
			case errors.Is(err, usecase.ErrInvalidOID):
				return SendLFSError(c, http.StatusBadRequest, "oidフィールドは必須です")
			case errors.Is(err, usecase.ErrInvalidSize):
				return SendLFSError(c, http.StatusBadRequest, "sizeフィールドは正の整数である必要があります")
			default:
				return SendLFSError(c, http.StatusBadRequest, err.Error())
			}
		}

		if err := uc.VerifyUpload(ctx, req.OID, req.Size); err != nil {
			switch {
			case errors.Is(err, usecase.ErrInvalidOID):
				return SendLFSError(c, http.StatusBadRequest, "OID形式が不正です")
			case errors.Is(err, usecase.ErrObjectNotFound):
				return SendLFSError(c, http.StatusNotFound, "オブジェクトが見つかりません")
			case errors.Is(err, usecase.ErrSizeMismatch):
				return SendLFSError(c, http.StatusUnprocessableEntity, "サイズが一致しません")
			default:
				return SendLFSError(c, http.StatusInternalServerError, "サーバー内部エラーが発生しました")
			}
		}

		return c.JSON(http.StatusOK, VerifyResponse{
			Message: "success",
		})
	}
}
