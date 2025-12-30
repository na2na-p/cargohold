package s3

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/mock/gomock"

	mocks3 "github.com/na2na-p/cargohold/tests/infrastructure/s3"
)

func TestS3HealthChecker_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "正常系: 's3'が返る",
			want: "s3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockAPI := mocks3.NewMockS3API(ctrl)
			client := NewMockS3Client(mockAPI, "test-bucket")
			checker := NewS3HealthChecker(client)

			got := checker.Name()

			if got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestS3HealthChecker_Check(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name      string
		setupMock func(ctrl *gomock.Controller) *mocks3.MockS3API
		args      args
		wantErr   bool
	}{
		{
			name: "正常系: HeadBucketが成功した場合、nilが返る",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&s3.HeadBucketOutput{}, nil)
				return mock
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "異常系: HeadBucketが失敗した場合、エラーが返る",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("bucket not found"))
				return mock
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockAPI := tt.setupMock(ctrl)
			client := NewMockS3Client(mockAPI, "test-bucket")
			checker := NewS3HealthChecker(client)

			err := checker.Check(tt.args.ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
