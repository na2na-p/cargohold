package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/infrastructure/postgres"
	"github.com/pashagolub/pgxmock/v4"
)

// TestNewAccessPolicyRepository はNewAccessPolicyRepository関数のテスト
func TestNewAccessPolicyRepository(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "正常系: Repositoryインスタンスが生成される",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("モックプールの作成に失敗しました: %v", err)
			}
			defer mock.Close()

			repo := postgres.NewAccessPolicyRepository(mock)

			if repo == nil {
				t.Fatal("NewAccessPolicyRepository() returned nil")
			}
		})
	}
}

// TestAccessPolicyRepositoryImpl_FindByOID はFindByOID処理のテーブルドリブンテスト
func TestAccessPolicyRepositoryImpl_FindByOID(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	validOID := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	notFoundOID := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	type args struct {
		oid string
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantID    int64
		wantOID   string
		wantRepo  string
		wantNil   bool
		wantErr   error
	}{
		{
			name: "正常系: FindByOIDに成功",
			args: args{
				oid: validOID,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "lfs_object_oid", "repository", "created_at"}).
					AddRow(
						int64(1),
						validOID,
						"owner/repo",
						fixedTime,
					)
				mock.ExpectQuery(`SELECT id, lfs_object_oid, repository, created_at FROM lfs_object_access_policies WHERE lfs_object_oid`).
					WithArgs(validOID).
					WillReturnRows(rows)
			},
			wantID:   1,
			wantOID:  validOID,
			wantRepo: "owner/repo",
			wantNil:  false,
			wantErr:  nil,
		},
		{
			name: "異常系: 存在しないOIDの場合はErrAccessPolicyNotFoundを返す",
			args: args{
				oid: notFoundOID,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT id, lfs_object_oid, repository, created_at FROM lfs_object_access_policies WHERE lfs_object_oid`).
					WithArgs(notFoundOID).
					WillReturnError(pgx.ErrNoRows)
			},
			wantNil: true,
			wantErr: domain.ErrAccessPolicyNotFound,
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

			repo := postgres.NewAccessPolicyRepository(mock)
			ctx := context.Background()

			oid, err := domain.NewOID(tt.args.oid)
			if err != nil {
				t.Fatalf("OIDの作成に失敗しました: %v", err)
			}

			got, err := repo.FindByOID(ctx, oid)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("FindByOID() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("FindByOID() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("FindByOID() unexpected error = %v", err)
				}
				if tt.wantNil {
					if got != nil {
						t.Errorf("FindByOID() got = %v, want nil", got)
					}
				} else {
					if got == nil {
						t.Fatalf("FindByOID() got = nil, want non-nil")
					}
					if got.ID().Int64() != tt.wantID {
						t.Errorf("FindByOID() ID = %v, want %v", got.ID().Int64(), tt.wantID)
					}
					if got.OID().String() != tt.wantOID {
						t.Errorf("FindByOID() OID = %v, want %v", got.OID().String(), tt.wantOID)
					}
					if got.Repository().FullName() != tt.wantRepo {
						t.Errorf("FindByOID() Repository = %v, want %v", got.Repository().FullName(), tt.wantRepo)
					}
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestAccessPolicyRepositoryImpl_Save はSave処理のテーブルドリブンテスト
func TestAccessPolicyRepositoryImpl_Save(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	validOID := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	type args struct {
		oid        string
		repository string
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "正常系: 新規AccessPolicyの保存に成功",
			args: args{
				oid:        validOID,
				repository: "owner/repo",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO lfs_object_access_policies`).
					WithArgs(
						validOID,
						"owner/repo",
						pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
		{
			name: "正常系: 既存AccessPolicyの更新に成功（UPSERT）",
			args: args{
				oid:        validOID,
				repository: "new-owner/new-repo",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO lfs_object_access_policies`).
					WithArgs(
						validOID,
						"new-owner/new-repo",
						pgxmock.AnyArg(),
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

			repo := postgres.NewAccessPolicyRepository(mock)
			ctx := context.Background()

			oid, err := domain.NewOID(tt.args.oid)
			if err != nil {
				t.Fatalf("OIDの作成に失敗しました: %v", err)
			}

			repoIdentifier, err := domain.NewRepositoryIdentifier(tt.args.repository)
			if err != nil {
				t.Fatalf("RepositoryIdentifierの作成に失敗しました: %v", err)
			}

			policyID, err := domain.NewAccessPolicyID(0)
			if err != nil {
				t.Fatalf("AccessPolicyIDの作成に失敗しました: %v", err)
			}

			policy := domain.NewAccessPolicy(policyID, oid, repoIdentifier, fixedTime)

			err = repo.Save(ctx, policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestAccessPolicyRepositoryImpl_Delete はDelete処理のテーブルドリブンテスト
func TestAccessPolicyRepositoryImpl_Delete(t *testing.T) {
	validOID := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	notFoundOID := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

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
				oid: validOID,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM lfs_object_access_policies WHERE lfs_object_oid`).
					WithArgs(validOID).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: nil,
		},
		{
			name: "異常系: 存在しないOIDの削除",
			args: args{
				oid: notFoundOID,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM lfs_object_access_policies WHERE lfs_object_oid`).
					WithArgs(notFoundOID).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: domain.ErrAccessPolicyNotFound,
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

			repo := postgres.NewAccessPolicyRepository(mock)
			ctx := context.Background()

			oid, err := domain.NewOID(tt.args.oid)
			if err != nil {
				t.Fatalf("OIDの作成に失敗しました: %v", err)
			}

			err = repo.Delete(ctx, oid)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Delete() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
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

// TestRowToAccessPolicy_ConversionError は変換エラーケースのテスト
func TestAccessPolicyRepositoryImpl_FindByOID_ConversionError(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "異常系: 不正なOID形式でドメイン変換失敗",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "lfs_object_oid", "repository", "created_at"}).
					AddRow(
						int64(1),
						"invalid-oid", // 不正なOID
						"owner/repo",
						time.Now(),
					)
				mock.ExpectQuery(`SELECT id, lfs_object_oid, repository, created_at FROM lfs_object_access_policies WHERE lfs_object_oid`).
					WithArgs("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef").
					WillReturnRows(rows)
			},
			wantErr: true,
		},
		{
			name: "異常系: 不正なリポジトリ形式でドメイン変換失敗",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "lfs_object_oid", "repository", "created_at"}).
					AddRow(
						int64(1),
						"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
						"invalid-repo-format", // owner/repo形式ではない
						time.Now(),
					)
				mock.ExpectQuery(`SELECT id, lfs_object_oid, repository, created_at FROM lfs_object_access_policies WHERE lfs_object_oid`).
					WithArgs("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef").
					WillReturnRows(rows)
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

			repo := postgres.NewAccessPolicyRepository(mock)
			ctx := context.Background()

			validOID := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
			oid, err := domain.NewOID(validOID)
			if err != nil {
				t.Fatalf("OIDの作成に失敗しました: %v", err)
			}

			_, err = repo.FindByOID(ctx, oid)

			if (err != nil) != tt.wantErr {
				t.Errorf("FindByOID() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("期待されたモック呼び出しが行われませんでした: %v", err)
			}
		})
	}
}

// TestAccessPolicy_DomainObjectValues はAccessPolicyドメインオブジェクトの値検証テスト
func TestAccessPolicy_DomainObjectValues(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	validOID := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	tests := []struct {
		name       string
		oid        string
		repository string
		id         int64
		createdAt  time.Time
	}{
		{
			name:       "正常系: AccessPolicyの変換",
			oid:        validOID,
			repository: "owner/repo",
			id:         1,
			createdAt:  fixedTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, err := domain.NewOID(tt.oid)
			if err != nil {
				t.Fatalf("OIDの作成に失敗しました: %v", err)
			}

			repoIdentifier, err := domain.NewRepositoryIdentifier(tt.repository)
			if err != nil {
				t.Fatalf("RepositoryIdentifierの作成に失敗しました: %v", err)
			}

			policyID, err := domain.NewAccessPolicyID(tt.id)
			if err != nil {
				t.Fatalf("AccessPolicyIDの作成に失敗しました: %v", err)
			}

			policy := domain.NewAccessPolicy(policyID, oid, repoIdentifier, tt.createdAt)

			// Verify the domain object holds correct values
			if policy.ID().Int64() != tt.id {
				t.Errorf("ID = %v, want %v", policy.ID().Int64(), tt.id)
			}
			if policy.OID().String() != tt.oid {
				t.Errorf("OID = %v, want %v", policy.OID().String(), tt.oid)
			}
			if policy.Repository().FullName() != tt.repository {
				t.Errorf("Repository = %v, want %v", policy.Repository().FullName(), tt.repository)
			}
			if !cmp.Equal(policy.CreatedAt(), tt.createdAt) {
				t.Errorf("CreatedAt = %v, want %v", policy.CreatedAt(), tt.createdAt)
			}
		})
	}
}
