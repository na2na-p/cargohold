package s3_test

import (
	"context"
	"errors"
	"testing"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/go-cmp/cmp"

	s3client "github.com/na2na-p/cargohold/internal/infrastructure/s3"
)

type mockPresignClient struct {
	presignPutObjectFunc func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
	presignGetObjectFunc func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

func (m *mockPresignClient) PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	if m.presignPutObjectFunc != nil {
		return m.presignPutObjectFunc(ctx, params, optFns...)
	}
	return &v4.PresignedHTTPRequest{URL: "https://example.com/default"}, nil
}

func (m *mockPresignClient) PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	if m.presignGetObjectFunc != nil {
		return m.presignGetObjectFunc(ctx, params, optFns...)
	}
	return &v4.PresignedHTTPRequest{URL: "https://example.com/default"}, nil
}

func createMockPresignClientFactory(mock *mockPresignClient) s3client.PresignClientFactory {
	return func(_ *s3.Client) s3client.PresignClientInterface {
		return mock
	}
}

type mockS3API struct{}

func (m *mockS3API) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3API) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{}, nil
}

func (m *mockS3API) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return &s3.HeadObjectOutput{}, nil
}

func (m *mockS3API) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	return &s3.HeadBucketOutput{}, nil
}

func TestS3Client_GeneratePutURL(t *testing.T) {
	type fields struct {
		mockPresign func() *mockPresignClient
	}
	type args struct {
		key string
		ttl time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr error
	}{
		{
			name: "正常系: デフォルトTTLでPut用URLが生成される",
			fields: fields{
				mockPresign: func() *mockPresignClient {
					return &mockPresignClient{
						presignPutObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
							if *params.Bucket != "test-bucket" {
								t.Errorf("unexpected bucket: got %v, want test-bucket", *params.Bucket)
							}
							if *params.Key != "test/object.txt" {
								t.Errorf("unexpected key: got %v, want test/object.txt", *params.Key)
							}
							opts := &s3.PresignOptions{}
							for _, fn := range optFns {
								fn(opts)
							}
							if opts.Expires != 15*time.Minute {
								t.Errorf("unexpected expires: got %v, want %v", opts.Expires, 15*time.Minute)
							}
							return &v4.PresignedHTTPRequest{URL: "https://s3.example.com/test-bucket/test/object.txt?presigned"}, nil
						},
					}
				},
			},
			args: args{
				key: "test/object.txt",
				ttl: 0,
			},
			want:    "https://s3.example.com/test-bucket/test/object.txt?presigned",
			wantErr: nil,
		},
		{
			name: "正常系: カスタムTTLでPut用URLが生成される",
			fields: fields{
				mockPresign: func() *mockPresignClient {
					return &mockPresignClient{
						presignPutObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
							opts := &s3.PresignOptions{}
							for _, fn := range optFns {
								fn(opts)
							}
							if opts.Expires != 30*time.Minute {
								t.Errorf("unexpected expires: got %v, want %v", opts.Expires, 30*time.Minute)
							}
							return &v4.PresignedHTTPRequest{URL: "https://s3.example.com/custom-ttl"}, nil
						},
					}
				},
			},
			args: args{
				key: "test/custom.txt",
				ttl: 30 * time.Minute,
			},
			want:    "https://s3.example.com/custom-ttl",
			wantErr: nil,
		},
		{
			name: "異常系: Presign失敗時にエラーが返る",
			fields: fields{
				mockPresign: func() *mockPresignClient {
					return &mockPresignClient{
						presignPutObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
							return nil, errors.New("presign failed")
						},
					}
				},
			},
			args: args{
				key: "test/error.txt",
				ttl: 0,
			},
			want:    "",
			wantErr: errors.New("failed to presign put object: presign failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockAPI := &mockS3API{}
			mockPresign := tt.fields.mockPresign()
			factory := createMockPresignClientFactory(mockPresign)

			client := s3client.NewS3ClientWithPresignFactory(mockAPI, nil, "test-bucket", factory)

			got, err := client.GeneratePutURL(ctx, tt.args.key, tt.args.ttl)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("error mismatch: got %v, want %v", err.Error(), tt.wantErr.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestS3Client_GenerateGetURL(t *testing.T) {
	type fields struct {
		mockPresign func() *mockPresignClient
	}
	type args struct {
		key string
		ttl time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr error
	}{
		{
			name: "正常系: デフォルトTTLでGet用URLが生成される",
			fields: fields{
				mockPresign: func() *mockPresignClient {
					return &mockPresignClient{
						presignGetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
							if *params.Bucket != "test-bucket" {
								t.Errorf("unexpected bucket: got %v, want test-bucket", *params.Bucket)
							}
							if *params.Key != "test/download.txt" {
								t.Errorf("unexpected key: got %v, want test/download.txt", *params.Key)
							}
							opts := &s3.PresignOptions{}
							for _, fn := range optFns {
								fn(opts)
							}
							if opts.Expires != 15*time.Minute {
								t.Errorf("unexpected expires: got %v, want %v", opts.Expires, 15*time.Minute)
							}
							return &v4.PresignedHTTPRequest{URL: "https://s3.example.com/test-bucket/test/download.txt?presigned"}, nil
						},
					}
				},
			},
			args: args{
				key: "test/download.txt",
				ttl: 0,
			},
			want:    "https://s3.example.com/test-bucket/test/download.txt?presigned",
			wantErr: nil,
		},
		{
			name: "正常系: カスタムTTLでGet用URLが生成される",
			fields: fields{
				mockPresign: func() *mockPresignClient {
					return &mockPresignClient{
						presignGetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
							opts := &s3.PresignOptions{}
							for _, fn := range optFns {
								fn(opts)
							}
							if opts.Expires != 1*time.Hour {
								t.Errorf("unexpected expires: got %v, want %v", opts.Expires, 1*time.Hour)
							}
							return &v4.PresignedHTTPRequest{URL: "https://s3.example.com/custom-ttl-get"}, nil
						},
					}
				},
			},
			args: args{
				key: "test/custom-get.txt",
				ttl: 1 * time.Hour,
			},
			want:    "https://s3.example.com/custom-ttl-get",
			wantErr: nil,
		},
		{
			name: "異常系: PresignGetObject失敗時にエラーが返る",
			fields: fields{
				mockPresign: func() *mockPresignClient {
					return &mockPresignClient{
						presignGetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
							return nil, errors.New("presign get failed")
						},
					}
				},
			},
			args: args{
				key: "test/error-get.txt",
				ttl: 0,
			},
			want:    "",
			wantErr: errors.New("failed to presign get object: presign get failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockAPI := &mockS3API{}
			mockPresign := tt.fields.mockPresign()
			factory := createMockPresignClientFactory(mockPresign)

			client := s3client.NewS3ClientWithPresignFactory(mockAPI, nil, "test-bucket", factory)

			got, err := client.GenerateGetURL(ctx, tt.args.key, tt.args.ttl)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("error mismatch: got %v, want %v", err.Error(), tt.wantErr.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewS3ClientWithPresignFactory(t *testing.T) {
	type args struct {
		bucket  string
		factory s3client.PresignClientFactory
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常系: ファクトリを指定してクライアントが生成される",
			args: args{
				bucket: "custom-bucket",
				factory: func(_ *s3.Client) s3client.PresignClientInterface {
					return &mockPresignClient{}
				},
			},
		},
		{
			name: "正常系: ファクトリがnilの場合デフォルトファクトリが使用される",
			args: args{
				bucket:  "default-factory-bucket",
				factory: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := &mockS3API{}
			client := s3client.NewS3ClientWithPresignFactory(mockAPI, nil, tt.args.bucket, tt.args.factory)

			if client == nil {
				t.Fatal("NewS3ClientWithPresignFactory() returned nil")
			}
		})
	}
}
