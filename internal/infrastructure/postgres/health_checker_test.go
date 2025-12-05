package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/na2na-p/cargohold/internal/infrastructure/postgres"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPostgresHealthChecker_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "正常系: 'postgres'が返る",
			want: "postgres",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("failed to create mock pool: %v", err)
			}
			defer mock.Close()

			checker := postgres.NewPostgresHealthChecker(mock)
			got := checker.Name()

			if got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostgresHealthChecker_Check(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name      string
		setupMock func(mock pgxmock.PgxPoolIface)
		args      args
		wantErr   bool
	}{
		{
			name: "正常系: Pingが成功した場合、nilが返る",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectPing()
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "異常系: Pingが失敗した場合、エラーが返る",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectPing().WillReturnError(errors.New("connection refused"))
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("failed to create mock pool: %v", err)
			}
			defer mock.Close()

			tt.setupMock(mock)

			checker := postgres.NewPostgresHealthChecker(mock)
			err = checker.Check(tt.args.ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("mock expectations not met: %v", err)
			}
		})
	}
}
