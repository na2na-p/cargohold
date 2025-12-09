package s3

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/go-cmp/cmp"
)

// TestS3Client_PutObject はPutObjectのテーブルドリブンテスト
func TestS3Client_PutObject(t *testing.T) {
	type fields struct {
		mockAPI func() S3API
	}
	type args struct {
		key     string
		content string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "正常系: オブジェクトのアップロード成功",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						PutObjectFunc: createMockPutObjectSuccess(),
					}
				},
			},
			args: args{
				key:     "test/object.txt",
				content: "Hello, S3!",
			},
			wantErr: false,
		},
		{
			name: "正常系: 大きなコンテンツ",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						PutObjectFunc: createMockPutObjectSuccess(),
					}
				},
			},
			args: args{
				key:     "test/large.txt",
				content: strings.Repeat("A", 10000),
			},
			wantErr: false,
		},
		{
			name: "正常系: 特殊文字を含むキー",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						PutObjectFunc: createMockPutObjectSuccess(),
					}
				},
			},
			args: args{
				key:     "test/special-chars_123.txt",
				content: "Special content",
			},
			wantErr: false,
		},
		{
			name: "正常系: 空のコンテンツ",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						PutObjectFunc: createMockPutObjectSuccess(),
					}
				},
			},
			args: args{
				key:     "test/empty.txt",
				content: "",
			},
			wantErr: false,
		},
		{
			name: "異常系: S3エラー",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						PutObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
							return nil, errors.New("mock s3 error")
						},
					}
				},
			},
			args: args{
				key:     "test/error.txt",
				content: "content",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client := NewMockS3Client(tt.fields.mockAPI(), "test-bucket")

			err := client.PutObject(ctx, tt.args.key, strings.NewReader(tt.args.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("PutObject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestS3Client_GetObject はGetObjectのテーブルドリブンテスト
func TestS3Client_GetObject(t *testing.T) {
	type fields struct {
		mockAPI func() S3API
	}
	type args struct {
		key string
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantContent string
		wantErr     bool
	}{
		{
			name: "正常系: オブジェクトの取得成功",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						GetObjectFunc: createMockGetObjectSuccess("Hello, S3!"),
					}
				},
			},
			args: args{
				key: "test/get-object.txt",
			},
			wantContent: "Hello, S3!",
			wantErr:     false,
		},
		{
			name: "正常系: 大きなコンテンツの取得",
			fields: fields{
				mockAPI: func() S3API {
					largeContent := strings.Repeat("B", 5000)
					return &MockS3API{
						GetObjectFunc: createMockGetObjectSuccess(largeContent),
					}
				},
			},
			args: args{
				key: "test/get-large.txt",
			},
			wantContent: strings.Repeat("B", 5000),
			wantErr:     false,
		},
		{
			name: "異常系: オブジェクトが存在しない",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						GetObjectFunc: createMockGetObjectNotFound(),
					}
				},
			},
			args: args{
				key: "test/non-existing.txt",
			},
			wantContent: "",
			wantErr:     true,
		},
		{
			name: "異常系: S3エラー",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						GetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
							return nil, errors.New("mock s3 error")
						},
					}
				},
			},
			args: args{
				key: "test/error.txt",
			},
			wantContent: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client := NewMockS3Client(tt.fields.mockAPI(), "test-bucket")

			reader, err := client.GetObject(ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				defer func() { _ = reader.Close() }()
				content, err := io.ReadAll(reader)
				if err != nil {
					t.Fatalf("コンテンツの読み込みに失敗しました: %v", err)
				}
				if diff := cmp.Diff(tt.wantContent, string(content)); diff != "" {
					t.Errorf("content mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// TestS3Client_HeadObject はHeadObjectのテーブルドリブンテスト
func TestS3Client_HeadObject(t *testing.T) {
	type fields struct {
		mockAPI func() S3API
	}
	type args struct {
		key string
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantExists bool
		wantErr    bool
	}{
		{
			name: "正常系: オブジェクトが存在する",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						HeadObjectFunc: createMockHeadObjectSuccess(),
					}
				},
			},
			args: args{
				key: "test/existing.txt",
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "正常系: オブジェクトが存在しない",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						HeadObjectFunc: createMockHeadObjectNotFound(),
					}
				},
			},
			args: args{
				key: "test/non-existing.txt",
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "異常系: NotFound以外のS3エラー",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						HeadObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
							return nil, errors.New("mock internal server error")
						},
					}
				},
			},
			args: args{
				key: "test/error.txt",
			},
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client := NewMockS3Client(tt.fields.mockAPI(), "test-bucket")

			exists, err := client.HeadObject(ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("HeadObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.wantExists, exists); diff != "" {
				t.Errorf("HeadObject() exists mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestS3Client_Integration は統合的な動作確認テスト（Put→Head→Get）
func TestS3Client_Integration(t *testing.T) {
	type args struct {
		key     string
		content string
	}
	key1, err := GenerateStorageKey("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "sha256")
	if err != nil {
		t.Fatalf("GenerateStorageKey failed for key1: %v", err)
	}
	key2, err := GenerateStorageKey("1111111111111111111111111111111111111111111111111111111111111111", "sha256")
	if err != nil {
		t.Fatalf("GenerateStorageKey failed for key2: %v", err)
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常系: 標準的なワークフロー",
			args: args{
				key:     key1,
				content: "test content for integration",
			},
		},
		{
			name: "正常系: 大きなコンテンツのワークフロー",
			args: args{
				key:     key2,
				content: strings.Repeat("X", 20000),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// モックストレージ（インメモリ）
			storage := make(map[string]string)

			mockAPI := &MockS3API{
				PutObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					// 実際にReadして保存
					body, err := io.ReadAll(params.Body)
					if err != nil {
						return nil, err
					}
					storage[*params.Key] = string(body)
					return &s3.PutObjectOutput{ETag: aws.String("mock-etag")}, nil
				},
				GetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					content, exists := storage[*params.Key]
					if !exists {
						return nil, NewMockNotFoundError()
					}
					return &s3.GetObjectOutput{
						Body:          io.NopCloser(strings.NewReader(content)),
						ContentLength: aws.Int64(int64(len(content))),
					}, nil
				},
				HeadObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
					content, exists := storage[*params.Key]
					if !exists {
						return nil, NewMockNotFoundError()
					}
					return &s3.HeadObjectOutput{
						ContentLength: aws.Int64(int64(len(content))),
					}, nil
				},
			}

			client := NewMockS3Client(mockAPI, "test-bucket")

			// Put
			if err := client.PutObject(ctx, tt.args.key, strings.NewReader(tt.args.content)); err != nil {
				t.Fatalf("PutObject failed: %v", err)
			}

			// Head (exists)
			exists, err := client.HeadObject(ctx, tt.args.key)
			if err != nil {
				t.Fatalf("HeadObject failed: %v", err)
			}
			if !exists {
				t.Error("HeadObject returned false for existing object")
			}

			// Get
			reader, err := client.GetObject(ctx, tt.args.key)
			if err != nil {
				t.Fatalf("GetObject failed: %v", err)
			}
			defer func() { _ = reader.Close() }()

			data, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("failed to read content: %v", err)
			}

			if diff := cmp.Diff(tt.args.content, string(data)); diff != "" {
				t.Errorf("content mismatch (-want +got):\n%s", diff)
			}

			// Head (not exists)
			exists, err = client.HeadObject(ctx, "non-existing-key")
			if err != nil {
				t.Fatalf("HeadObject failed: %v", err)
			}
			if exists {
				t.Error("HeadObject returned true for non-existing object")
			}
		})
	}
}

// TestS3Client_MultipleObjects は複数オブジェクトの保存と取得のテーブルドリブンテスト
func TestS3Client_MultipleObjects(t *testing.T) {
	type objectData struct {
		key     string
		content string
	}

	tests := []struct {
		name    string
		objects []objectData
	}{
		{
			name: "正常系: 複数オブジェクトの保存と取得",
			objects: []objectData{
				{"objects/sha256/11/11/111111.txt", "content 1"},
				{"objects/sha256/22/22/222222.txt", "content 2"},
				{"objects/sha512/33/33/333333.txt", "content 3"},
			},
		},
		{
			name: "正常系: 大量オブジェクトの保存と取得",
			objects: []objectData{
				{"test/batch/obj1.txt", "data 1"},
				{"test/batch/obj2.txt", "data 2"},
				{"test/batch/obj3.txt", "data 3"},
				{"test/batch/obj4.txt", "data 4"},
				{"test/batch/obj5.txt", "data 5"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// モックストレージ（インメモリ）
			storage := make(map[string]string)

			mockAPI := &MockS3API{
				PutObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					body, err := io.ReadAll(params.Body)
					if err != nil {
						return nil, err
					}
					storage[*params.Key] = string(body)
					return &s3.PutObjectOutput{ETag: aws.String("mock-etag")}, nil
				},
				GetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					content, exists := storage[*params.Key]
					if !exists {
						return nil, NewMockNotFoundError()
					}
					return &s3.GetObjectOutput{
						Body:          io.NopCloser(strings.NewReader(content)),
						ContentLength: aws.Int64(int64(len(content))),
					}, nil
				},
			}

			client := NewMockS3Client(mockAPI, "test-bucket")

			// 全てのオブジェクトをアップロード
			for _, obj := range tt.objects {
				if err := client.PutObject(ctx, obj.key, strings.NewReader(obj.content)); err != nil {
					t.Fatalf("PutObject failed (key=%s): %v", obj.key, err)
				}
			}

			// 全てのオブジェクトが取得できることを確認
			for _, obj := range tt.objects {
				func() {
					reader, err := client.GetObject(ctx, obj.key)
					if err != nil {
						t.Fatalf("GetObject failed (key=%s): %v", obj.key, err)
					}
					defer func() { _ = reader.Close() }()

					data, err := io.ReadAll(reader)
					if err != nil {
						t.Fatalf("failed to read content (key=%s): %v", obj.key, err)
					}

					if diff := cmp.Diff(obj.content, string(data)); diff != "" {
						t.Errorf("content mismatch (key=%s) (-want +got):\n%s", obj.key, diff)
					}
				}()
			}
		})
	}
}

// TestNewS3Client はNewS3Clientのテスト
func TestNewS3Client(t *testing.T) {
	type args struct {
		bucket string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常系: クライアントが正しく生成される",
			args: args{
				bucket: "test-bucket",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// nilのclientでも構造体は作成できる（presignClientのテストは別途実施）
			client := NewS3Client(nil, tt.args.bucket)

			if client == nil {
				t.Fatal("NewS3Client() returned nil")
			}

			if diff := cmp.Diff(tt.args.bucket, client.bucket); diff != "" {
				t.Errorf("bucket mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestS3Client_HeadBucket(t *testing.T) {
	type fields struct {
		mockAPI func() S3API
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr error
	}{
		{
			name: "正常系: バケットが存在する",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						HeadBucketFunc: func(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
							return &s3.HeadBucketOutput{}, nil
						},
					}
				},
			},
			wantErr: nil,
		},
		{
			name: "異常系: バケットが存在しない",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						HeadBucketFunc: func(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
							return nil, errors.New("bucket not found")
						},
					}
				},
			},
			wantErr: errors.New("failed to head bucket: bucket not found"),
		},
		{
			name: "異常系: S3接続エラー",
			fields: fields{
				mockAPI: func() S3API {
					return &MockS3API{
						HeadBucketFunc: func(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
							return nil, errors.New("connection error")
						},
					}
				},
			},
			wantErr: errors.New("failed to head bucket: connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client := NewMockS3Client(tt.fields.mockAPI(), "test-bucket")

			err := client.HeadBucket(ctx)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				if diff := cmp.Diff(tt.wantErr.Error(), err.Error()); diff != "" {
					t.Errorf("HeadBucket() error mismatch (-want +got):\n%s", diff)
				}
			} else {
				if err != nil {
					t.Fatalf("want no error, but got %v", err)
				}
			}
		})
	}
}
