package s3

import (
	"context"
	"fmt"
)

// S3HealthChecker はS3のヘルスチェックを行う
type S3HealthChecker struct {
	client *S3Client
}

// NewS3HealthChecker は新しいS3HealthCheckerを生成する
func NewS3HealthChecker(client *S3Client) *S3HealthChecker {
	return &S3HealthChecker{
		client: client,
	}
}

// Name はチェッカーの名前を返す
func (c *S3HealthChecker) Name() string {
	return "s3"
}

// Check はS3へのヘルスチェックを実行する
func (c *S3HealthChecker) Check(ctx context.Context) error {
	if err := c.client.HeadBucket(ctx); err != nil {
		return fmt.Errorf("s3 health check failed: %w", err)
	}
	return nil
}
