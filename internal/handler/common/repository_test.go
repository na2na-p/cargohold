package common_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/labstack/echo/v5"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/handler/common"
)

func TestExtractRepositoryIdentifier(t *testing.T) {
	type args struct {
		owner string
		repo  string
	}
	tests := []struct {
		name    string
		args    args
		want    *domain.RepositoryIdentifier
		wantErr error
	}{
		{
			name: "正常系: owner/repo形式のリポジトリ識別子が正しく抽出される",
			args: args{
				owner: "testowner",
				repo:  "testrepo",
			},
			want: func() *domain.RepositoryIdentifier {
				ri, _ := domain.NewRepositoryIdentifier("testowner/testrepo")
				return ri
			}(),
			wantErr: nil,
		},
		{
			name: "正常系: ハイフンを含むowner/repoが正しく抽出される",
			args: args{
				owner: "my-org",
				repo:  "my-repo-name",
			},
			want: func() *domain.RepositoryIdentifier {
				ri, _ := domain.NewRepositoryIdentifier("my-org/my-repo-name")
				return ri
			}(),
			wantErr: nil,
		},
		{
			name: "異常系: ownerが空の場合、エラーが返る",
			args: args{
				owner: "",
				repo:  "testrepo",
			},
			want:    nil,
			wantErr: domain.ErrInvalidRepositoryIdentifierFormat,
		},
		{
			name: "異常系: repoが空の場合、エラーが返る",
			args: args{
				owner: "testowner",
				repo:  "",
			},
			want:    nil,
			wantErr: domain.ErrInvalidRepositoryIdentifierFormat,
		},
		{
			name: "異常系: owner/repoの両方が空の場合、エラーが返る",
			args: args{
				owner: "",
				repo:  "",
			},
			want:    nil,
			wantErr: domain.ErrInvalidRepositoryIdentifierFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/"+tt.args.owner+"/"+tt.args.repo+"/info/lfs/objects/batch", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{
				{Name: "owner", Value: tt.args.owner},
				{Name: "repo", Value: tt.args.repo},
			})

			got, err := common.ExtractRepositoryIdentifier(c)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if err != tt.wantErr {
					t.Errorf("want error %v, but got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("want no error, but got %v", err)
			}

			if diff := cmp.Diff(tt.want.FullName(), got.FullName()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
