package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/na2na-p/cargohold/internal/infrastructure/postgres"
	"github.com/pashagolub/pgxmock/v4"
)

// TestNewLFSObjectDAO はNewLFSObjectDAO関数のテスト
func TestNewLFSObjectDAO(t *testing.T) {
	// モックプールの作成
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	dao := postgres.NewLFSObjectDAO(mock)

	if dao == nil {
		t.Fatal("NewLFSObjectDAO() returned nil")
	}
}

// TestLFSObjectDAO_Insert はInsert処理のテーブルドリブンテスト
func TestLFSObjectDAO_Insert(t *testing.T) {
	type args struct {
		row *postgres.LFSObjectRow
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "正常系: Insertに成功",
			args: args{
				row: &postgres.LFSObjectRow{
					OID:        "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					Size:       1024,
					HashAlgo:   "sha256",
					StorageKey: "test/storage/key",
					Uploaded:   false,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				},
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO lfs_objects`).
					WithArgs(
						"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
						int64(1024),
						"sha256",
						"test/storage/key",
						false,
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
		{
			name: "正常系: 大きなサイズのInsert",
			args: args{
				row: &postgres.LFSObjectRow{
					OID:        "2222222222222222222222222222222222222222222222222222222222222222",
					Size:       1073741824,
					HashAlgo:   "sha256",
					StorageKey: "test/storage/large",
					Uploaded:   false,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				},
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO lfs_objects`).
					WithArgs(
						"2222222222222222222222222222222222222222222222222222222222222222",
						int64(1073741824),
						"sha256",
						"test/storage/large",
						false,
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックプールの作成
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("モックプールの作成に失敗しました: %v", err)
			}
			defer mock.Close()

			// モックのセットアップ
			tt.mockSetup(mock)

			dao := postgres.NewLFSObjectDAO(mock)
			ctx := context.Background()

			err = dao.Insert(ctx, tt.args.row)
			if (err != nil) != tt.wantErr {
				t.Errorf("Insert() error = %v, wantErr %v", err, tt.wantErr)
			}

			// モックの期待値検証
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestLFSObjectDAO_FindByOID はFindByOID処理のテーブルドリブンテスト
func TestLFSObjectDAO_FindByOID(t *testing.T) {
	type args struct {
		oid string
		row *postgres.LFSObjectRow
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "正常系: FindByOIDに成功",
			args: args{
				oid: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				row: &postgres.LFSObjectRow{
					OID:        "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					Size:       1024,
					HashAlgo:   "sha256",
					StorageKey: "test/storage/key",
					Uploaded:   false,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				},
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"oid", "size", "hash_algo", "storage_key", "uploaded", "created_at", "updated_at"}).
					AddRow(
						"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
						int64(1024),
						"sha256",
						"test/storage/key",
						false,
						time.Now(),
						time.Now(),
					)
				mock.ExpectQuery(`SELECT oid, size, hash_algo, storage_key, uploaded, created_at, updated_at FROM lfs_objects WHERE oid`).
					WithArgs("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef").
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "異常系: 存在しないOID",
			args: args{
				oid: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				row: nil,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT oid, size, hash_algo, storage_key, uploaded, created_at, updated_at FROM lfs_objects WHERE oid`).
					WithArgs("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890").
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックプールの作成
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("モックプールの作成に失敗しました: %v", err)
			}
			defer mock.Close()

			// モックのセットアップ
			tt.mockSetup(mock)

			dao := postgres.NewLFSObjectDAO(mock)
			ctx := context.Background()

			result, err := dao.FindByOID(ctx, tt.args.oid)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByOID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.args.row != nil {
				if result.OID != tt.args.row.OID {
					t.Errorf("FindByOID() OID = %v, want %v", result.OID, tt.args.row.OID)
				}
				if result.Size != tt.args.row.Size {
					t.Errorf("FindByOID() Size = %v, want %v", result.Size, tt.args.row.Size)
				}
			}

			// モックの期待値検証
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestLFSObjectDAO_Update はUpdate処理のテーブルドリブンテスト
func TestLFSObjectDAO_Update(t *testing.T) {
	type args struct {
		updateRow *postgres.LFSObjectRow
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "正常系: Updateに成功",
			args: args{
				updateRow: &postgres.LFSObjectRow{
					OID:        "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					Size:       1024,
					HashAlgo:   "sha256",
					StorageKey: "test/storage/key",
					Uploaded:   true,
					UpdatedAt:  time.Now(),
				},
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`UPDATE lfs_objects SET`).
					WithArgs(
						"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
						int64(1024),
						"sha256",
						"test/storage/key",
						true,
						pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			wantErr: false,
		},
		{
			name: "異常系: 存在しないOIDの更新",
			args: args{
				updateRow: &postgres.LFSObjectRow{
					OID:        "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
					Size:       1024,
					HashAlgo:   "sha256",
					StorageKey: "test/storage/key",
					Uploaded:   true,
					UpdatedAt:  time.Now(),
				},
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`UPDATE lfs_objects SET`).
					WithArgs(
						"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
						int64(1024),
						"sha256",
						"test/storage/key",
						true,
						pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックプールの作成
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("モックプールの作成に失敗しました: %v", err)
			}
			defer mock.Close()

			// モックのセットアップ
			tt.mockSetup(mock)

			dao := postgres.NewLFSObjectDAO(mock)
			ctx := context.Background()

			err = dao.Update(ctx, tt.args.updateRow)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// モックの期待値検証
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestLFSObjectDAO_Exists はExists処理のテーブルドリブンテスト
func TestLFSObjectDAO_Exists(t *testing.T) {
	type args struct {
		oid string
	}
	tests := []struct {
		name       string
		args       args
		mockSetup  func(mock pgxmock.PgxPoolIface)
		wantExists bool
		wantErr    bool
	}{
		{
			name: "正常系: オブジェクトが存在する",
			args: args{
				oid: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef").
					WillReturnRows(rows)
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "正常系: オブジェクトが存在しない",
			args: args{
				oid: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890").
					WillReturnRows(rows)
			},
			wantExists: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックプールの作成
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("モックプールの作成に失敗しました: %v", err)
			}
			defer mock.Close()

			// モックのセットアップ
			tt.mockSetup(mock)

			dao := postgres.NewLFSObjectDAO(mock)
			ctx := context.Background()

			exists, err := dao.Exists(ctx, tt.args.oid)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if exists != tt.wantExists {
				t.Errorf("Exists() = %v, want %v", exists, tt.wantExists)
			}

			// モックの期待値検証
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}
