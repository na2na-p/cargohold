package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// TxPoolInterface はトランザクション機能を持つPoolInterface
type TxPoolInterface interface {
	PoolInterface
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// TxWrapper はpgx.TxをPoolInterfaceとして使用するためのラッパー
// これにより、既存のDAOをトランザクション内で再利用できる
type TxWrapper struct {
	tx pgx.Tx
}

// NewTxWrapper は新しいTxWrapperを作成する
func NewTxWrapper(tx pgx.Tx) *TxWrapper {
	return &TxWrapper{tx: tx}
}

// Exec はSQLを実行する（PoolInterface実装）
func (w *TxWrapper) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return w.tx.Exec(ctx, sql, arguments...)
}

// QueryRow は単一行を取得する（PoolInterface実装）
func (w *TxWrapper) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return w.tx.QueryRow(ctx, sql, args...)
}

// Query は複数行を取得する（PoolInterface実装）
func (w *TxWrapper) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return w.tx.Query(ctx, sql, args...)
}

// Close はトランザクションラッパーのクローズ処理（no-op）
// 実際のトランザクション終了はCommit/Rollbackで行う
func (w *TxWrapper) Close() {
	// トランザクションのクローズはCommit/Rollbackで行うため、ここでは何もしない
}

// Commit はトランザクションをコミットする
func (w *TxWrapper) Commit(ctx context.Context) error {
	return w.tx.Commit(ctx)
}

// Rollback はトランザクションをロールバックする
func (w *TxWrapper) Rollback(ctx context.Context) error {
	return w.tx.Rollback(ctx)
}

// TransactionManager はトランザクションを管理する
type TransactionManager struct {
	pool TxPoolInterface
}

// NewTransactionManager は新しいTransactionManagerを作成する
func NewTransactionManager(pool TxPoolInterface) *TransactionManager {
	return &TransactionManager{pool: pool}
}

// BeginTx は新しいトランザクションを開始する
func (tm *TransactionManager) BeginTx(ctx context.Context, opts pgx.TxOptions) (*TxWrapper, error) {
	tx, err := tm.pool.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return NewTxWrapper(tx), nil
}

// WithTransaction はトランザクション内で処理を実行する
// fnがエラーを返した場合、または panicが発生した場合は自動的にロールバックする
// fnが正常に完了した場合は自動的にコミットする
func (tm *TransactionManager) WithTransaction(ctx context.Context, opts pgx.TxOptions, fn func(ctx context.Context, tx PoolInterface) error) error {
	txWrapper, err := tm.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			_ = txWrapper.Rollback(ctx)
			panic(r)
		}
	}()

	if err := fn(ctx, txWrapper); err != nil {
		if rbErr := txWrapper.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %w (original error: %v)", rbErr, err)
		}
		return err
	}

	if err := txWrapper.Commit(ctx); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

// DefaultTxOptions はデフォルトのトランザクションオプションを返す
func DefaultTxOptions() pgx.TxOptions {
	return pgx.TxOptions{}
}

// ReadCommittedTxOptions はReadCommitted分離レベルのトランザクションオプションを返す
func ReadCommittedTxOptions() pgx.TxOptions {
	return pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	}
}

// RepeatableReadTxOptions はRepeatableRead分離レベルのトランザクションオプションを返す
func RepeatableReadTxOptions() pgx.TxOptions {
	return pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead,
	}
}

// SerializableTxOptions はSerializable分離レベルのトランザクションオプションを返す
func SerializableTxOptions() pgx.TxOptions {
	return pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	}
}
