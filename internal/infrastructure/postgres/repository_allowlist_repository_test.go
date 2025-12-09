package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/na2na-p/cargohold/internal/domain"
	infra "github.com/na2na-p/cargohold/internal/infrastructure"
	"github.com/na2na-p/cargohold/internal/infrastructure/postgres"
	"github.com/pashagolub/pgxmock/v4"
)

func TestRepositoryAllowlistRepositoryImpl_IsAllowed(t *testing.T) {
	type args struct {
		owner string
		repo  string
	}
	tests := []struct {
		name        string
		args        args
		mockSetup   func(mock pgxmock.PgxPoolIface)
		wantAllowed bool
		wantErr     bool
	}{
		{
			name: "正常系: リポジトリが許可リストに存在する",
			args: args{
				owner: "owner",
				repo:  "repo",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs("owner/repo").
					WillReturnRows(rows)
			},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name: "正常系: リポジトリが許可リストに存在しない",
			args: args{
				owner: "unknown",
				repo:  "repo",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs("unknown/repo").
					WillReturnRows(rows)
			},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name: "異常系: データベースエラー",
			args: args{
				owner: "owner",
				repo:  "repo",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs("owner/repo").
					WillReturnError(errors.New("database connection error"))
			},
			wantAllowed: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("モックプールの作成に失敗しました: %v", err)
			}
			defer mock.Close()

			tt.mockSetup(mock)

			repo := postgres.NewRepositoryAllowlistRepository(mock)
			ctx := context.Background()

			allowedRepo := mustNewAllowedRepository(t, tt.args.owner, tt.args.repo)
			allowed, err := repo.IsAllowed(ctx, allowedRepo)

			if (err != nil) != tt.wantErr {
				t.Errorf("IsAllowed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if allowed != tt.wantAllowed {
				t.Errorf("IsAllowed() = %v, want %v", allowed, tt.wantAllowed)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

func TestRepositoryAllowlistRepositoryImpl_Add(t *testing.T) {
	type args struct {
		owner string
		repo  string
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "正常系: 新規リポジトリの追加に成功",
			args: args{
				owner: "owner",
				repo:  "new-repo",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO repository_allowlist`).
					WithArgs("owner/new-repo").
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
		{
			name: "正常系: 既存リポジトリの重複追加（ON CONFLICT DO NOTHING）",
			args: args{
				owner: "owner",
				repo:  "existing-repo",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO repository_allowlist`).
					WithArgs("owner/existing-repo").
					WillReturnResult(pgxmock.NewResult("INSERT", 0))
			},
			wantErr: false,
		},
		{
			name: "異常系: データベースエラー",
			args: args{
				owner: "owner",
				repo:  "repo",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO repository_allowlist`).
					WithArgs("owner/repo").
					WillReturnError(errors.New("database connection error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("モックプールの作成に失敗しました: %v", err)
			}
			defer mock.Close()

			tt.mockSetup(mock)

			repo := postgres.NewRepositoryAllowlistRepository(mock)
			ctx := context.Background()

			allowedRepo := mustNewAllowedRepository(t, tt.args.owner, tt.args.repo)
			err = repo.Add(ctx, allowedRepo)

			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

func TestRepositoryAllowlistRepositoryImpl_Remove(t *testing.T) {
	type args struct {
		owner string
		repo  string
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name: "正常系: リポジトリの削除に成功",
			args: args{
				owner: "owner",
				repo:  "repo",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM repository_allowlist WHERE repository`).
					WithArgs("owner/repo").
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: nil,
		},
		{
			name: "異常系: 存在しないリポジトリの削除",
			args: args{
				owner: "unknown",
				repo:  "repo",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM repository_allowlist WHERE repository`).
					WithArgs("unknown/repo").
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: infra.ErrNotFound,
		},
		{
			name: "異常系: データベースエラー",
			args: args{
				owner: "owner",
				repo:  "repo",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM repository_allowlist WHERE repository`).
					WithArgs("owner/repo").
					WillReturnError(errors.New("database connection error"))
			},
			wantErr: errors.New("database connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("モックプールの作成に失敗しました: %v", err)
			}
			defer mock.Close()

			tt.mockSetup(mock)

			repo := postgres.NewRepositoryAllowlistRepository(mock)
			ctx := context.Background()

			allowedRepo := mustNewAllowedRepository(t, tt.args.owner, tt.args.repo)
			err = repo.Remove(ctx, allowedRepo)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Remove() error = nil, wantErr %v", tt.wantErr)
				}
				if errors.Is(err, tt.wantErr) {
					return
				}
				if tt.wantErr.Error() != "" && err.Error() == tt.wantErr.Error() {
					return
				}
				t.Errorf("Remove() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				if err != nil {
					t.Errorf("Remove() unexpected error: %v", err)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

func TestRepositoryAllowlistRepositoryImpl_List(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(mock pgxmock.PgxPoolIface)
		want      []*domain.AllowedRepository
		wantErr   bool
	}{
		{
			name: "正常系: 複数のリポジトリを取得",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"repository"}).
					AddRow("other/repo3").
					AddRow("owner/repo1").
					AddRow("owner/repo2")
				mock.ExpectQuery(`SELECT repository FROM repository_allowlist ORDER BY repository`).
					WillReturnRows(rows)
			},
			want: []*domain.AllowedRepository{
				mustNewAllowedRepositoryHelper("other", "repo3"),
				mustNewAllowedRepositoryHelper("owner", "repo1"),
				mustNewAllowedRepositoryHelper("owner", "repo2"),
			},
			wantErr: false,
		},
		{
			name: "正常系: 空のリスト",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"repository"})
				mock.ExpectQuery(`SELECT repository FROM repository_allowlist ORDER BY repository`).
					WillReturnRows(rows)
			},
			want:    []*domain.AllowedRepository{},
			wantErr: false,
		},
		{
			name: "異常系: データベースエラー",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT repository FROM repository_allowlist ORDER BY repository`).
					WillReturnError(errors.New("database connection error"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("モックプールの作成に失敗しました: %v", err)
			}
			defer mock.Close()

			tt.mockSetup(mock)

			repo := postgres.NewRepositoryAllowlistRepository(mock)
			ctx := context.Background()

			got, err := repo.List(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("List() returned %d items, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].String() != tt.want[i].String() {
					t.Errorf("List()[%d] = %s, want %s", i, got[i].String(), tt.want[i].String())
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

func TestRepositoryAllowlistRepositoryImpl_AddAndList(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	repositories := []*domain.AllowedRepository{
		mustNewAllowedRepositoryHelper("other", "repo3"),
		mustNewAllowedRepositoryHelper("owner", "repo1"),
		mustNewAllowedRepositoryHelper("owner", "repo2"),
	}

	for _, r := range repositories {
		mock.ExpectExec(`INSERT INTO repository_allowlist`).
			WithArgs(r.String()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}

	rows := pgxmock.NewRows([]string{"repository"})
	for _, r := range repositories {
		rows.AddRow(r.String())
	}
	mock.ExpectQuery(`SELECT repository FROM repository_allowlist ORDER BY repository`).
		WillReturnRows(rows)

	repo := postgres.NewRepositoryAllowlistRepository(mock)
	ctx := context.Background()

	for _, r := range repositories {
		if err := repo.Add(ctx, r); err != nil {
			t.Fatalf("リポジトリの追加に失敗しました (repository=%s): %v", r.String(), err)
		}
	}

	got, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("リスト取得に失敗しました: %v", err)
	}

	if len(got) != len(repositories) {
		t.Fatalf("List() returned %d items, want %d", len(got), len(repositories))
	}

	for i := range got {
		if got[i].String() != repositories[i].String() {
			t.Errorf("List()[%d] = %s, want %s", i, got[i].String(), repositories[i].String())
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
	}
}

func TestRepositoryAllowlistRepositoryImpl_IsAllowed_NilRepository(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	repo := postgres.NewRepositoryAllowlistRepository(mock)
	ctx := context.Background()

	allowed, err := repo.IsAllowed(ctx, nil)

	if err == nil {
		t.Fatal("IsAllowed() error = nil, want error for nil repository")
	}
	if allowed != false {
		t.Errorf("IsAllowed() = %v, want false", allowed)
	}
}

func TestRepositoryAllowlistRepositoryImpl_Add_NilRepository(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	repo := postgres.NewRepositoryAllowlistRepository(mock)
	ctx := context.Background()

	err = repo.Add(ctx, nil)

	if err == nil {
		t.Fatal("Add() error = nil, want error for nil repository")
	}
}

func TestRepositoryAllowlistRepositoryImpl_Remove_NilRepository(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	repo := postgres.NewRepositoryAllowlistRepository(mock)
	ctx := context.Background()

	err = repo.Remove(ctx, nil)

	if err == nil {
		t.Fatal("Remove() error = nil, want error for nil repository")
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

func mustNewAllowedRepositoryHelper(owner, repo string) *domain.AllowedRepository {
	ar, err := domain.NewAllowedRepository(owner, repo)
	if err != nil {
		panic(err)
	}
	return ar
}
