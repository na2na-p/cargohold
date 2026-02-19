package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/labstack/echo/v5"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/handler"
	"github.com/na2na-p/cargohold/internal/handler/middleware"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_usecase "github.com/na2na-p/cargohold/tests/usecase"
	"go.uber.org/mock/gomock"
)

func TestBatchHandler_Handle(t *testing.T) {
	type fields struct {
		setupMock func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface
	}
	type args struct {
		method   string
		path     string
		owner    string
		repo     string
		body     interface{}
		headers  map[string]string
		userInfo *domain.UserInfo
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
		wantBody       interface{}
	}{
		{
			name: "正常系: uploadオペレーションが成功する",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
					m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ interface{}, _, _, _ string, _ usecase.BatchRequest, _ string) (usecase.BatchResponse, error) {
							uploadAction := usecase.NewAction("https://s3.example.com/upload", nil, 900)
							actions := usecase.NewActions(&uploadAction, nil)
							return usecase.NewBatchResponse(
								"basic",
								[]usecase.ResponseObject{
									usecase.NewResponseObject("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", 123456, true, &actions, nil),
								},
								"sha256",
							), nil
						},
					)
					return m
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
					"transfers": []string{"basic"},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, true, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusOK,
			wantBody: map[string]interface{}{
				"transfer": "basic",
				"objects": []interface{}{
					map[string]interface{}{
						"oid":           "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
						"size":          float64(123456),
						"authenticated": true,
						"actions": map[string]interface{}{
							"upload": map[string]interface{}{
								"href":       "https://s3.example.com/upload",
								"expires_in": float64(900),
							},
						},
					},
				},
				"hash_algo": "sha256",
			},
		},
		{
			name: "正常系: downloadオペレーションが成功する",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
					m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ interface{}, _, _, _ string, _ usecase.BatchRequest, _ string) (usecase.BatchResponse, error) {
							downloadAction := usecase.NewAction("https://s3.example.com/download", nil, 900)
							actions := usecase.NewActions(nil, &downloadAction)
							return usecase.NewBatchResponse(
								"basic",
								[]usecase.ResponseObject{
									usecase.NewResponseObject("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", 123456, true, &actions, nil),
								},
								"sha256",
							), nil
						},
					)
					return m
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "download",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
					"transfers": []string{"basic"},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, false, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusOK,
			wantBody: map[string]interface{}{
				"transfer": "basic",
				"objects": []interface{}{
					map[string]interface{}{
						"oid":           "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
						"size":          float64(123456),
						"authenticated": true,
						"actions": map[string]interface{}{
							"download": map[string]interface{}{
								"href":       "https://s3.example.com/download",
								"expires_in": float64(900),
							},
						},
					},
				},
				"hash_algo": "sha256",
			},
		},
		{
			name: "異常系: Acceptヘッダーが不正な場合、400エラーが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					return mock_usecase.NewMockBatchUseCaseInterface(ctrl)
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
				},
				headers: map[string]string{
					"Accept":       "application/json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody: map[string]interface{}{
				"message": "accept ヘッダーは application/vnd.git-lfs+json である必要があります",
			},
		},
		{
			name: "異常系: Content-Typeヘッダーが不正な場合、400エラーが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					return mock_usecase.NewMockBatchUseCaseInterface(ctrl)
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/json",
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody: map[string]interface{}{
				"message": "Content-Typeは application/vnd.git-lfs+json である必要があります",
			},
		},
		{
			name: "異常系: リクエストボディのパースに失敗した場合、422エラーが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					return mock_usecase.NewMockBatchUseCaseInterface(ctrl)
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body:   "invalid json",
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
			},
			wantStatusCode: http.StatusUnprocessableEntity,
			wantBody: map[string]interface{}{
				"message": "リクエストボディのパースに失敗しました",
			},
		},
		{
			name: "異常系: UseCaseが不正なオペレーションエラーを返した場合、422エラーが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
					m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
						usecase.NewBatchResponse("", []usecase.ResponseObject{}, ""), usecase.ErrInvalidOperation,
					)
					return m
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "download",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, false, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusUnprocessableEntity,
			wantBody: map[string]interface{}{
				"message": "不正なオペレーションです",
			},
		},
		{
			name: "異常系: UseCaseが空のオブジェクトエラーを返した場合、422エラーが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
					m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
						usecase.NewBatchResponse("", []usecase.ResponseObject{}, ""), usecase.ErrNoObjects,
					)
					return m
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "upload",
					"objects":   []map[string]interface{}{},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, true, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusUnprocessableEntity,
			wantBody: map[string]interface{}{
				"message": "オブジェクトが指定されていません",
			},
		},
		{
			name: "異常系: UseCaseが不正なOIDエラーを返した場合、422エラーが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
					m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
						usecase.NewBatchResponse("", []usecase.ResponseObject{}, ""), usecase.ErrInvalidOID,
					)
					return m
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  "invalid-oid",
							"size": 123456,
						},
					},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, true, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusUnprocessableEntity,
			wantBody: map[string]interface{}{
				"message": "不正なOIDです",
			},
		},
		{
			name: "異常系: UseCaseが不正なサイズエラーを返した場合、422エラーが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
					m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
						usecase.NewBatchResponse("", []usecase.ResponseObject{}, ""), usecase.ErrInvalidSize,
					)
					return m
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": -1,
						},
					},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, true, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusUnprocessableEntity,
			wantBody: map[string]interface{}{
				"message": "不正なサイズです",
			},
		},
		{
			name: "異常系: UseCaseが不正なハッシュアルゴリズムエラーを返した場合、422エラーが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
					m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
						usecase.NewBatchResponse("", []usecase.ResponseObject{}, ""), usecase.ErrInvalidHashAlgorithm,
					)
					return m
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "upload",
					"hash_algo": "sha512",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, true, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusUnprocessableEntity,
			wantBody: map[string]interface{}{
				"message": "不正なハッシュアルゴリズムです",
			},
		},
		{
			name: "異常系: UseCaseが未知のエラーを返した場合、500エラーが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
					m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
						usecase.NewBatchResponse("", []usecase.ResponseObject{}, ""), errors.New("unknown error"),
					)
					return m
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, true, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBody: map[string]interface{}{
				"message": "サーバー内部エラーが発生しました",
			},
		},
		{
			name: "正常系: Per-objectエラーを含むレスポンスが正しく返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
					m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ interface{}, _, _, _ string, _ usecase.BatchRequest, _ string) (usecase.BatchResponse, error) {
							objErr := usecase.NewObjectError(404, "オブジェクトが存在しません")
							return usecase.NewBatchResponse(
								"basic",
								[]usecase.ResponseObject{
									usecase.NewResponseObject("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", 123456, false, nil, &objErr),
								},
								"sha256",
							), nil
						},
					)
					return m
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "download",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
					"transfers": []string{"basic"},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, false, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusOK,
			wantBody: map[string]interface{}{
				"transfer": "basic",
				"objects": []interface{}{
					map[string]interface{}{
						"oid":           "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
						"size":          float64(123456),
						"authenticated": false,
						"error": map[string]interface{}{
							"code":    float64(404),
							"message": "オブジェクトが存在しません",
						},
					},
				},
				"hash_algo": "sha256",
			},
		},
		{
			name: "異常系: uploadオペレーションで権限不足の場合、403エラーが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					return mock_usecase.NewMockBatchUseCaseInterface(ctrl)
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, false, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusForbidden,
			wantBody: map[string]interface{}{
				"message": "このオペレーションを実行する権限がありません",
			},
		},
		{
			name: "異常系: downloadオペレーションで権限不足の場合、403エラーが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					return mock_usecase.NewMockBatchUseCaseInterface(ctrl)
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "download",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, false, false, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusForbidden,
			wantBody: map[string]interface{}{
				"message": "このオペレーションを実行する権限がありません",
			},
		},
		{
			name: "正常系: uploadオペレーションでpush権限がある場合、成功する",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
					m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ interface{}, _, _, _ string, _ usecase.BatchRequest, _ string) (usecase.BatchResponse, error) {
							uploadAction := usecase.NewAction("https://s3.example.com/upload", nil, 900)
							actions := usecase.NewActions(&uploadAction, nil)
							return usecase.NewBatchResponse(
								"basic",
								[]usecase.ResponseObject{
									usecase.NewResponseObject("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", 123456, true, &actions, nil),
								},
								"sha256",
							), nil
						},
					)
					return m
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
					"transfers": []string{"basic"},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, true, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusOK,
			wantBody: map[string]interface{}{
				"transfer": "basic",
				"objects": []interface{}{
					map[string]interface{}{
						"oid":           "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
						"size":          float64(123456),
						"authenticated": true,
						"actions": map[string]interface{}{
							"upload": map[string]interface{}{
								"href":       "https://s3.example.com/upload",
								"expires_in": float64(900),
							},
						},
					},
				},
				"hash_algo": "sha256",
			},
		},
		{
			name: "正常系: downloadオペレーションでpull権限がある場合、成功する",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
					m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ interface{}, _, _, _ string, _ usecase.BatchRequest, _ string) (usecase.BatchResponse, error) {
							downloadAction := usecase.NewAction("https://s3.example.com/download", nil, 900)
							actions := usecase.NewActions(nil, &downloadAction)
							return usecase.NewBatchResponse(
								"basic",
								[]usecase.ResponseObject{
									usecase.NewResponseObject("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", 123456, true, &actions, nil),
								},
								"sha256",
							), nil
						},
					)
					return m
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "download",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
					"transfers": []string{"basic"},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, false, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusOK,
			wantBody: map[string]interface{}{
				"transfer": "basic",
				"objects": []interface{}{
					map[string]interface{}{
						"oid":           "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
						"size":          float64(123456),
						"authenticated": true,
						"actions": map[string]interface{}{
							"download": map[string]interface{}{
								"href":       "https://s3.example.com/download",
								"expires_in": float64(900),
							},
						},
					},
				},
				"hash_algo": "sha256",
			},
		},
		{
			name: "異常系: UserInfoがコンテキストに設定されていない場合、403エラーが返る",
			fields: fields{
				setupMock: func(ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					return mock_usecase.NewMockBatchUseCaseInterface(ctrl)
				},
			},
			args: args{
				method: http.MethodPost,
				path:   "/testowner/testrepo/info/lfs/objects/batch",
				owner:  "testowner",
				repo:   "testrepo",
				body: map[string]interface{}{
					"operation": "upload",
					"objects": []map[string]interface{}{
						{
							"oid":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							"size": 123456,
						},
					},
				},
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: nil,
			},
			wantStatusCode: http.StatusForbidden,
			wantBody: map[string]interface{}{
				"message": "認証情報が見つかりません",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler
			var reqBody []byte
			if bodyStr, ok := tt.args.body.(string); ok {
				reqBody = []byte(bodyStr)
			} else {
				reqBody, _ = json.Marshal(tt.args.body)
			}
			req := httptest.NewRequest(tt.args.method, tt.args.path, bytes.NewReader(reqBody))
			for k, v := range tt.args.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{
				{Name: "owner", Value: tt.args.owner},
				{Name: "repo", Value: tt.args.repo},
			})

			if tt.args.userInfo != nil {
				c.Set(middleware.UserInfoContextKey, tt.args.userInfo)
			}

			mockUseCase := tt.fields.setupMock(ctrl)
			h := handler.NewBatchHandler(mockUseCase)
			err := h.Handle(c)
			if err != nil {
				e.HTTPErrorHandler(c, err)
			}

			if rec.Code != tt.wantStatusCode {
				t.Errorf("Handle() status code = %v, want %v", rec.Code, tt.wantStatusCode)
			}

			if tt.wantBody != nil {
				var gotBody map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &gotBody); err != nil {
					t.Fatalf("Failed to unmarshal response body: %v", err)
				}

				if diff := cmp.Diff(tt.wantBody, gotBody); diff != "" {
					t.Errorf("Handle() body mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestBatchHandler_Integration(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(t *testing.T, ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface
		requestBody    string
		setupHeaders   func(req *http.Request)
		userInfo       *domain.UserInfo
		wantStatusCode int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "統合: 正常なuploadリクエストが成功する",
			setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
				m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
				m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ interface{}, _, _, _ string, req usecase.BatchRequest, _ string) (usecase.BatchResponse, error) {
						if req.Operation().String() != "upload" {
							t.Errorf("Expected operation 'upload', got '%s'", req.Operation().String())
						}
						if len(req.Objects()) != 1 {
							t.Errorf("Expected 1 object, got %d", len(req.Objects()))
						}
						uploadAction := usecase.NewAction("https://s3.example.com/upload", nil, 900)
						actions := usecase.NewActions(&uploadAction, nil)
						return usecase.NewBatchResponse(
							"basic",
							[]usecase.ResponseObject{
								usecase.NewResponseObject(req.Objects()[0].OID(), req.Objects()[0].Size(), true, &actions, nil),
							},
							"sha256",
						), nil
					},
				)
				return m
			},
			requestBody: `{
				"operation": "upload",
				"objects": [
					{
						"oid": "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
						"size": 123456
					}
				],
				"transfers": ["basic"]
			}`,
			setupHeaders: func(req *http.Request) {
				req.Header.Set(echo.HeaderContentType, handler.GitLFSContentType)
				req.Header.Set(echo.HeaderAccept, handler.GitLFSContentType)
			},
			userInfo: func() *domain.UserInfo {
				perms := domain.NewRepositoryPermissions(false, true, true, false, false)
				repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
				ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
				ui.SetPermissions(&perms)
				return ui
			}(),
			wantStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp usecase.BatchResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp.Transfer() != "basic" {
					t.Errorf("Expected transfer 'basic', got '%s'", resp.Transfer())
				}
				if len(resp.Objects()) != 1 {
					t.Fatalf("Expected 1 object, got %d", len(resp.Objects()))
				}
				if resp.Objects()[0].Actions() == nil || resp.Objects()[0].Actions().Upload() == nil {
					t.Error("Expected upload action to be present")
				}
			},
		},
		{
			name: "統合: ヘッダー不正でエラーが返る",
			setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
				return mock_usecase.NewMockBatchUseCaseInterface(ctrl)
			},
			requestBody: `{"operation": "upload"}`,
			setupHeaders: func(req *http.Request) {
				req.Header.Set(echo.HeaderContentType, "application/json")
				req.Header.Set(echo.HeaderAccept, "application/json")
			},
			userInfo:       nil,
			wantStatusCode: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var errResp map[string]interface{}
				if err := json.Unmarshal(body, &errResp); err != nil {
					t.Fatalf("Failed to unmarshal error response: %v", err)
				}
				if _, ok := errResp["message"]; !ok {
					t.Error("Expected 'message' field in error response")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/testowner/testrepo/info/lfs/objects/batch", bytes.NewBufferString(tt.requestBody))
			if tt.setupHeaders != nil {
				tt.setupHeaders(req)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{
				{Name: "owner", Value: "testowner"},
				{Name: "repo", Value: "testrepo"},
			})

			if tt.userInfo != nil {
				c.Set(middleware.UserInfoContextKey, tt.userInfo)
			}

			mockUseCase := tt.setupMock(t, ctrl)
			h := handler.NewBatchHandler(mockUseCase)

			_ = h.Handle(c)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.wantStatusCode, rec.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.Bytes())
			}
		})
	}
}

func TestBatchHandler_Handle_PayloadTooLarge(t *testing.T) {
	type fields struct {
		setupMock func(t *testing.T, ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface
	}
	type args struct {
		body     io.Reader
		headers  map[string]string
		userInfo *domain.UserInfo
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
		wantBody       map[string]interface{}
	}{
		{
			name: "異常系: リクエストボディが10MBを超える場合、413エラーが返る",
			fields: fields{
				setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					return mock_usecase.NewMockBatchUseCaseInterface(ctrl)
				},
			},
			args: args{
				body: strings.NewReader(strings.Repeat("a", 11<<20)),
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: nil,
			},
			wantStatusCode: http.StatusRequestEntityTooLarge,
			wantBody: map[string]interface{}{
				"message": "リクエストボディが大きすぎます",
			},
		},
		{
			name: "正常系: リクエストボディが10MB以下の場合、処理が継続する",
			fields: fields{
				setupMock: func(t *testing.T, ctrl *gomock.Controller) *mock_usecase.MockBatchUseCaseInterface {
					m := mock_usecase.NewMockBatchUseCaseInterface(ctrl)
					m.EXPECT().HandleBatchRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
						usecase.NewBatchResponse("basic", []usecase.ResponseObject{}, "sha256"), nil,
					)
					return m
				},
			},
			args: args{
				body: bytes.NewReader([]byte(`{"operation": "upload", "objects": [{"oid": "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "size": 123456}]}`)),
				headers: map[string]string{
					"Accept":       "application/vnd.git-lfs+json",
					"Content-Type": "application/vnd.git-lfs+json",
				},
				userInfo: func() *domain.UserInfo {
					perms := domain.NewRepositoryPermissions(false, true, true, false, false)
					repoID, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
					ui, _ := domain.NewUserInfo("test-sub", "test@example.com", "Test User", domain.ProviderTypeGitHub, repoID, "refs/heads/main")
					ui.SetPermissions(&perms)
					return ui
				}(),
			},
			wantStatusCode: http.StatusOK,
			wantBody:       nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/testowner/testrepo/info/lfs/objects/batch", tt.args.body)
			for k, v := range tt.args.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{
				{Name: "owner", Value: "testowner"},
				{Name: "repo", Value: "testrepo"},
			})

			if tt.args.userInfo != nil {
				c.Set(middleware.UserInfoContextKey, tt.args.userInfo)
			}

			mockUseCase := tt.fields.setupMock(t, ctrl)
			h := handler.NewBatchHandler(mockUseCase)
			err := h.Handle(c)
			if err != nil {
				e.HTTPErrorHandler(c, err)
			}

			if rec.Code != tt.wantStatusCode {
				t.Errorf("Handle() status code = %v, want %v", rec.Code, tt.wantStatusCode)
			}

			if tt.wantBody != nil {
				var gotBody map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &gotBody); err != nil {
					t.Fatalf("Failed to unmarshal response body: %v", err)
				}

				if diff := cmp.Diff(tt.wantBody, gotBody); diff != "" {
					t.Errorf("Handle() body mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
