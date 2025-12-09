package domain_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewAllowedRepository(t *testing.T) {
	type args struct {
		owner string
		repo  string
	}
	tests := []struct {
		name    string
		args    args
		want    *domain.AllowedRepository
		wantErr error
	}{
		{
			name: "正常系: 有効なowner/repoでAllowedRepositoryが作成される",
			args: args{
				owner: "na2na-p",
				repo:  "cargohold",
			},
			want:    mustNewAllowedRepository(t, "na2na-p", "cargohold"),
			wantErr: nil,
		},
		{
			name: "異常系: ownerが空の場合エラーが返る",
			args: args{
				owner: "",
				repo:  "cargohold",
			},
			want:    nil,
			wantErr: domain.ErrInvalidAllowedRepositoryFormat,
		},
		{
			name: "異常系: repoが空の場合エラーが返る",
			args: args{
				owner: "na2na-p",
				repo:  "",
			},
			want:    nil,
			wantErr: domain.ErrInvalidAllowedRepositoryFormat,
		},
		{
			name: "異常系: ownerとrepoが両方空の場合エラーが返る",
			args: args{
				owner: "",
				repo:  "",
			},
			want:    nil,
			wantErr: domain.ErrInvalidAllowedRepositoryFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewAllowedRepository(tt.args.owner, tt.args.repo)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if err != tt.wantErr {
					t.Fatalf("want error %v, but got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("want no error, but got %v", err)
			}

			if diff := cmp.Diff(tt.want.Owner(), got.Owner()); diff != "" {
				t.Errorf("Owner mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Repo(), got.Repo()); diff != "" {
				t.Errorf("Repo mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewAllowedRepositoryFromString(t *testing.T) {
	type args struct {
		fullName string
	}
	tests := []struct {
		name    string
		args    args
		want    *domain.AllowedRepository
		wantErr error
	}{
		{
			name: "正常系: 有効なowner/repo形式でAllowedRepositoryが作成される",
			args: args{
				fullName: "na2na-p/cargohold",
			},
			want:    mustNewAllowedRepository(t, "na2na-p", "cargohold"),
			wantErr: nil,
		},
		{
			name: "異常系: スラッシュがない場合エラーが返る",
			args: args{
				fullName: "na2na-pcargohold",
			},
			want:    nil,
			wantErr: domain.ErrInvalidAllowedRepositoryFormat,
		},
		{
			name: "異常系: スラッシュが複数ある場合エラーが返る",
			args: args{
				fullName: "na2na-p/cargohold/extra",
			},
			want:    nil,
			wantErr: domain.ErrInvalidAllowedRepositoryFormat,
		},
		{
			name: "異常系: owner部分が空の場合エラーが返る",
			args: args{
				fullName: "/cargohold",
			},
			want:    nil,
			wantErr: domain.ErrInvalidAllowedRepositoryFormat,
		},
		{
			name: "異常系: repo部分が空の場合エラーが返る",
			args: args{
				fullName: "na2na-p/",
			},
			want:    nil,
			wantErr: domain.ErrInvalidAllowedRepositoryFormat,
		},
		{
			name: "異常系: 空文字列の場合エラーが返る",
			args: args{
				fullName: "",
			},
			want:    nil,
			wantErr: domain.ErrInvalidAllowedRepositoryFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewAllowedRepositoryFromString(tt.args.fullName)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if err != tt.wantErr {
					t.Fatalf("want error %v, but got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("want no error, but got %v", err)
			}

			if diff := cmp.Diff(tt.want.Owner(), got.Owner()); diff != "" {
				t.Errorf("Owner mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Repo(), got.Repo()); diff != "" {
				t.Errorf("Repo mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAllowedRepository_String(t *testing.T) {
	tests := []struct {
		name  string
		owner string
		repo  string
		want  string
	}{
		{
			name:  "正常系: owner/repo形式の文字列が返される",
			owner: "na2na-p",
			repo:  "cargohold",
			want:  "na2na-p/cargohold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ar := mustNewAllowedRepository(t, tt.owner, tt.repo)
			got := ar.String()

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("String mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAllowedRepository_Equals(t *testing.T) {
	tests := []struct {
		name  string
		ar    *domain.AllowedRepository
		other *domain.AllowedRepository
		want  bool
	}{
		{
			name:  "正常系: 同じowner/repoの場合trueが返る",
			ar:    mustNewAllowedRepository(t, "na2na-p", "cargohold"),
			other: mustNewAllowedRepository(t, "na2na-p", "cargohold"),
			want:  true,
		},
		{
			name:  "正常系: ownerが異なる場合falseが返る",
			ar:    mustNewAllowedRepository(t, "na2na-p", "cargohold"),
			other: mustNewAllowedRepository(t, "other-owner", "cargohold"),
			want:  false,
		},
		{
			name:  "正常系: repoが異なる場合falseが返る",
			ar:    mustNewAllowedRepository(t, "na2na-p", "cargohold"),
			other: mustNewAllowedRepository(t, "na2na-p", "OtherRepo"),
			want:  false,
		},
		{
			name:  "正常系: otherがnilの場合falseが返る",
			ar:    mustNewAllowedRepository(t, "na2na-p", "cargohold"),
			other: nil,
			want:  false,
		},
		{
			name:  "正常系: arがnilでotherもnilの場合trueが返る",
			ar:    nil,
			other: nil,
			want:  true,
		},
		{
			name:  "正常系: arがnilでotherが非nilの場合falseが返る",
			ar:    nil,
			other: mustNewAllowedRepository(t, "na2na-p", "cargohold"),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ar.Equals(tt.other)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Equals mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func mustNewAllowedRepository(t *testing.T, owner, repo string) *domain.AllowedRepository {
	t.Helper()
	ar, err := domain.NewAllowedRepository(owner, repo)
	if err != nil {
		t.Fatalf("failed to create AllowedRepository: %v", err)
	}
	return ar
}
