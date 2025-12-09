package domain_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewRepositoryIdentifier(t *testing.T) {
	type args struct {
		fullName string
	}
	tests := []struct {
		name       string
		args       args
		wantOwner  string
		wantName   string
		wantFull   string
		wantErr    bool
		wantErrVal error
	}{
		{
			name: "正常系: owner/repo形式の文字列から生成できる",
			args: args{
				fullName: "octocat/hello-world",
			},
			wantOwner:  "octocat",
			wantName:   "hello-world",
			wantFull:   "octocat/hello-world",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "正常系: 数字を含むowner/repo",
			args: args{
				fullName: "user123/repo456",
			},
			wantOwner:  "user123",
			wantName:   "repo456",
			wantFull:   "user123/repo456",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "正常系: ハイフンとアンダースコアを含むowner/repo",
			args: args{
				fullName: "my-org_name/my-repo_name",
			},
			wantOwner:  "my-org_name",
			wantName:   "my-repo_name",
			wantFull:   "my-org_name/my-repo_name",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "正常系: ドットを含むrepo名",
			args: args{
				fullName: "owner/repo.go",
			},
			wantOwner:  "owner",
			wantName:   "repo.go",
			wantFull:   "owner/repo.go",
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "異常系: 空文字列",
			args: args{
				fullName: "",
			},
			wantOwner:  "",
			wantName:   "",
			wantFull:   "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidRepositoryIdentifierFormat,
		},
		{
			name: "異常系: スラッシュがない",
			args: args{
				fullName: "noslash",
			},
			wantOwner:  "",
			wantName:   "",
			wantFull:   "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidRepositoryIdentifierFormat,
		},
		{
			name: "異常系: ownerが空",
			args: args{
				fullName: "/repo",
			},
			wantOwner:  "",
			wantName:   "",
			wantFull:   "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidRepositoryIdentifierFormat,
		},
		{
			name: "異常系: repoが空",
			args: args{
				fullName: "owner/",
			},
			wantOwner:  "",
			wantName:   "",
			wantFull:   "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidRepositoryIdentifierFormat,
		},
		{
			name: "異常系: スラッシュのみ",
			args: args{
				fullName: "/",
			},
			wantOwner:  "",
			wantName:   "",
			wantFull:   "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidRepositoryIdentifierFormat,
		},
		{
			name: "異常系: 複数のスラッシュ",
			args: args{
				fullName: "owner/repo/extra",
			},
			wantOwner:  "",
			wantName:   "",
			wantFull:   "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidRepositoryIdentifierFormat,
		},
		{
			name: "異常系: 先頭にスラッシュがある",
			args: args{
				fullName: "/owner/repo",
			},
			wantOwner:  "",
			wantName:   "",
			wantFull:   "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidRepositoryIdentifierFormat,
		},
		{
			name: "異常系: 末尾にスラッシュがある",
			args: args{
				fullName: "owner/repo/",
			},
			wantOwner:  "",
			wantName:   "",
			wantFull:   "",
			wantErr:    true,
			wantErrVal: domain.ErrInvalidRepositoryIdentifierFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewRepositoryIdentifier(tt.args.fullName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewRepositoryIdentifier() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErrVal) {
					t.Errorf("NewRepositoryIdentifier() error = %v, want %v", err, tt.wantErrVal)
				}
				return
			}

			if err != nil {
				t.Errorf("NewRepositoryIdentifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.wantOwner, got.Owner()); diff != "" {
				t.Errorf("RepositoryIdentifier.Owner() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantName, got.Name()); diff != "" {
				t.Errorf("RepositoryIdentifier.Name() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantFull, got.FullName()); diff != "" {
				t.Errorf("RepositoryIdentifier.FullName() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepositoryIdentifier_Equals(t *testing.T) {
	type args struct {
		fullName1 string
		fullName2 string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "正常系: 同じowner/repoは等価",
			args: args{
				fullName1: "octocat/hello-world",
				fullName2: "octocat/hello-world",
			},
			want: true,
		},
		{
			name: "正常系: ownerが異なる場合は非等価",
			args: args{
				fullName1: "octocat/hello-world",
				fullName2: "anothercat/hello-world",
			},
			want: false,
		},
		{
			name: "正常系: repoが異なる場合は非等価",
			args: args{
				fullName1: "octocat/hello-world",
				fullName2: "octocat/goodbye-world",
			},
			want: false,
		},
		{
			name: "正常系: owner/repo両方異なる場合は非等価",
			args: args{
				fullName1: "octocat/hello-world",
				fullName2: "anothercat/goodbye-world",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ri1, err := domain.NewRepositoryIdentifier(tt.args.fullName1)
			if err != nil {
				t.Fatalf("NewRepositoryIdentifier(%q) failed: %v", tt.args.fullName1, err)
			}

			ri2, err := domain.NewRepositoryIdentifier(tt.args.fullName2)
			if err != nil {
				t.Fatalf("NewRepositoryIdentifier(%q) failed: %v", tt.args.fullName2, err)
			}

			got := ri1.Equals(ri2)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("RepositoryIdentifier.Equals() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepositoryIdentifier_Equals_WithNil(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{
			name: "正常系: nilとの比較はfalse",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ri, err := domain.NewRepositoryIdentifier("octocat/hello-world")
			if err != nil {
				t.Fatalf("NewRepositoryIdentifier() failed: %v", err)
			}

			got := ri.Equals(nil)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("RepositoryIdentifier.Equals(nil) mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepositoryIdentifier_Equals_NilReceiver(t *testing.T) {
	type args struct {
		other *domain.RepositoryIdentifier
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "正常系: nilレシーバーとnil引数の比較はtrue",
			args: args{
				other: nil,
			},
			want: true,
		},
		{
			name: "正常系: nilレシーバーと非nil引数の比較はfalse",
			args: args{
				other: func() *domain.RepositoryIdentifier {
					ri, _ := domain.NewRepositoryIdentifier("owner/repo")
					return ri
				}(),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ri *domain.RepositoryIdentifier
			got := ri.Equals(tt.args.other)
			if got != tt.want {
				t.Errorf("RepositoryIdentifier.Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepositoryIdentifier_Immutability(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
	}{
		{
			name:     "正常系: コピー後も元の値は不変",
			fullName: "octocat/hello-world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ri1, err := domain.NewRepositoryIdentifier(tt.fullName)
			if err != nil {
				t.Fatalf("NewRepositoryIdentifier() failed: %v", err)
			}

			ri2 := ri1

			if diff := cmp.Diff(ri1.FullName(), ri2.FullName()); diff != "" {
				t.Errorf("コピー後のFullName()が異なる mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(ri1.Owner(), ri2.Owner()); diff != "" {
				t.Errorf("コピー後のOwner()が異なる mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(ri1.Name(), ri2.Name()); diff != "" {
				t.Errorf("コピー後のName()が異なる mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.fullName, ri1.FullName()); diff != "" {
				t.Errorf("元のFullName()が変更された mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepositoryIdentifier_EqualsFold(t *testing.T) {
	type args struct {
		fullName1 string
		fullName2 string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "正常系: 同じowner/repoは等価",
			args: args{
				fullName1: "octocat/hello-world",
				fullName2: "octocat/hello-world",
			},
			want: true,
		},
		{
			name: "正常系: 大文字小文字が異なるが等価として扱う",
			args: args{
				fullName1: "OctoCat/Hello-World",
				fullName2: "octocat/hello-world",
			},
			want: true,
		},
		{
			name: "正常系: ownerのみ大文字小文字が異なる",
			args: args{
				fullName1: "OCTOCAT/hello-world",
				fullName2: "octocat/hello-world",
			},
			want: true,
		},
		{
			name: "正常系: repoのみ大文字小文字が異なる",
			args: args{
				fullName1: "octocat/HELLO-WORLD",
				fullName2: "octocat/hello-world",
			},
			want: true,
		},
		{
			name: "正常系: ownerが異なる場合は非等価",
			args: args{
				fullName1: "octocat/hello-world",
				fullName2: "anothercat/hello-world",
			},
			want: false,
		},
		{
			name: "正常系: repoが異なる場合は非等価",
			args: args{
				fullName1: "octocat/hello-world",
				fullName2: "octocat/goodbye-world",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ri1, err := domain.NewRepositoryIdentifier(tt.args.fullName1)
			if err != nil {
				t.Fatalf("NewRepositoryIdentifier(%q) failed: %v", tt.args.fullName1, err)
			}

			ri2, err := domain.NewRepositoryIdentifier(tt.args.fullName2)
			if err != nil {
				t.Fatalf("NewRepositoryIdentifier(%q) failed: %v", tt.args.fullName2, err)
			}

			got := ri1.EqualsFold(ri2)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("RepositoryIdentifier.EqualsFold() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepositoryIdentifier_EqualsFold_WithNil(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{
			name: "正常系: nilとの比較はfalse",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ri, err := domain.NewRepositoryIdentifier("octocat/hello-world")
			if err != nil {
				t.Fatalf("NewRepositoryIdentifier() failed: %v", err)
			}

			got := ri.EqualsFold(nil)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("RepositoryIdentifier.EqualsFold(nil) mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepositoryIdentifier_EqualsFold_NilReceiver(t *testing.T) {
	type args struct {
		other *domain.RepositoryIdentifier
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "正常系: nilレシーバーとnil引数の比較はtrue",
			args: args{
				other: nil,
			},
			want: true,
		},
		{
			name: "正常系: nilレシーバーと非nil引数の比較はfalse",
			args: args{
				other: func() *domain.RepositoryIdentifier {
					ri, _ := domain.NewRepositoryIdentifier("owner/repo")
					return ri
				}(),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ri *domain.RepositoryIdentifier
			got := ri.EqualsFold(tt.args.other)
			if got != tt.want {
				t.Errorf("RepositoryIdentifier.EqualsFold() = %v, want %v", got, tt.want)
			}
		})
	}
}
