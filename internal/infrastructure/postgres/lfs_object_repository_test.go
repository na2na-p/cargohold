package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/infrastructure/postgres"
	"github.com/pashagolub/pgxmock/v4"
)

// TestLFSObjectRepositoryImpl_Save は保存処理のテーブルドリブンテスト
func TestLFSObjectRepositoryImpl_Save(t *testing.T) {
	type args struct {
		oid        string
		size       int64
		hashAlgo   string
		storageKey string
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "正常系: オブジェクトの保存に成功",
			args: args{
				oid:        "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				size:       1024,
				hashAlgo:   "sha256",
				storageKey: "test/storage/key",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				// Insert成功を期待
				mock.ExpectExec(`INSERT INTO lfs_objects`).
					WithArgs(
						"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
						int64(1024),
						"sha256",
						"test/storage/key",
						false,
						pgxmock.AnyArg(), // created_at
						pgxmock.AnyArg(), // updated_at
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
		{
			name: "正常系: 大きなサイズの保存",
			args: args{
				oid:        "2222222222222222222222222222222222222222222222222222222222222222",
				size:       1073741824, // 1GB
				hashAlgo:   "sha256",
				storageKey: "test/storage/large",
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
		{
			name: "正常系: 異なるストレージキーで保存",
			args: args{
				oid:        "3333333333333333333333333333333333333333333333333333333333333333",
				size:       2048,
				hashAlgo:   "sha256",
				storageKey: "test/storage/different",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO lfs_objects`).
					WithArgs(
						"3333333333333333333333333333333333333333333333333333333333333333",
						int64(2048),
						"sha256",
						"test/storage/different",
						false,
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
		{
			name: "異常系: 重複保存",
			args: args{
				oid:        "4444444444444444444444444444444444444444444444444444444444444444",
				size:       1024,
				hashAlgo:   "sha256",
				storageKey: "test/storage/duplicate",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				// 重複エラーをシミュレート
				mock.ExpectExec(`INSERT INTO lfs_objects`).
					WithArgs(
						"4444444444444444444444444444444444444444444444444444444444444444",
						int64(1024),
						"sha256",
						"test/storage/duplicate",
						false,
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnError(errors.New("duplicate key value violates unique constraint"))
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

			repo := postgres.NewLFSObjectRepository(mock)
			ctx := context.Background()

			oid, err := domain.NewOID(tt.args.oid)
			if err != nil {
				t.Fatalf("OIDの作成に失敗しました: %v", err)
			}

			size, err := domain.NewSize(tt.args.size)
			if err != nil {
				t.Fatalf("Sizeの作成に失敗しました: %v", err)
			}

			hashAlgo, err := domain.NewHashAlgorithm(tt.args.hashAlgo)
			if err != nil {
				t.Fatalf("HashAlgorithmの作成に失敗しました: %v", err)
			}

			obj, err := domain.NewLFSObject(ctx, oid, size, hashAlgo, tt.args.storageKey)
			if err != nil {
				t.Fatalf("LFSObjectの作成に失敗しました: %v", err)
			}

			err = repo.Save(ctx, obj)

			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}

			// モックの期待値検証
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestLFSObjectRepositoryImpl_FindByOID は取得処理のテーブルドリブンテスト
func TestLFSObjectRepositoryImpl_FindByOID(t *testing.T) {
	type args struct {
		oid        string
		size       int64
		hashAlgo   string
		storageKey string
		uploaded   bool
		createdAt  time.Time
		updatedAt  time.Time
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface, args args)
		wantErr   error
	}{
		{
			name: "正常系: オブジェクトの取得に成功",
			args: args{
				oid:        "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				size:       1024,
				hashAlgo:   "sha256",
				storageKey: "test/storage/key",
				uploaded:   false,
				createdAt:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				updatedAt:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, args args) {
				rows := pgxmock.NewRows([]string{"oid", "size", "hash_algo", "storage_key", "uploaded", "created_at", "updated_at"}).
					AddRow(
						args.oid,
						args.size,
						args.hashAlgo,
						args.storageKey,
						args.uploaded,
						args.createdAt,
						args.updatedAt,
					)
				mock.ExpectQuery(`SELECT oid, size, hash_algo, storage_key, uploaded, created_at, updated_at FROM lfs_objects WHERE oid`).
					WithArgs(args.oid).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "正常系: uploaded=trueのオブジェクト取得時にタイムスタンプが保持される",
			args: args{
				oid:        "2222222222222222222222222222222222222222222222222222222222222222",
				size:       2048,
				hashAlgo:   "sha256",
				storageKey: "test/storage/uploaded",
				uploaded:   true,
				createdAt:  time.Date(2023, 6, 1, 8, 0, 0, 0, time.UTC),
				updatedAt:  time.Date(2024, 1, 10, 14, 30, 0, 0, time.UTC),
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, args args) {
				rows := pgxmock.NewRows([]string{"oid", "size", "hash_algo", "storage_key", "uploaded", "created_at", "updated_at"}).
					AddRow(
						args.oid,
						args.size,
						args.hashAlgo,
						args.storageKey,
						args.uploaded,
						args.createdAt,
						args.updatedAt,
					)
				mock.ExpectQuery(`SELECT oid, size, hash_algo, storage_key, uploaded, created_at, updated_at FROM lfs_objects WHERE oid`).
					WithArgs(args.oid).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "異常系: 存在しないOID",
			args: args{
				oid:        "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				size:       0,
				hashAlgo:   "",
				storageKey: "",
				uploaded:   false,
				createdAt:  time.Time{},
				updatedAt:  time.Time{},
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, args args) {
				mock.ExpectQuery(`SELECT oid, size, hash_algo, storage_key, uploaded, created_at, updated_at FROM lfs_objects WHERE oid`).
					WithArgs(args.oid).
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: domain.ErrNotFound,
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
			tt.mockSetup(mock, tt.args)

			repo := postgres.NewLFSObjectRepository(mock)
			ctx := context.Background()

			oid, err := domain.NewOID(tt.args.oid)
			if err != nil {
				t.Fatalf("OIDの作成に失敗しました: %v", err)
			}

			// テスト実行
			retrieved, err := repo.FindByOID(ctx, oid)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("FindByOID() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			} else if err != nil {
				t.Errorf("FindByOID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 正常系の場合はフィールドを検証
			if tt.wantErr == nil {
				if retrieved.OID().String() != oid.String() {
					t.Errorf("OIDが一致しません: expected=%s, got=%s", oid.String(), retrieved.OID().String())
				}
				if retrieved.Size().Int64() != tt.args.size {
					t.Errorf("Sizeが一致しません: expected=%d, got=%d", tt.args.size, retrieved.Size().Int64())
				}
				if retrieved.HashAlgo() != tt.args.hashAlgo {
					t.Errorf("HashAlgoが一致しません: expected=%s, got=%s", tt.args.hashAlgo, retrieved.HashAlgo())
				}
				if retrieved.GetStorageKey() != tt.args.storageKey {
					t.Errorf("StorageKeyが一致しません: expected=%s, got=%s", tt.args.storageKey, retrieved.GetStorageKey())
				}
				if retrieved.IsUploaded() != tt.args.uploaded {
					t.Errorf("Uploadedフラグが一致しません: expected=%v, got=%v", tt.args.uploaded, retrieved.IsUploaded())
				}
				if !retrieved.CreatedAt().Equal(tt.args.createdAt) {
					t.Errorf("CreatedAtが一致しません: expected=%v, got=%v", tt.args.createdAt, retrieved.CreatedAt())
				}
				if !retrieved.UpdatedAt().Equal(tt.args.updatedAt) {
					t.Errorf("UpdatedAtが一致しません: expected=%v, got=%v", tt.args.updatedAt, retrieved.UpdatedAt())
				}
			}

			// モックの期待値検証
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestLFSObjectRepositoryImpl_Update は更新処理のテーブルドリブンテスト
func TestLFSObjectRepositoryImpl_Update(t *testing.T) {
	type args struct {
		oid        string
		size       int64
		hashAlgo   string
		storageKey string
		uploaded   bool
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name: "正常系: アップロード完了フラグの更新",
			args: args{
				oid:        "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				size:       1024,
				hashAlgo:   "sha256",
				storageKey: "test/storage/key",
				uploaded:   true,
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
			wantErr: nil,
		},
		{
			name: "異常系: 存在しないオブジェクトの更新",
			args: args{
				oid:        "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				size:       1024,
				hashAlgo:   "sha256",
				storageKey: "test/storage/key",
				uploaded:   true,
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
			wantErr: domain.ErrNotFound,
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

			repo := postgres.NewLFSObjectRepository(mock)
			ctx := context.Background()

			oid, err := domain.NewOID(tt.args.oid)
			if err != nil {
				t.Fatalf("OIDの作成に失敗しました: %v", err)
			}

			size, err := domain.NewSize(tt.args.size)
			if err != nil {
				t.Fatalf("Sizeの作成に失敗しました: %v", err)
			}

			hashAlgo, err := domain.NewHashAlgorithm(tt.args.hashAlgo)
			if err != nil {
				t.Fatalf("HashAlgorithmの作成に失敗しました: %v", err)
			}

			obj, err := domain.NewLFSObject(ctx, oid, size, hashAlgo, tt.args.storageKey)
			if err != nil {
				t.Fatalf("LFSObjectの作成に失敗しました: %v", err)
			}

			if tt.args.uploaded {
				obj.MarkAsUploaded(ctx)
			}

			err = repo.Update(ctx, obj)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			} else if err != nil {
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

// TestLFSObjectRepositoryImpl_ExistsByOID は存在確認のテーブルドリブンテスト
func TestLFSObjectRepositoryImpl_ExistsByOID(t *testing.T) {
	type args struct {
		oid string
	}
	tests := []struct {
		name       string
		args       args
		mockSetup  func(mock pgxmock.PgxPoolIface)
		wantExists bool
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

			repo := postgres.NewLFSObjectRepository(mock)
			ctx := context.Background()

			oid, err := domain.NewOID(tt.args.oid)
			if err != nil {
				t.Fatalf("OIDの作成に失敗しました: %v", err)
			}

			// テスト実行
			exists, err := repo.ExistsByOID(ctx, oid)
			if err != nil {
				t.Fatalf("ExistsByOID() error = %v", err)
			}

			if exists != tt.wantExists {
				t.Errorf("ExistsByOID() = %v, want %v", exists, tt.wantExists)
			}

			// モックの期待値検証
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestLFSObjectRepositoryImpl_SaveAndFindByOID は保存と取得の統合テスト
func TestLFSObjectRepositoryImpl_SaveAndFindByOID(t *testing.T) {
	// モックプールの作成
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	testCases := []struct {
		oid        string
		size       int64
		hashAlgo   string
		storageKey string
	}{
		{"1111111111111111111111111111111111111111111111111111111111111111", 1024, "sha256", "test/storage/key1"},
		{"2222222222222222222222222222222222222222222222222222222222222222", 2048, "sha256", "test/storage/key2"},
		{"3333333333333333333333333333333333333333333333333333333333333333", 4096, "sha256", "test/storage/key3"},
	}

	// モックのセットアップ: 3つのInsertを期待
	for _, tc := range testCases {
		mock.ExpectExec(`INSERT INTO lfs_objects`).
			WithArgs(
				tc.oid,
				tc.size,
				tc.hashAlgo,
				tc.storageKey,
				false,
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
			).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}

	// モックのセットアップ: 3つのSELECTを期待
	for _, tc := range testCases {
		rows := pgxmock.NewRows([]string{"oid", "size", "hash_algo", "storage_key", "uploaded", "created_at", "updated_at"}).
			AddRow(tc.oid, tc.size, tc.hashAlgo, tc.storageKey, false, time.Now(), time.Now())
		mock.ExpectQuery(`SELECT oid, size, hash_algo, storage_key, uploaded, created_at, updated_at FROM lfs_objects WHERE oid`).
			WithArgs(tc.oid).
			WillReturnRows(rows)
	}

	repo := postgres.NewLFSObjectRepository(mock)
	ctx := context.Background()

	// 全てのオブジェクトを保存
	for _, tc := range testCases {
		oid, err := domain.NewOID(tc.oid)
		if err != nil {
			t.Fatalf("OIDの作成に失敗しました: %v", err)
		}
		size, err := domain.NewSize(tc.size)
		if err != nil {
			t.Fatalf("Sizeの作成に失敗しました: %v", err)
		}
		hashAlgo, err := domain.NewHashAlgorithm(tc.hashAlgo)
		if err != nil {
			t.Fatalf("HashAlgorithmの作成に失敗しました: %v", err)
		}
		obj, err := domain.NewLFSObject(ctx, oid, size, hashAlgo, tc.storageKey)
		if err != nil {
			t.Fatalf("LFSObjectの作成に失敗しました: %v", err)
		}

		if err := repo.Save(ctx, obj); err != nil {
			t.Fatalf("オブジェクトの保存に失敗しました (OID=%s): %v", tc.oid, err)
		}
	}

	// 全てのオブジェクトが取得できることを確認
	for _, tc := range testCases {
		oid, err := domain.NewOID(tc.oid)
		if err != nil {
			t.Fatalf("OIDの作成に失敗しました: %v", err)
		}

		retrieved, err := repo.FindByOID(ctx, oid)
		if err != nil {
			t.Fatalf("オブジェクトの取得に失敗しました (OID=%s): %v", tc.oid, err)
		}

		if retrieved.OID().String() != tc.oid {
			t.Errorf("OIDが一致しません: expected=%s, got=%s", tc.oid, retrieved.OID().String())
		}
		if retrieved.Size().Int64() != tc.size {
			t.Errorf("Sizeが一致しません: expected=%d, got=%d", tc.size, retrieved.Size().Int64())
		}
		if retrieved.HashAlgo() != tc.hashAlgo {
			t.Errorf("HashAlgoが一致しません: expected=%s, got=%s", tc.hashAlgo, retrieved.HashAlgo())
		}
		if retrieved.GetStorageKey() != tc.storageKey {
			t.Errorf("StorageKeyが一致しません: expected=%s, got=%s", tc.storageKey, retrieved.GetStorageKey())
		}
	}

	// モックの期待値検証
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
	}
}

// TestLFSObjectRepositoryImpl_cmpDiff はcmpパッケージを使った比較テスト
func TestLFSObjectRepositoryImpl_cmpDiff(t *testing.T) {
	// モックプールの作成
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	oid, err := domain.NewOID("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	if err != nil {
		t.Fatalf("OIDの作成に失敗しました: %v", err)
	}
	size, err := domain.NewSize(1024)
	if err != nil {
		t.Fatalf("Sizeの作成に失敗しました: %v", err)
	}
	hashAlgo, err := domain.NewHashAlgorithm("sha256")
	if err != nil {
		t.Fatalf("HashAlgorithmの作成に失敗しました: %v", err)
	}
	ctx := context.Background()
	obj, err := domain.NewLFSObject(ctx, oid, size, hashAlgo, "test/storage/key")
	if err != nil {
		t.Fatalf("LFSObjectの作成に失敗しました: %v", err)
	}

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

	// SELECTモックのセットアップ
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

	repo := postgres.NewLFSObjectRepository(mock)

	// 保存
	if err := repo.Save(ctx, obj); err != nil {
		t.Fatalf("オブジェクトの保存に失敗しました: %v", err)
	}

	// 取得
	retrieved, err := repo.FindByOID(ctx, oid)
	if err != nil {
		t.Fatalf("オブジェクトの取得に失敗しました: %v", err)
	}

	// cmp.Diffを使った比較（タイムスタンプは無視）
	opts := cmpopts.IgnoreFields(domain.LFSObject{}, "createdAt", "updatedAt")
	if diff := cmp.Diff(obj, retrieved, opts, cmp.AllowUnexported(domain.LFSObject{}, domain.OID{}, domain.Size{}, domain.HashAlgorithm{}, domain.StorageKey{})); diff != "" {
		t.Errorf("取得したオブジェクトが期待値と異なります (-want +got):\n%s", diff)
	}

	// モックの期待値検証
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
	}
}
