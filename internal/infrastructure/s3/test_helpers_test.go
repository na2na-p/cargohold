package s3

import (
	"github.com/aws/smithy-go"
)

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

func NewMockNotFoundError() error {
	return &mockNotFoundError{
		code:    "NotFound",
		message: "object not found",
	}
}

func NewMockS3Client(mockAPI S3API, bucket string) *S3Client {
	return &S3Client{
		client:               mockAPI,
		presignClient:        nil,
		presignClientFactory: DefaultPresignClientFactory,
		bucket:               bucket,
	}
}
