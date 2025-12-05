package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/handler/dto"
	"github.com/na2na-p/cargohold/internal/usecase"
)

func TestBatchRequestDTO_ToBatchRequest(t *testing.T) {
	type args struct {
		repo *domain.RepositoryIdentifier
	}

	repo1, _ := domain.NewRepositoryIdentifier("owner/repo")
	repo2, _ := domain.NewRepositoryIdentifier("user/project")
	repo3, _ := domain.NewRepositoryIdentifier("owner/repo")

	tests := []struct {
		name    string
		dto     dto.BatchRequestDTO
		args    args
		want    usecase.BatchRequest
		wantErr error
	}{
		{
			name: "正常系: downloadオペレーションで正しくBatchRequestに変換される",
			dto: dto.BatchRequestDTO{
				Operation: "download",
				Objects: []dto.RequestObjectDTO{
					{OID: "abc123", Size: 100},
				},
				Transfers: []string{"basic"},
				HashAlgo:  "sha256",
			},
			args: args{
				repo: repo1,
			},
			want: usecase.NewBatchRequest(
				domain.OperationDownload,
				[]usecase.RequestObject{usecase.NewRequestObject("abc123", 100)},
				[]string{"basic"},
				nil,
				"sha256",
				repo1,
			),
			wantErr: nil,
		},
		{
			name: "正常系: uploadオペレーションでrefありで正しく変換される",
			dto: dto.BatchRequestDTO{
				Operation: "upload",
				Objects: []dto.RequestObjectDTO{
					{OID: "def456", Size: 200},
					{OID: "ghi789", Size: 300},
				},
				Ref:      &dto.RefInfoDTO{Name: "refs/heads/main"},
				HashAlgo: "sha256",
			},
			args: args{
				repo: repo2,
			},
			want: func() usecase.BatchRequest {
				ref := usecase.NewRefInfo("refs/heads/main")
				return usecase.NewBatchRequest(
					domain.OperationUpload,
					[]usecase.RequestObject{
						usecase.NewRequestObject("def456", 200),
						usecase.NewRequestObject("ghi789", 300),
					},
					nil,
					&ref,
					"sha256",
					repo2,
				)
			}(),
			wantErr: nil,
		},
		{
			name: "異常系: 不正なオペレーションでエラーが返る",
			dto: dto.BatchRequestDTO{
				Operation: "invalid",
				Objects: []dto.RequestObjectDTO{
					{OID: "abc123", Size: 100},
				},
			},
			args: args{
				repo: repo3,
			},
			want:    usecase.BatchRequest{},
			wantErr: domain.ErrInvalidOperation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.dto.ToBatchRequest(tt.args.repo)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, but got nil", tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("want no error, but got %v", err)
			}

			if diff := cmp.Diff(tt.want.Operation(), got.Operation(), cmp.AllowUnexported(domain.Operation{})); diff != "" {
				t.Errorf("Operation mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(len(tt.want.Objects()), len(got.Objects())); diff != "" {
				t.Errorf("Objects count mismatch (-want +got):\n%s", diff)
			}
			for i, wantObj := range tt.want.Objects() {
				gotObj := got.Objects()[i]
				if wantObj.OID() != gotObj.OID() {
					t.Errorf("Object[%d].OID mismatch: want %s, got %s", i, wantObj.OID(), gotObj.OID())
				}
				if wantObj.Size() != gotObj.Size() {
					t.Errorf("Object[%d].Size mismatch: want %d, got %d", i, wantObj.Size(), gotObj.Size())
				}
			}
			if diff := cmp.Diff(tt.want.HashAlgo(), got.HashAlgo()); diff != "" {
				t.Errorf("HashAlgo mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBatchRequestDTO_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    dto.BatchRequestDTO
		wantErr bool
	}{
		{
			name: "正常系: 基本的なJSONがパースできる",
			json: `{"operation":"download","objects":[{"oid":"abc123","size":100}],"transfers":["basic"],"hash_algo":"sha256"}`,
			want: dto.BatchRequestDTO{
				Operation: "download",
				Objects:   []dto.RequestObjectDTO{{OID: "abc123", Size: 100}},
				Transfers: []string{"basic"},
				HashAlgo:  "sha256",
			},
			wantErr: false,
		},
		{
			name: "正常系: refを含むJSONがパースできる",
			json: `{"operation":"upload","objects":[{"oid":"def456","size":200}],"ref":{"name":"refs/heads/main"}}`,
			want: dto.BatchRequestDTO{
				Operation: "upload",
				Objects:   []dto.RequestObjectDTO{{OID: "def456", Size: 200}},
				Ref:       &dto.RefInfoDTO{Name: "refs/heads/main"},
			},
			wantErr: false,
		},
		{
			name:    "異常系: 不正なJSONでエラーが返る",
			json:    `{invalid json}`,
			want:    dto.BatchRequestDTO{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got dto.BatchRequestDTO
			err := json.Unmarshal([]byte(tt.json), &got)

			if tt.wantErr {
				if err == nil {
					t.Fatal("want error, but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("want no error, but got %v", err)
			}

			if diff := cmp.Diff(tt.want.Operation, got.Operation); diff != "" {
				t.Errorf("Operation mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(len(tt.want.Objects), len(got.Objects)); diff != "" {
				t.Errorf("Objects count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
