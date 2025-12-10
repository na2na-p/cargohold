package url_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/infrastructure/url"
	"github.com/na2na-p/cargohold/internal/usecase"
)

var _ usecase.ActionURLGenerator = (*url.ProxyActionURLGenerator)(nil)

func TestProxyActionURLGenerator_GenerateUploadURL(t *testing.T) {
	type args struct {
		baseURL string
		owner   string
		repo    string
		oid     string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "正常系: 標準的なbaseURL",
			args: args{
				baseURL: "https://example.com",
				owner:   "testowner",
				repo:    "testrepo",
				oid:     "abc123def456",
			},
			want: "https://example.com/testowner/testrepo/info/lfs/objects/abc123def456",
		},
		{
			name: "正常系: baseURLが末尾スラッシュあり",
			args: args{
				baseURL: "https://example.com/",
				owner:   "testowner",
				repo:    "testrepo",
				oid:     "abc123def456",
			},
			want: "https://example.com/testowner/testrepo/info/lfs/objects/abc123def456",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := url.NewProxyActionURLGenerator()
			got := g.GenerateUploadURL(tt.args.baseURL, tt.args.owner, tt.args.repo, tt.args.oid)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProxyActionURLGenerator_GenerateDownloadURL(t *testing.T) {
	type args struct {
		baseURL string
		owner   string
		repo    string
		oid     string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "正常系: 標準的なbaseURL",
			args: args{
				baseURL: "https://example.com",
				owner:   "testowner",
				repo:    "testrepo",
				oid:     "abc123def456",
			},
			want: "https://example.com/testowner/testrepo/info/lfs/objects/abc123def456",
		},
		{
			name: "正常系: baseURLが末尾スラッシュあり",
			args: args{
				baseURL: "https://example.com/",
				owner:   "testowner",
				repo:    "testrepo",
				oid:     "abc123def456",
			},
			want: "https://example.com/testowner/testrepo/info/lfs/objects/abc123def456",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := url.NewProxyActionURLGenerator()
			got := g.GenerateDownloadURL(tt.args.baseURL, tt.args.owner, tt.args.repo, tt.args.oid)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
