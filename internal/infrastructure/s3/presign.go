//go:generate mockgen -source=$GOFILE -destination=../../../tests/infrastructure/s3/mock_presign.go -package=s3
package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type PresignClientInterface interface {
	PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
	PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

type PresignClientFactory func(client *s3.Client) PresignClientInterface

func DefaultPresignClientFactory(client *s3.Client) PresignClientInterface {
	return s3.NewPresignClient(client)
}

const (
	DefaultPresignTTL = 15 * time.Minute
)

func (c *S3Client) GeneratePutURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if ttl == 0 {
		ttl = DefaultPresignTTL
	}

	presignClient := c.presignClientFactory(c.presignClient)
	presignResult, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign put object: %w", err)
	}

	return presignResult.URL, nil
}

func (c *S3Client) GenerateGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if ttl == 0 {
		ttl = DefaultPresignTTL
	}

	presignClient := c.presignClientFactory(c.presignClient)
	presignResult, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign get object: %w", err)
	}

	return presignResult.URL, nil
}
