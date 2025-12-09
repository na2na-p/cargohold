package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/na2na-p/cargohold/internal/usecase"
	mock_usecase "github.com/na2na-p/cargohold/tests/usecase"
	"go.uber.org/mock/gomock"
)

func TestReadinessUseCase_Execute(t *testing.T) {
	type fields struct {
		checkers func(ctrl *gomock.Controller) []usecase.HealthChecker
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "正常系: すべてのヘルスチェッカーが正常な場合、nilが返る",
			fields: fields{
				checkers: func(ctrl *gomock.Controller) []usecase.HealthChecker {
					postgres := mock_usecase.NewMockHealthChecker(ctrl)
					postgres.EXPECT().Name().Return("postgres").AnyTimes()
					postgres.EXPECT().Check(gomock.Any()).Return(nil)

					redis := mock_usecase.NewMockHealthChecker(ctrl)
					redis.EXPECT().Name().Return("redis").AnyTimes()
					redis.EXPECT().Check(gomock.Any()).Return(nil)

					s3 := mock_usecase.NewMockHealthChecker(ctrl)
					s3.EXPECT().Name().Return("s3").AnyTimes()
					s3.EXPECT().Check(gomock.Any()).Return(nil)

					return []usecase.HealthChecker{postgres, redis, s3}
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: nil,
		},
		{
			name: "正常系: チェッカーが0個の場合、nilが返る",
			fields: fields{
				checkers: func(ctrl *gomock.Controller) []usecase.HealthChecker {
					return []usecase.HealthChecker{}
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: nil,
		},
		{
			name: "異常系: 1つのチェッカーが失敗した場合、エラーが返る",
			fields: fields{
				checkers: func(ctrl *gomock.Controller) []usecase.HealthChecker {
					postgres := mock_usecase.NewMockHealthChecker(ctrl)
					postgres.EXPECT().Name().Return("postgres").AnyTimes()
					postgres.EXPECT().Check(gomock.Any()).Return(nil)

					redis := mock_usecase.NewMockHealthChecker(ctrl)
					redis.EXPECT().Name().Return("redis").AnyTimes()
					redis.EXPECT().Check(gomock.Any()).Return(errors.New("connection refused"))

					s3 := mock_usecase.NewMockHealthChecker(ctrl)
					s3.EXPECT().Name().Return("s3").AnyTimes()
					s3.EXPECT().Check(gomock.Any()).Return(nil)

					return []usecase.HealthChecker{postgres, redis, s3}
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: usecase.ErrHealthCheckFailed,
		},
		{
			name: "異常系: 複数のチェッカーが失敗した場合、エラーが返る",
			fields: fields{
				checkers: func(ctrl *gomock.Controller) []usecase.HealthChecker {
					postgres := mock_usecase.NewMockHealthChecker(ctrl)
					postgres.EXPECT().Name().Return("postgres").AnyTimes()
					postgres.EXPECT().Check(gomock.Any()).Return(errors.New("connection timeout"))

					redis := mock_usecase.NewMockHealthChecker(ctrl)
					redis.EXPECT().Name().Return("redis").AnyTimes()
					redis.EXPECT().Check(gomock.Any()).Return(errors.New("connection refused"))

					return []usecase.HealthChecker{postgres, redis}
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: usecase.ErrHealthCheckFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			uc := usecase.NewReadinessUseCase(tt.fields.checkers(ctrl)...)

			err := uc.Execute(tt.args.ctx)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error mismatch: want %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}
		})
	}
}

func TestReadinessUseCase_ExecuteDetails(t *testing.T) {
	type fields struct {
		checkers func(ctrl *gomock.Controller) []usecase.HealthChecker
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []usecase.HealthCheckResult
		wantErr error
	}{
		{
			name: "正常系: すべてのヘルスチェッカーが正常な場合、全結果が返る",
			fields: fields{
				checkers: func(ctrl *gomock.Controller) []usecase.HealthChecker {
					postgres := mock_usecase.NewMockHealthChecker(ctrl)
					postgres.EXPECT().Name().Return("postgres").AnyTimes()
					postgres.EXPECT().Check(gomock.Any()).Return(nil)

					redis := mock_usecase.NewMockHealthChecker(ctrl)
					redis.EXPECT().Name().Return("redis").AnyTimes()
					redis.EXPECT().Check(gomock.Any()).Return(nil)

					return []usecase.HealthChecker{postgres, redis}
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want: []usecase.HealthCheckResult{
				{Name: "postgres", Healthy: true, Error: nil},
				{Name: "redis", Healthy: true, Error: nil},
			},
			wantErr: nil,
		},
		{
			name: "異常系: 一部が失敗した場合、失敗情報を含む結果とエラーが返る",
			fields: fields{
				checkers: func(ctrl *gomock.Controller) []usecase.HealthChecker {
					postgres := mock_usecase.NewMockHealthChecker(ctrl)
					postgres.EXPECT().Name().Return("postgres").AnyTimes()
					postgres.EXPECT().Check(gomock.Any()).Return(nil)

					redis := mock_usecase.NewMockHealthChecker(ctrl)
					redis.EXPECT().Name().Return("redis").AnyTimes()
					redis.EXPECT().Check(gomock.Any()).Return(errors.New("connection refused"))

					return []usecase.HealthChecker{postgres, redis}
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want: []usecase.HealthCheckResult{
				{Name: "postgres", Healthy: true, Error: nil},
				{Name: "redis", Healthy: false, Error: errors.New("connection refused")},
			},
			wantErr: usecase.ErrHealthCheckFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			uc := usecase.NewReadinessUseCase(tt.fields.checkers(ctrl)...)

			got, err := uc.ExecuteDetails(tt.args.ctx)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error mismatch: want %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}

			// Compare results
			if len(got) != len(tt.want) {
				t.Fatalf("result count mismatch: want %d, got %d", len(tt.want), len(got))
			}

			for i, want := range tt.want {
				if got[i].Name != want.Name {
					t.Errorf("result[%d].Name mismatch: want %s, got %s", i, want.Name, got[i].Name)
				}
				if got[i].Healthy != want.Healthy {
					t.Errorf("result[%d].Healthy mismatch: want %v, got %v", i, want.Healthy, got[i].Healthy)
				}
				if (got[i].Error == nil) != (want.Error == nil) {
					t.Errorf("result[%d].Error mismatch: want %v, got %v", i, want.Error, got[i].Error)
				}
				if got[i].Error != nil && want.Error != nil {
					if got[i].Error.Error() != want.Error.Error() {
						t.Errorf("result[%d].Error message mismatch: want %s, got %s", i, want.Error.Error(), got[i].Error.Error())
					}
				}
			}
		})
	}
}
