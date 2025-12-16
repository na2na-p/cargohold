# Implement S3 Streaming Upload

## Backgrounds
- Current `S3Client.PutObject` implementation uses `io.ReadAll(body)` which loads entire file into memory
- Git LFS handles large files (hundreds of MB to GB), causing OOM risk
- AWS SDK v2 supports streaming uploads when `ContentLength` is provided
- Location: `internal/infrastructure/s3/client.go:77`
- Design validated with Modifius DDD defect analysis

## Goals
- Implement streaming upload to S3 without loading entire file into memory
- Use `LFSObject.Size()` from domain object to provide content length (DDD-compliant)
- Reduce memory footprint for large file uploads

## NonGoals
- Multipart upload implementation (future enhancement)
- Client-side direct S3 upload via presigned URLs (architectural change)

## Implementation Details

### Files to Modify

| File | Change |
|------|--------|
| `internal/usecase/external_interfaces.go` | Add `contentLength int64` parameter to `ObjectStorage.PutObject` |
| `internal/infrastructure/s3/client.go` | Remove `io.ReadAll`, use `ContentLength` in `PutObjectInput` |
| `internal/usecase/proxy_upload_usecase.go` | Pass `lfsObject.Size()` to `PutObject` call |
| `internal/infrastructure/s3/client_test.go` | Update tests for new signature |
| `internal/usecase/proxy_upload_usecase_test.go` | Update mock expectations for new signature |
| Mock files | Regenerate with `go generate ./...` |

### Key Code Changes

**ObjectStorage Interface** (`internal/usecase/external_interfaces.go`):
```go
type ObjectStorage interface {
    PutObject(ctx context.Context, key string, body io.Reader, contentLength int64) error
    GetObject(ctx context.Context, key string) (io.ReadCloser, error)
}
```

**S3Client.PutObject** (`internal/infrastructure/s3/client.go`):
```go
func (c *S3Client) PutObject(ctx context.Context, key string, body io.Reader, contentLength int64) error {
    _, err := c.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket:        aws.String(c.bucket),
        Key:           aws.String(key),
        Body:          body,
        ContentLength: aws.Int64(contentLength),
    })
    // ...
}
```

**ProxyUploadUseCase** (`internal/usecase/proxy_upload_usecase.go`):
```go
// In Execute method, change:
err = u.storage.PutObject(ctx, key, body)
// To:
err = u.storage.PutObject(ctx, key, body, lfsObject.Size())
```

## Checks
- [ ] Confirmed `CLAUDE.md` guidelines
- [ ] Ran defect analysis using MCP modifius before implementation
- [ ] Confirmed all implemented tests comply with conventions
- [ ] All tests passed (`go test ./...`)
- [ ] `golangci-lint` checks passed (`golangci-lint run`)
- [ ] Pushed development branch to remote
- [ ] Created pull request
