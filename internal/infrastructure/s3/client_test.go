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
	"go.uber.org/mock/gomock"

	mocks3 "github.com/na2na-p/cargohold/tests/infrastructure/s3"
)

func TestS3Client_PutObject(t *testing.T) {
	type args struct {
		key           string
		content       string
		contentLength int64
	}
	tests := []struct {
		name      string
		setupMock func(ctrl *gomock.Controller) *mocks3.MockS3API
		args      args
		wantErr   bool
	}{
		{
			name: "正常系: オブジェクトのアップロード成功",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					PutObject(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&s3.PutObjectOutput{ETag: aws.String("mock-etag")}, nil)
				return mock
			},
			args: args{
				key:           "test/object.txt",
				content:       "Hello, S3!",
				contentLength: 10,
			},
			wantErr: false,
		},
		{
			name: "正常系: 大きなコンテンツ",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					PutObject(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&s3.PutObjectOutput{ETag: aws.String("mock-etag")}, nil)
				return mock
			},
			args: args{
				key:           "test/large.txt",
				content:       strings.Repeat("A", 10000),
				contentLength: 10000,
			},
			wantErr: false,
		},
		{
			name: "正常系: 特殊文字を含むキー",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					PutObject(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&s3.PutObjectOutput{ETag: aws.String("mock-etag")}, nil)
				return mock
			},
			args: args{
				key:           "test/special-chars_123.txt",
				content:       "Special content",
				contentLength: 15,
			},
			wantErr: false,
		},
		{
			name: "正常系: 空のコンテンツ",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					PutObject(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&s3.PutObjectOutput{ETag: aws.String("mock-etag")}, nil)
				return mock
			},
			args: args{
				key:           "test/empty.txt",
				content:       "",
				contentLength: 0,
			},
			wantErr: false,
		},
		{
			name: "異常系: S3エラー",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					PutObject(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("mock s3 error"))
				return mock
			},
			args: args{
				key:           "test/error.txt",
				content:       "content",
				contentLength: 7,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctx := context.Background()
			mockAPI := tt.setupMock(ctrl)
			client := NewMockS3Client(mockAPI, "test-bucket")

			err := client.PutObject(ctx, tt.args.key, strings.NewReader(tt.args.content), tt.args.contentLength)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutObject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestS3Client_GetObject(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name        string
		setupMock   func(ctrl *gomock.Controller) *mocks3.MockS3API
		args        args
		wantContent string
		wantErr     bool
	}{
		{
			name: "正常系: オブジェクトの取得成功",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				content := "Hello, S3!"
				mock.EXPECT().
					GetObject(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&s3.GetObjectOutput{
						Body:          io.NopCloser(strings.NewReader(content)),
						ContentLength: aws.Int64(int64(len(content))),
					}, nil)
				return mock
			},
			args: args{
				key: "test/get-object.txt",
			},
			wantContent: "Hello, S3!",
			wantErr:     false,
		},
		{
			name: "正常系: 大きなコンテンツの取得",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				largeContent := strings.Repeat("B", 5000)
				mock.EXPECT().
					GetObject(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&s3.GetObjectOutput{
						Body:          io.NopCloser(strings.NewReader(largeContent)),
						ContentLength: aws.Int64(int64(len(largeContent))),
					}, nil)
				return mock
			},
			args: args{
				key: "test/get-large.txt",
			},
			wantContent: strings.Repeat("B", 5000),
			wantErr:     false,
		},
		{
			name: "異常系: オブジェクトが存在しない",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					GetObject(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, NewMockNotFoundError())
				return mock
			},
			args: args{
				key: "test/non-existing.txt",
			},
			wantContent: "",
			wantErr:     true,
		},
		{
			name: "異常系: S3エラー",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					GetObject(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("mock s3 error"))
				return mock
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
			ctrl := gomock.NewController(t)
			ctx := context.Background()
			mockAPI := tt.setupMock(ctrl)
			client := NewMockS3Client(mockAPI, "test-bucket")

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

func TestS3Client_HeadObject(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name       string
		setupMock  func(ctrl *gomock.Controller) *mocks3.MockS3API
		args       args
		wantExists bool
		wantErr    bool
	}{
		{
			name: "正常系: オブジェクトが存在する",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					HeadObject(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&s3.HeadObjectOutput{ContentLength: aws.Int64(100)}, nil)
				return mock
			},
			args: args{
				key: "test/existing.txt",
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "正常系: オブジェクトが存在しない",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					HeadObject(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, NewMockNotFoundError())
				return mock
			},
			args: args{
				key: "test/non-existing.txt",
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "異常系: NotFound以外のS3エラー",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					HeadObject(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("mock internal server error"))
				return mock
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
			ctrl := gomock.NewController(t)
			ctx := context.Background()
			mockAPI := tt.setupMock(ctrl)
			client := NewMockS3Client(mockAPI, "test-bucket")

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
			ctrl := gomock.NewController(t)
			ctx := context.Background()

			storage := make(map[string]string)
			mockAPI := mocks3.NewMockS3API(ctrl)

			mockAPI.EXPECT().
				PutObject(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					body, err := io.ReadAll(params.Body)
					if err != nil {
						return nil, err
					}
					storage[*params.Key] = string(body)
					return &s3.PutObjectOutput{ETag: aws.String("mock-etag")}, nil
				})

			mockAPI.EXPECT().
				HeadObject(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
					content, exists := storage[*params.Key]
					if !exists {
						return nil, NewMockNotFoundError()
					}
					return &s3.HeadObjectOutput{ContentLength: aws.Int64(int64(len(content)))}, nil
				}).Times(2)

			mockAPI.EXPECT().
				GetObject(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					content, exists := storage[*params.Key]
					if !exists {
						return nil, NewMockNotFoundError()
					}
					return &s3.GetObjectOutput{
						Body:          io.NopCloser(strings.NewReader(content)),
						ContentLength: aws.Int64(int64(len(content))),
					}, nil
				})

			client := NewMockS3Client(mockAPI, "test-bucket")

			if err := client.PutObject(ctx, tt.args.key, strings.NewReader(tt.args.content), int64(len(tt.args.content))); err != nil {
				t.Fatalf("PutObject failed: %v", err)
			}

			exists, err := client.HeadObject(ctx, tt.args.key)
			if err != nil {
				t.Fatalf("HeadObject failed: %v", err)
			}
			if !exists {
				t.Error("HeadObject returned false for existing object")
			}

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
			ctrl := gomock.NewController(t)
			ctx := context.Background()

			storage := make(map[string]string)
			mockAPI := mocks3.NewMockS3API(ctrl)

			mockAPI.EXPECT().
				PutObject(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					body, err := io.ReadAll(params.Body)
					if err != nil {
						return nil, err
					}
					storage[*params.Key] = string(body)
					return &s3.PutObjectOutput{ETag: aws.String("mock-etag")}, nil
				}).Times(len(tt.objects))

			mockAPI.EXPECT().
				GetObject(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					content, exists := storage[*params.Key]
					if !exists {
						return nil, NewMockNotFoundError()
					}
					return &s3.GetObjectOutput{
						Body:          io.NopCloser(strings.NewReader(content)),
						ContentLength: aws.Int64(int64(len(content))),
					}, nil
				}).Times(len(tt.objects))

			client := NewMockS3Client(mockAPI, "test-bucket")

			for _, obj := range tt.objects {
				if err := client.PutObject(ctx, obj.key, strings.NewReader(obj.content), int64(len(obj.content))); err != nil {
					t.Fatalf("PutObject failed (key=%s): %v", obj.key, err)
				}
			}

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
	tests := []struct {
		name      string
		setupMock func(ctrl *gomock.Controller) *mocks3.MockS3API
		wantErr   error
	}{
		{
			name: "正常系: バケットが存在する",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&s3.HeadBucketOutput{}, nil)
				return mock
			},
			wantErr: nil,
		},
		{
			name: "異常系: バケットが存在しない",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("bucket not found"))
				return mock
			},
			wantErr: errors.New("failed to head bucket: bucket not found"),
		},
		{
			name: "異常系: S3接続エラー",
			setupMock: func(ctrl *gomock.Controller) *mocks3.MockS3API {
				mock := mocks3.NewMockS3API(ctrl)
				mock.EXPECT().
					HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("connection error"))
				return mock
			},
			wantErr: errors.New("failed to head bucket: connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctx := context.Background()
			mockAPI := tt.setupMock(ctrl)
			client := NewMockS3Client(mockAPI, "test-bucket")

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

func TestS3Client_ErrorTypes(t *testing.T) {
	t.Run("PutObject returns StorageError with OperationPut", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ctx := context.Background()
		mockAPI := mocks3.NewMockS3API(ctrl)
		mockAPI.EXPECT().
			PutObject(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("mock s3 error"))
		client := NewMockS3Client(mockAPI, "test-bucket")

		err := client.PutObject(ctx, "test/key", strings.NewReader("content"), 7)
		if err == nil {
			t.Fatal("want error, but got nil")
		}

		var storageErr *StorageError
		if !errors.As(err, &storageErr) {
			t.Fatalf("want StorageError, but got %T", err)
		}
		if storageErr.Operation != OperationPut {
			t.Errorf("want OperationPut, but got %s", storageErr.Operation)
		}
	})

	t.Run("GetObject returns StorageError with OperationGet", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ctx := context.Background()
		mockAPI := mocks3.NewMockS3API(ctrl)
		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("mock s3 error"))
		client := NewMockS3Client(mockAPI, "test-bucket")

		_, err := client.GetObject(ctx, "test/key")
		if err == nil {
			t.Fatal("want error, but got nil")
		}

		var storageErr *StorageError
		if !errors.As(err, &storageErr) {
			t.Fatalf("want StorageError, but got %T", err)
		}
		if storageErr.Operation != OperationGet {
			t.Errorf("want OperationGet, but got %s", storageErr.Operation)
		}
	})

	t.Run("HeadObject returns StorageError with OperationHead", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ctx := context.Background()
		mockAPI := mocks3.NewMockS3API(ctrl)
		mockAPI.EXPECT().
			HeadObject(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("mock internal server error"))
		client := NewMockS3Client(mockAPI, "test-bucket")

		_, err := client.HeadObject(ctx, "test/key")
		if err == nil {
			t.Fatal("want error, but got nil")
		}

		var storageErr *StorageError
		if !errors.As(err, &storageErr) {
			t.Fatalf("want StorageError, but got %T", err)
		}
		if storageErr.Operation != OperationHead {
			t.Errorf("want OperationHead, but got %s", storageErr.Operation)
		}
	})

	t.Run("IsStorageError returns true for wrapped StorageError", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ctx := context.Background()
		mockAPI := mocks3.NewMockS3API(ctrl)
		mockAPI.EXPECT().
			PutObject(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("mock s3 error"))
		client := NewMockS3Client(mockAPI, "test-bucket")

		err := client.PutObject(ctx, "test/key", strings.NewReader("content"), 7)
		if err == nil {
			t.Fatal("want error, but got nil")
		}

		if !IsStorageError(err) {
			t.Error("IsStorageError() = false, want true")
		}
	})
}
