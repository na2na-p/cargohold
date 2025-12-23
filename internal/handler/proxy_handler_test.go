package handler_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/handler"
	"github.com/na2na-p/cargohold/internal/handler/middleware"
	"github.com/na2na-p/cargohold/internal/infrastructure/s3"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_usecase "github.com/na2na-p/cargohold/tests/usecase"
	"go.uber.org/mock/gomock"
)

func TestProxyHandler_HandleUpload(t *testing.T) {
	type fields struct {
		setupUploadMock   func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase
		setupDownloadMock func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase
		proxyTimeout      time.Duration
	}
	type args struct {
		method  string
		path    string
		owner   string
		repo    string
		oid     string
		body    string
		headers map[string]string
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		wantStatusCode   int
		wantBodyContains string
	}{
		{
			name: "正常系: アップロードが成功する",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					m := mock_usecase.NewMockProxyUploadUseCase(ctrl)
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
					return m
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					return mock_usecase.NewMockProxyDownloadUseCase(ctrl)
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodPut,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				body:   "test file content",
				headers: map[string]string{
					"Accept":       "application/octet-stream",
					"Content-Type": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "異常系: OIDが不正な場合、422エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					return mock_usecase.NewMockProxyUploadUseCase(ctrl)
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					return mock_usecase.NewMockProxyDownloadUseCase(ctrl)
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodPut,
				path:   "/testowner/testrepo/info/lfs/objects/invalid-oid",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "invalid-oid",
				body:   "test file content",
				headers: map[string]string{
					"Accept":       "application/octet-stream",
					"Content-Type": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name: "異常系: オブジェクトが存在しない場合、404エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					m := mock_usecase.NewMockProxyUploadUseCase(ctrl)
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.ErrObjectNotFound)
					return m
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					return mock_usecase.NewMockProxyDownloadUseCase(ctrl)
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodPut,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				body:   "test file content",
				headers: map[string]string{
					"Accept":       "application/octet-stream",
					"Content-Type": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "異常系: タイムアウトした場合、504エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					m := mock_usecase.NewMockProxyUploadUseCase(ctrl)
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(context.DeadlineExceeded)
					return m
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					return mock_usecase.NewMockProxyDownloadUseCase(ctrl)
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodPut,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				body:   "test file content",
				headers: map[string]string{
					"Accept":       "application/octet-stream",
					"Content-Type": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusGatewayTimeout,
		},
		{
			name: "異常系: ストレージエラーの場合、502エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					m := mock_usecase.NewMockProxyUploadUseCase(ctrl)
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(s3.NewStorageError(s3.OperationPut, errors.New("connection refused")))
					return m
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					return mock_usecase.NewMockProxyDownloadUseCase(ctrl)
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodPut,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				body:   "test file content",
				headers: map[string]string{
					"Accept":       "application/octet-stream",
					"Content-Type": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusBadGateway,
		},
		{
			name: "異常系: 未知のエラーの場合、500エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					m := mock_usecase.NewMockProxyUploadUseCase(ctrl)
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("unknown error"))
					return m
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					return mock_usecase.NewMockProxyDownloadUseCase(ctrl)
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodPut,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				body:   "test file content",
				headers: map[string]string{
					"Accept":       "application/octet-stream",
					"Content-Type": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "異常系: アクセス拒否の場合、403エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					m := mock_usecase.NewMockProxyUploadUseCase(ctrl)
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.ErrAccessDenied)
					return m
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					return mock_usecase.NewMockProxyDownloadUseCase(ctrl)
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodPut,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				body:   "test file content",
				headers: map[string]string{
					"Accept":       "application/octet-stream",
					"Content-Type": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusForbidden,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler
			req := httptest.NewRequest(tt.args.method, tt.args.path, strings.NewReader(tt.args.body))
			for k, v := range tt.args.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("owner", "repo", "oid")
			c.SetParamValues(tt.args.owner, tt.args.repo, tt.args.oid)

			mockUploadUC := tt.fields.setupUploadMock(ctrl)
			mockDownloadUC := tt.fields.setupDownloadMock(ctrl)
			h := handler.NewProxyHandler(mockUploadUC, mockDownloadUC, tt.fields.proxyTimeout)
			err := h.HandleUpload(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}

			if rec.Code != tt.wantStatusCode {
				t.Errorf("HandleUpload() status code = %v, want %v", rec.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestProxyHandler_HandleDownload(t *testing.T) {
	type fields struct {
		setupUploadMock   func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase
		setupDownloadMock func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase
		proxyTimeout      time.Duration
	}
	type args struct {
		method  string
		path    string
		owner   string
		repo    string
		oid     string
		headers map[string]string
	}
	tests := []struct {
		name              string
		fields            fields
		args              args
		wantStatusCode    int
		wantBody          string
		wantContentLength string
	}{
		{
			name: "正常系: ダウンロードが成功する",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					return mock_usecase.NewMockProxyUploadUseCase(ctrl)
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					m := mock_usecase.NewMockProxyDownloadUseCase(ctrl)
					content := "test file content"
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
						io.NopCloser(strings.NewReader(content)),
						int64(len(content)),
						nil,
					)
					return m
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodGet,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				headers: map[string]string{
					"Accept": "application/octet-stream",
				},
			},
			wantStatusCode:    http.StatusOK,
			wantBody:          "test file content",
			wantContentLength: "17",
		},
		{
			name: "異常系: OIDが不正な場合、422エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					return mock_usecase.NewMockProxyUploadUseCase(ctrl)
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					return mock_usecase.NewMockProxyDownloadUseCase(ctrl)
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodGet,
				path:   "/testowner/testrepo/info/lfs/objects/invalid-oid",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "invalid-oid",
				headers: map[string]string{
					"Accept": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name: "異常系: オブジェクトが存在しない場合、404エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					return mock_usecase.NewMockProxyUploadUseCase(ctrl)
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					m := mock_usecase.NewMockProxyDownloadUseCase(ctrl)
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, int64(0), usecase.ErrObjectNotFound)
					return m
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodGet,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				headers: map[string]string{
					"Accept": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "異常系: オブジェクトがまだアップロードされていない場合、404エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					return mock_usecase.NewMockProxyUploadUseCase(ctrl)
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					m := mock_usecase.NewMockProxyDownloadUseCase(ctrl)
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, int64(0), usecase.ErrNotUploaded)
					return m
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodGet,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				headers: map[string]string{
					"Accept": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "異常系: タイムアウトした場合、504エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					return mock_usecase.NewMockProxyUploadUseCase(ctrl)
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					m := mock_usecase.NewMockProxyDownloadUseCase(ctrl)
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, int64(0), context.DeadlineExceeded)
					return m
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodGet,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				headers: map[string]string{
					"Accept": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusGatewayTimeout,
		},
		{
			name: "異常系: ストレージエラーの場合、502エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					return mock_usecase.NewMockProxyUploadUseCase(ctrl)
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					m := mock_usecase.NewMockProxyDownloadUseCase(ctrl)
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, int64(0), s3.NewStorageError(s3.OperationGet, errors.New("connection refused")))
					return m
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodGet,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				headers: map[string]string{
					"Accept": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusBadGateway,
		},
		{
			name: "異常系: 未知のエラーの場合、500エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					return mock_usecase.NewMockProxyUploadUseCase(ctrl)
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					m := mock_usecase.NewMockProxyDownloadUseCase(ctrl)
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, int64(0), errors.New("unknown error"))
					return m
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodGet,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				headers: map[string]string{
					"Accept": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "異常系: アクセス拒否の場合、403エラーが返る",
			fields: fields{
				setupUploadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyUploadUseCase {
					return mock_usecase.NewMockProxyUploadUseCase(ctrl)
				},
				setupDownloadMock: func(ctrl *gomock.Controller) *mock_usecase.MockProxyDownloadUseCase {
					m := mock_usecase.NewMockProxyDownloadUseCase(ctrl)
					m.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, int64(0), usecase.ErrAccessDenied)
					return m
				},
				proxyTimeout: 10 * time.Minute,
			},
			args: args{
				method: http.MethodGet,
				path:   "/testowner/testrepo/info/lfs/objects/abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				owner:  "testowner",
				repo:   "testrepo",
				oid:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				headers: map[string]string{
					"Accept": "application/octet-stream",
				},
			},
			wantStatusCode: http.StatusForbidden,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler
			req := httptest.NewRequest(tt.args.method, tt.args.path, nil)
			for k, v := range tt.args.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("owner", "repo", "oid")
			c.SetParamValues(tt.args.owner, tt.args.repo, tt.args.oid)

			mockUploadUC := tt.fields.setupUploadMock(ctrl)
			mockDownloadUC := tt.fields.setupDownloadMock(ctrl)
			h := handler.NewProxyHandler(mockUploadUC, mockDownloadUC, tt.fields.proxyTimeout)
			err := h.HandleDownload(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}

			if rec.Code != tt.wantStatusCode {
				t.Errorf("HandleDownload() status code = %v, want %v", rec.Code, tt.wantStatusCode)
			}

			if tt.wantBody != "" {
				if diff := cmp.Diff(tt.wantBody, rec.Body.String()); diff != "" {
					t.Errorf("HandleDownload() body mismatch (-want +got):\n%s", diff)
				}
			}

			if tt.wantContentLength != "" {
				got := rec.Header().Get("Content-Length")
				if diff := cmp.Diff(tt.wantContentLength, got); diff != "" {
					t.Errorf("HandleDownload() Content-Length mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestNewProxyHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUploadUC := mock_usecase.NewMockProxyUploadUseCase(ctrl)
	mockDownloadUC := mock_usecase.NewMockProxyDownloadUseCase(ctrl)
	timeout := 5 * time.Minute

	h := handler.NewProxyHandler(mockUploadUC, mockDownloadUC, timeout)

	if h == nil {
		t.Fatal("NewProxyHandler() returned nil")
	}
}

var _ = bytes.Buffer{}
var _ = domain.OID{}
