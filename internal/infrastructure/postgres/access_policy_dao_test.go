package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/na2na-p/cargohold/internal/infrastructure/postgres"
	"github.com/pashagolub/pgxmock/v4"
)

// TestNewAccessPolicyDAO はNewAccessPolicyDAO関数のテスト
func TestNewAccessPolicyDAO(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "正常系: DAOインスタンスが生成される",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("モックプールの作成に失敗しました: %v", err)
			}
			defer mock.Close()

			dao := postgres.NewAccessPolicyDAO(mock)

			if dao == nil {
				t.Fatal("NewAccessPolicyDAO() returned nil")
			}
		})
	}
}

// TestAccessPolicyDAO_FindByOID はFindByOID処理のテーブルドリブンテスト
func TestAccessPolicyDAO_FindByOID(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	type args struct {
		oid string
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		want      *postgres.AccessPolicyRow
		wantErr   error
	}{
		{
			name: "正常系: FindByOIDに成功",
			args: args{
				oid: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "lfs_object_oid", "repository", "created_at"}).
					AddRow(
						int64(1),
						"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
						"owner/repo",
						fixedTime,
					)
				mock.ExpectQuery(`SELECT id, lfs_object_oid, repository, created_at FROM lfs_object_access_policies WHERE lfs_object_oid`).
					WithArgs("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef").
					WillReturnRows(rows)
			},
			want: &postgres.AccessPolicyRow{
				ID:           1,
				LfsObjectOid: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				Repository:   "owner/repo",
				CreatedAt:    fixedTime,
			},
			wantErr: nil,
		},
		{
			name: "異常系: 存在しないOID",
			args: args{
				oid: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT id, lfs_object_oid, repository, created_at FROM lfs_object_access_policies WHERE lfs_object_oid`).
					WithArgs("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890").
					WillReturnError(pgx.ErrNoRows)
			},
			want:    nil,
			wantErr: pgx.ErrNoRows,
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

			dao := postgres.NewAccessPolicyDAO(mock)
			ctx := context.Background()

			got, err := dao.FindByOID(ctx, tt.args.oid)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("FindByOID() error = nil, wantErr %v", tt.wantErr)
				}
				if !cmp.Equal(err.Error(), tt.wantErr.Error()) {
					t.Errorf("FindByOID() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("FindByOID() unexpected error = %v", err)
				}
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("FindByOID() mismatch (-want +got):\n%s", diff)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestAccessPolicyDAO_Upsert はUpsert処理のテーブルドリブンテスト
func TestAccessPolicyDAO_Upsert(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	type args struct {
		row *postgres.AccessPolicyRow
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "正常系: 新規レコードの挿入に成功",
			args: args{
				row: &postgres.AccessPolicyRow{
					LfsObjectOid: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					Repository:   "owner/repo",
					CreatedAt:    fixedTime,
				},
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO lfs_object_access_policies`).
					WithArgs(
						"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
						"owner/repo",
						fixedTime,
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
		{
			name: "正常系: 既存レコードの更新に成功（UPSERT）",
			args: args{
				row: &postgres.AccessPolicyRow{
					LfsObjectOid: "2222222222222222222222222222222222222222222222222222222222222222",
					Repository:   "new-owner/new-repo",
					CreatedAt:    fixedTime,
				},
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO lfs_object_access_policies`).
					WithArgs(
						"2222222222222222222222222222222222222222222222222222222222222222",
						"new-owner/new-repo",
						fixedTime,
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
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

			dao := postgres.NewAccessPolicyDAO(mock)
			ctx := context.Background()

			err = dao.Upsert(ctx, tt.args.row)
			if (err != nil) != tt.wantErr {
				t.Errorf("Upsert() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestAccessPolicyDAO_Delete はDelete処理のテーブルドリブンテスト
func TestAccessPolicyDAO_Delete(t *testing.T) {
	type args struct {
		oid string
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name: "正常系: 削除に成功",
			args: args{
				oid: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM lfs_object_access_policies WHERE lfs_object_oid`).
					WithArgs("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef").
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: nil,
		},
		{
			name: "異常系: 存在しないOIDの削除",
			args: args{
				oid: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM lfs_object_access_policies WHERE lfs_object_oid`).
					WithArgs("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890").
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: pgx.ErrNoRows,
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

			dao := postgres.NewAccessPolicyDAO(mock)
			ctx := context.Background()

			err = dao.Delete(ctx, tt.args.oid)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Delete() error = nil, wantErr %v", tt.wantErr)
				}
				if !cmp.Equal(err.Error(), tt.wantErr.Error()) {
					t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("Delete() unexpected error = %v", err)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}
