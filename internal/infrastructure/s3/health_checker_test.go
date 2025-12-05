package s3

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
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
			mockAPI := &MockS3API{}
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
		setupMock func() *MockS3API
		args      args
		wantErr   bool
	}{
		{
			name: "正常系: HeadBucketが成功した場合、nilが返る",
			setupMock: func() *MockS3API {
				return &MockS3API{
					HeadBucketFunc: func(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
						return &s3.HeadBucketOutput{}, nil
					},
				}
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "異常系: HeadBucketが失敗した場合、エラーが返る",
			setupMock: func() *MockS3API {
				return &MockS3API{
					HeadBucketFunc: func(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
						return nil, errors.New("bucket not found")
					},
				}
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := tt.setupMock()
			client := NewMockS3Client(mockAPI, "test-bucket")
			checker := NewS3HealthChecker(client)

			err := checker.Check(tt.args.ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
