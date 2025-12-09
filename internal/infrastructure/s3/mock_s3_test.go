package s3

import (
	"context"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// MockS3API はS3APIのモック実装
type MockS3API struct {
	PutObjectFunc  func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObjectFunc  func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObjectFunc func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	HeadBucketFunc func(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

// PutObject はPutObjectのモック実装
func (m *MockS3API) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.PutObjectFunc != nil {
		return m.PutObjectFunc(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

// GetObject はGetObjectのモック実装
func (m *MockS3API) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.GetObjectFunc != nil {
		return m.GetObjectFunc(ctx, params, optFns...)
	}
	return &s3.GetObjectOutput{}, nil
}

// HeadObject はHeadObjectのモック実装
func (m *MockS3API) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.HeadObjectFunc != nil {
		return m.HeadObjectFunc(ctx, params, optFns...)
	}
	return &s3.HeadObjectOutput{}, nil
}

// HeadBucket はHeadBucketのモック実装
func (m *MockS3API) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	if m.HeadBucketFunc != nil {
		return m.HeadBucketFunc(ctx, params, optFns...)
	}
	return &s3.HeadBucketOutput{}, nil
}

// mockNotFoundError はNotFoundエラーのモック実装
type mockNotFoundError struct {
	code    string
	message string
}

func (e *mockNotFoundError) Error() string {
	return e.message
}

func (e *mockNotFoundError) ErrorCode() string {
	return e.code
}

func (e *mockNotFoundError) ErrorMessage() string {
	return e.message
}

func (e *mockNotFoundError) ErrorFault() smithy.ErrorFault {
	return smithy.FaultClient
}

// NewMockNotFoundError はNotFoundエラーを作成します
func NewMockNotFoundError() error {
	return &mockNotFoundError{
		code:    "NotFound",
		message: "object not found",
	}
}

// NewMockS3Client はテスト用のモックS3クライアントを作成します
func NewMockS3Client(mockAPI S3API, bucket string) *S3Client {
	return &S3Client{
		client:               mockAPI,
		presignClient:        nil,
		presignClientFactory: DefaultPresignClientFactory,
		bucket:               bucket,
	}
}

// createMockPutObjectSuccess は正常なPutObjectのモック関数を返します
func createMockPutObjectSuccess() func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
		return &s3.PutObjectOutput{
			ETag: aws.String("mock-etag"),
		}, nil
	}
}

// createMockGetObjectSuccess は正常なGetObjectのモック関数を返します（コンテンツ指定）
func createMockGetObjectSuccess(content string) func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
		return &s3.GetObjectOutput{
			Body:          io.NopCloser(strings.NewReader(content)),
			ContentLength: aws.Int64(int64(len(content))),
		}, nil
	}
}

// createMockGetObjectNotFound は存在しないオブジェクトのGetObjectモック関数を返します
func createMockGetObjectNotFound() func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
		return nil, NewMockNotFoundError()
	}
}

// createMockHeadObjectSuccess は正常なHeadObjectのモック関数を返します（存在する）
func createMockHeadObjectSuccess() func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
		return &s3.HeadObjectOutput{
			ContentLength: aws.Int64(100),
		}, nil
	}
}

// createMockHeadObjectNotFound は存在しないオブジェクトのHeadObjectモック関数を返します
func createMockHeadObjectNotFound() func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
		return nil, NewMockNotFoundError()
	}
}
