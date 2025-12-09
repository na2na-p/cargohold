package postgres_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/na2na-p/cargohold/internal/infrastructure/postgres"
	"github.com/pashagolub/pgxmock/v4"
)

// TestNewTxWrapper はNewTxWrapper関数のテスト
func TestNewTxWrapper(t *testing.T) {
	// モックプールの作成
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	// トランザクション開始を期待
	mock.ExpectBegin()

	ctx := context.Background()
	tx, err := mock.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始に失敗しました: %v", err)
	}

	wrapper := postgres.NewTxWrapper(tx)
	if wrapper == nil {
		t.Fatal("NewTxWrapper() returned nil")
	}

	// モックの期待値検証
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
	}
}

// TestTxWrapper_Exec はTxWrapper.Exec処理のテーブルドリブンテスト
func TestTxWrapper_Exec(t *testing.T) {
	type args struct {
		sql       string
		arguments []any
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "正常系: INSERT文の実行に成功",
			args: args{
				sql:       "INSERT INTO test_table (id, name) VALUES ($1, $2)",
				arguments: []any{1, "test"},
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO test_table`).
					WithArgs(1, "test").
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "異常系: SQL実行エラー",
			args: args{
				sql:       "INSERT INTO test_table (id, name) VALUES ($1, $2)",
				arguments: []any{1, "test"},
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO test_table`).
					WithArgs(1, "test").
					WillReturnError(errors.New("execution error"))
				mock.ExpectRollback()
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

			ctx := context.Background()
			tx, err := mock.Begin(ctx)
			if err != nil {
				t.Fatalf("トランザクション開始に失敗しました: %v", err)
			}

			wrapper := postgres.NewTxWrapper(tx)
			_, execErr := wrapper.Exec(ctx, tt.args.sql, tt.args.arguments...)

			if (execErr != nil) != tt.wantErr {
				t.Errorf("Exec() error = %v, wantErr %v", execErr, tt.wantErr)
			}

			// コミットまたはロールバック
			if execErr != nil {
				_ = wrapper.Rollback(ctx)
			} else {
				_ = wrapper.Commit(ctx)
			}

			// モックの期待値検証
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestTxWrapper_QueryRow はTxWrapper.QueryRow処理のテスト
func TestTxWrapper_QueryRow(t *testing.T) {
	// モックプールの作成
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	rows := pgxmock.NewRows([]string{"id", "name"}).AddRow(1, "test")
	mock.ExpectQuery(`SELECT id, name FROM test_table WHERE id`).
		WithArgs(1).
		WillReturnRows(rows)
	mock.ExpectCommit()

	ctx := context.Background()
	tx, err := mock.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始に失敗しました: %v", err)
	}

	wrapper := postgres.NewTxWrapper(tx)
	row := wrapper.QueryRow(ctx, "SELECT id, name FROM test_table WHERE id = $1", 1)

	var id int
	var name string
	if err := row.Scan(&id, &name); err != nil {
		t.Errorf("QueryRow().Scan() error = %v", err)
	}

	if id != 1 || name != "test" {
		t.Errorf("QueryRow() got id=%d, name=%s, want id=1, name=test", id, name)
	}

	_ = wrapper.Commit(ctx)

	// モックの期待値検証
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
	}
}

// TestTxWrapper_CommitRollback はCommitとRollbackのテスト
func TestTxWrapper_CommitRollback(t *testing.T) {
	tests := []struct {
		name      string
		operation string // "commit" or "rollback"
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name:      "正常系: Commitに成功",
			operation: "commit",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:      "正常系: Rollbackに成功",
			operation: "rollback",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			wantErr: false,
		},
		{
			name:      "異常系: Commitに失敗",
			operation: "commit",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectCommit().WillReturnError(errors.New("commit error"))
			},
			wantErr: true,
		},
		{
			name:      "異常系: Rollbackに失敗",
			operation: "rollback",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectRollback().WillReturnError(errors.New("rollback error"))
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

			ctx := context.Background()
			tx, err := mock.Begin(ctx)
			if err != nil {
				t.Fatalf("トランザクション開始に失敗しました: %v", err)
			}

			wrapper := postgres.NewTxWrapper(tx)

			var opErr error
			if tt.operation == "commit" {
				opErr = wrapper.Commit(ctx)
			} else {
				opErr = wrapper.Rollback(ctx)
			}

			if (opErr != nil) != tt.wantErr {
				t.Errorf("%s() error = %v, wantErr %v", tt.operation, opErr, tt.wantErr)
			}

			// モックの期待値検証
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestTransactionManager_BeginTx はTransactionManager.BeginTx処理のテーブルドリブンテスト
func TestTransactionManager_BeginTx(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "正常系: トランザクション開始に成功",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
			},
			wantErr: false,
		},
		{
			name: "異常系: トランザクション開始に失敗",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin().WillReturnError(errors.New("begin error"))
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

			tm := postgres.NewTransactionManager(mock)
			ctx := context.Background()

			txWrapper, err := tm.BeginTx(ctx, postgres.DefaultTxOptions())
			if (err != nil) != tt.wantErr {
				t.Errorf("BeginTx() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && txWrapper == nil {
				t.Error("BeginTx() returned nil TxWrapper for success case")
			}

			// モックの期待値検証
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestTransactionManager_WithTransaction はWithTransaction処理のテーブルドリブンテスト
func TestTransactionManager_WithTransaction(t *testing.T) {
	tests := []struct {
		name      string
		fn        func(ctx context.Context, tx postgres.PoolInterface) error
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "正常系: トランザクション内の処理が成功",
			fn: func(ctx context.Context, tx postgres.PoolInterface) error {
				_, err := tx.Exec(ctx, "INSERT INTO test_table (id) VALUES ($1)", 1)
				return err
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO test_table`).
					WithArgs(1).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "異常系: 処理でエラーが発生してロールバック",
			fn: func(ctx context.Context, tx postgres.PoolInterface) error {
				return errors.New("processing error")
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "processing error",
		},
		{
			name: "異常系: トランザクション開始に失敗",
			fn: func(ctx context.Context, tx postgres.PoolInterface) error {
				return nil
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin().WillReturnError(errors.New("begin error"))
			},
			wantErr: true,
			errMsg:  "failed to begin transaction",
		},
		{
			name: "異常系: コミットに失敗",
			fn: func(ctx context.Context, tx postgres.PoolInterface) error {
				return nil
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectCommit().WillReturnError(errors.New("commit error"))
			},
			wantErr: true,
			errMsg:  "commit failed",
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

			tm := postgres.NewTransactionManager(mock)
			ctx := context.Background()

			err = tm.WithTransaction(ctx, postgres.DefaultTxOptions(), tt.fn)
			if (err != nil) != tt.wantErr {
				t.Errorf("WithTransaction() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("WithTransaction() error = %v, want error containing %q", err, tt.errMsg)
				}
			}

			// モックの期待値検証
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestTransactionManager_WithTransaction_RollbackError はロールバック失敗時のエラーメッセージをテスト
func TestTransactionManager_WithTransaction_RollbackError(t *testing.T) {
	// モックプールの作成
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectRollback().WillReturnError(errors.New("rollback error"))

	tm := postgres.NewTransactionManager(mock)
	ctx := context.Background()

	originalErr := errors.New("original error")
	err = tm.WithTransaction(ctx, postgres.DefaultTxOptions(), func(ctx context.Context, tx postgres.PoolInterface) error {
		return originalErr
	})

	if err == nil {
		t.Fatal("WithTransaction() expected error, got nil")
	}

	// エラーメッセージに両方のエラー情報が含まれることを確認
	if !strings.Contains(err.Error(), "rollback failed") {
		t.Errorf("error should contain 'rollback failed', got: %v", err)
	}
	if !strings.Contains(err.Error(), "original error") {
		t.Errorf("error should contain 'original error', got: %v", err)
	}

	// モックの期待値検証
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
	}
}

// TestTransactionManager_WithTransaction_ContextCancellation はコンテキストキャンセル時の動作をテスト
func TestTransactionManager_WithTransaction_ContextCancellation(t *testing.T) {
	// モックプールの作成
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	tm := postgres.NewTransactionManager(mock)
	ctx, cancel := context.WithCancel(context.Background())

	err = tm.WithTransaction(ctx, postgres.DefaultTxOptions(), func(ctx context.Context, tx postgres.PoolInterface) error {
		cancel() // コンテキストをキャンセル
		return ctx.Err()
	})

	if err == nil {
		t.Fatal("WithTransaction() expected error from context cancellation, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("WithTransaction() error = %v, want context.Canceled", err)
	}

	// モックの期待値検証
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
	}
}

// TestTxWrapper_Close はClose処理のテスト（no-opであることを確認）
func TestTxWrapper_Close(t *testing.T) {
	// モックプールの作成
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	// Closeはno-opなので、CommitもRollbackも期待しない

	ctx := context.Background()
	tx, err := mock.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始に失敗しました: %v", err)
	}

	wrapper := postgres.NewTxWrapper(tx)

	// Closeを呼び出してもパニックしないことを確認
	wrapper.Close()

	// モックの期待値検証
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
	}
}

// TestDAOWithTransaction はDAOがトランザクション内で正常に動作することを確認
func TestDAOWithTransaction(t *testing.T) {
	// モックプールの作成
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("モックプールの作成に失敗しました: %v", err)
	}
	defer mock.Close()

	mock.ExpectBegin()
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
	mock.ExpectCommit()

	tm := postgres.NewTransactionManager(mock)
	ctx := context.Background()

	err = tm.WithTransaction(ctx, postgres.DefaultTxOptions(), func(ctx context.Context, tx postgres.PoolInterface) error {
		// トランザクション内でDAOを使用
		dao := postgres.NewLFSObjectDAO(tx)
		return dao.Insert(ctx, &postgres.LFSObjectRow{
			OID:        "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Size:       1024,
			HashAlgo:   "sha256",
			StorageKey: "test/storage/key",
			Uploaded:   false,
		})
	})

	if err != nil {
		t.Errorf("WithTransaction() error = %v", err)
	}

	// モックの期待値検証
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
	}
}

// TestDefaultTxOptions はDefaultTxOptions関数のテスト
func TestDefaultTxOptions(t *testing.T) {
	tests := []struct {
		name string
		want pgx.TxOptions
	}{
		{
			name: "正常系: デフォルトのトランザクションオプションが返却される",
			want: pgx.TxOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postgres.DefaultTxOptions()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("DefaultTxOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestReadCommittedTxOptions はReadCommittedTxOptions関数のテスト
func TestReadCommittedTxOptions(t *testing.T) {
	tests := []struct {
		name string
		want pgx.TxOptions
	}{
		{
			name: "正常系: ReadCommitted分離レベルのトランザクションオプションが返却される",
			want: pgx.TxOptions{
				IsoLevel: pgx.ReadCommitted,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postgres.ReadCommittedTxOptions()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ReadCommittedTxOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestRepeatableReadTxOptions はRepeatableReadTxOptions関数のテスト
func TestRepeatableReadTxOptions(t *testing.T) {
	tests := []struct {
		name string
		want pgx.TxOptions
	}{
		{
			name: "正常系: RepeatableRead分離レベルのトランザクションオプションが返却される",
			want: pgx.TxOptions{
				IsoLevel: pgx.RepeatableRead,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postgres.RepeatableReadTxOptions()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("RepeatableReadTxOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestSerializableTxOptions はSerializableTxOptions関数のテスト
func TestSerializableTxOptions(t *testing.T) {
	tests := []struct {
		name string
		want pgx.TxOptions
	}{
		{
			name: "正常系: Serializable分離レベルのトランザクションオプションが返却される",
			want: pgx.TxOptions{
				IsoLevel: pgx.Serializable,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postgres.SerializableTxOptions()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("SerializableTxOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
