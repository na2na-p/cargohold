package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/handler"
	"github.com/na2na-p/cargohold/internal/usecase"
	mock_handler "github.com/na2na-p/cargohold/tests/handler"
	"go.uber.org/mock/gomock"
)

func TestVerifyHandler(t *testing.T) {
	type fields struct {
		usecase func(ctrl *gomock.Controller) handler.VerifyUseCase
	}
	type args struct {
		body        interface{}
		contentType string
		accept      string
		owner       string
		repo        string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
		wantBodyJSON   map[string]interface{}
	}{
		{
			name: "正常系: アップロード完了通知が成功する",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) handler.VerifyUseCase {
					mock := mock_handler.NewMockVerifyUseCase(ctrl)
					mock.EXPECT().VerifyUpload(gomock.Any(), "abc123def4567890abc123def4567890abc123def4567890abc123def4567890", int64(1024)).Return(nil).Times(1)
					return mock
				},
			},
			args: args{
				body: map[string]interface{}{
					"oid":  "abc123def4567890abc123def4567890abc123def4567890abc123def4567890",
					"size": 1024,
				},
				contentType: "application/vnd.git-lfs+json",
				accept:      "application/vnd.git-lfs+json",
				owner:       "testowner",
				repo:        "testrepo",
			},
			wantStatusCode: http.StatusOK,
			wantBodyJSON: map[string]interface{}{
				"message": "success",
			},
		},
		{
			name: "異常系: Content-Typeヘッダーが不正",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) handler.VerifyUseCase {
					mock := mock_handler.NewMockVerifyUseCase(ctrl)
					return mock
				},
			},
			args: args{
				body: map[string]interface{}{
					"oid":  "abc123def4567890abc123def4567890abc123def4567890abc123def4567890",
					"size": 1024,
				},
				contentType: "application/json",
				accept:      "application/vnd.git-lfs+json",
				owner:       "testowner",
				repo:        "testrepo",
			},
			wantStatusCode: http.StatusBadRequest,
			wantBodyJSON: map[string]interface{}{
				"message": "Content-Typeは application/vnd.git-lfs+json である必要があります",
			},
		},
		{
			name: "異常系: Acceptヘッダーが不正",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) handler.VerifyUseCase {
					mock := mock_handler.NewMockVerifyUseCase(ctrl)
					return mock
				},
			},
			args: args{
				body: map[string]interface{}{
					"oid":  "abc123def4567890abc123def4567890abc123def4567890abc123def4567890",
					"size": 1024,
				},
				contentType: "application/vnd.git-lfs+json",
				accept:      "application/json",
				owner:       "testowner",
				repo:        "testrepo",
			},
			wantStatusCode: http.StatusBadRequest,
			wantBodyJSON: map[string]interface{}{
				"message": "accept ヘッダーは application/vnd.git-lfs+json である必要があります",
			},
		},
		{
			name: "異常系: リクエストボディが不正なJSON",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) handler.VerifyUseCase {
					mock := mock_handler.NewMockVerifyUseCase(ctrl)
					return mock
				},
			},
			args: args{
				body:        "{invalid json}",
				contentType: "application/vnd.git-lfs+json",
				accept:      "application/vnd.git-lfs+json",
				owner:       "testowner",
				repo:        "testrepo",
			},
			wantStatusCode: http.StatusBadRequest,
			wantBodyJSON: map[string]interface{}{
				"message": "リクエストボディの解析に失敗しました",
			},
		},
		{
			name: "異常系: oidフィールドが空",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) handler.VerifyUseCase {
					mock := mock_handler.NewMockVerifyUseCase(ctrl)
					return mock
				},
			},
			args: args{
				body: map[string]interface{}{
					"oid":  "",
					"size": 1024,
				},
				contentType: "application/vnd.git-lfs+json",
				accept:      "application/vnd.git-lfs+json",
				owner:       "testowner",
				repo:        "testrepo",
			},
			wantStatusCode: http.StatusBadRequest,
			wantBodyJSON: map[string]interface{}{
				"message": "oidフィールドは必須です",
			},
		},
		{
			name: "異常系: sizeフィールドが負の値",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) handler.VerifyUseCase {
					mock := mock_handler.NewMockVerifyUseCase(ctrl)
					return mock
				},
			},
			args: args{
				body: map[string]interface{}{
					"oid":  "abc123def4567890abc123def4567890abc123def4567890abc123def4567890",
					"size": -1,
				},
				contentType: "application/vnd.git-lfs+json",
				accept:      "application/vnd.git-lfs+json",
				owner:       "testowner",
				repo:        "testrepo",
			},
			wantStatusCode: http.StatusBadRequest,
			wantBodyJSON: map[string]interface{}{
				"message": "sizeフィールドは正の整数である必要があります",
			},
		},
		{
			name: "異常系: OID形式が不正",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) handler.VerifyUseCase {
					mock := mock_handler.NewMockVerifyUseCase(ctrl)
					return mock
				},
			},
			args: args{
				body: map[string]interface{}{
					"oid":  "invalid",
					"size": 1024,
				},
				contentType: "application/vnd.git-lfs+json",
				accept:      "application/vnd.git-lfs+json",
				owner:       "testowner",
				repo:        "testrepo",
			},
			wantStatusCode: http.StatusBadRequest,
			wantBodyJSON: map[string]interface{}{
				"message": "oidフィールドは必須です",
			},
		},
		{
			name: "異常系: オブジェクトが見つからない",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) handler.VerifyUseCase {
					mock := mock_handler.NewMockVerifyUseCase(ctrl)
					mock.EXPECT().VerifyUpload(gomock.Any(), "abc123def4567890abc123def4567890abc123def4567890abc123def4567890", int64(1024)).Return(usecase.ErrObjectNotFound).Times(1)
					return mock
				},
			},
			args: args{
				body: map[string]interface{}{
					"oid":  "abc123def4567890abc123def4567890abc123def4567890abc123def4567890",
					"size": 1024,
				},
				contentType: "application/vnd.git-lfs+json",
				accept:      "application/vnd.git-lfs+json",
				owner:       "testowner",
				repo:        "testrepo",
			},
			wantStatusCode: http.StatusNotFound,
			wantBodyJSON: map[string]interface{}{
				"message": "オブジェクトが見つかりません",
			},
		},
		{
			name: "異常系: サイズが一致しない",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) handler.VerifyUseCase {
					mock := mock_handler.NewMockVerifyUseCase(ctrl)
					mock.EXPECT().VerifyUpload(gomock.Any(), "abc123def4567890abc123def4567890abc123def4567890abc123def4567890", int64(1024)).Return(usecase.ErrSizeMismatch).Times(1)
					return mock
				},
			},
			args: args{
				body: map[string]interface{}{
					"oid":  "abc123def4567890abc123def4567890abc123def4567890abc123def4567890",
					"size": 1024,
				},
				contentType: "application/vnd.git-lfs+json",
				accept:      "application/vnd.git-lfs+json",
				owner:       "testowner",
				repo:        "testrepo",
			},
			wantStatusCode: http.StatusUnprocessableEntity,
			wantBodyJSON: map[string]interface{}{
				"message": "サイズが一致しません",
			},
		},
		{
			name: "異常系: サーバー内部エラー",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) handler.VerifyUseCase {
					mock := mock_handler.NewMockVerifyUseCase(ctrl)
					mock.EXPECT().VerifyUpload(gomock.Any(), "abc123def4567890abc123def4567890abc123def4567890abc123def4567890", int64(1024)).Return(errors.New("internal error")).Times(1)
					return mock
				},
			},
			args: args{
				body: map[string]interface{}{
					"oid":  "abc123def4567890abc123def4567890abc123def4567890abc123def4567890",
					"size": 1024,
				},
				contentType: "application/vnd.git-lfs+json",
				accept:      "application/vnd.git-lfs+json",
				owner:       "testowner",
				repo:        "testrepo",
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBodyJSON: map[string]interface{}{
				"message": "サーバー内部エラーが発生しました",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			e := echo.New()

			var reqBody *bytes.Buffer
			if str, ok := tt.args.body.(string); ok {
				reqBody = bytes.NewBufferString(str)
			} else {
				bodyBytes, _ := json.Marshal(tt.args.body)
				reqBody = bytes.NewBuffer(bodyBytes)
			}

			req := httptest.NewRequest(http.MethodPost, "/"+tt.args.owner+"/"+tt.args.repo+"/info/lfs/objects/verify", reqBody)
			req.Header.Set("Content-Type", tt.args.contentType)
			req.Header.Set("Accept", tt.args.accept)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("owner", "repo")
			c.SetParamValues(tt.args.owner, tt.args.repo)

			h := handler.VerifyHandler(tt.fields.usecase(ctrl))

			_ = h(c)

			if diff := cmp.Diff(tt.wantStatusCode, rec.Code); diff != "" {
				t.Errorf("ステータスコードが一致しません (-want +got):\n%s", diff)
			}

			var gotBodyJSON map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &gotBodyJSON); err != nil {
				t.Fatalf("レスポンスボディのパースに失敗しました: %v, body: %s", err, rec.Body.String())
			}

			if diff := cmp.Diff(tt.wantBodyJSON, gotBodyJSON); diff != "" {
				t.Errorf("レスポンスボディが一致しません (-want +got):\n%s", diff)
			}
		})
	}
}
