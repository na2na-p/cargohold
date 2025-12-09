package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewAccessPolicy(t *testing.T) {
	oid, err := domain.NewOID(strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("failed to create OID: %v", err)
	}
	repo, err := domain.NewRepositoryIdentifier("owner/repo")
	if err != nil {
		t.Fatalf("failed to create RepositoryIdentifier: %v", err)
	}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	policyID, err := domain.NewAccessPolicyID(1)
	if err != nil {
		t.Fatalf("failed to create AccessPolicyID: %v", err)
	}

	type args struct {
		id         domain.AccessPolicyID
		oid        domain.OID
		repository *domain.RepositoryIdentifier
		createdAt  time.Time
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "正常系: AccessPolicyが正しく生成される",
			args: args{
				id:         policyID,
				oid:        oid,
				repository: repo,
				createdAt:  fixedTime,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.NewAccessPolicy(tt.args.id, tt.args.oid, tt.args.repository, tt.args.createdAt)

			if got.ID().Int64() != tt.args.id.Int64() {
				t.Errorf("ID() = %v, want %v", got.ID().Int64(), tt.args.id.Int64())
			}
			if got.OID().String() != tt.args.oid.String() {
				t.Errorf("OID() = %v, want %v", got.OID().String(), tt.args.oid.String())
			}
			if !got.Repository().Equals(tt.args.repository) {
				t.Errorf("Repository() = %v, want %v", got.Repository().FullName(), tt.args.repository.FullName())
			}
			if !got.CreatedAt().Equal(tt.args.createdAt) {
				t.Errorf("CreatedAt() = %v, want %v", got.CreatedAt(), tt.args.createdAt)
			}
		})
	}
}

func TestAccessPolicy_ID(t *testing.T) {
	oid, _ := domain.NewOID(strings.Repeat("a", 64))
	repo, _ := domain.NewRepositoryIdentifier("owner/repo")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	policyID, _ := domain.NewAccessPolicyID(123)

	tests := []struct {
		name   string
		policy *domain.AccessPolicy
		want   int64
	}{
		{
			name:   "正常系: IDが正しく取得できる",
			policy: domain.NewAccessPolicy(policyID, oid, repo, fixedTime),
			want:   123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.ID()
			if diff := cmp.Diff(tt.want, got.Int64()); diff != "" {
				t.Errorf("ID() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAccessPolicy_OID(t *testing.T) {
	oid1, _ := domain.NewOID(strings.Repeat("a", 64))
	oid2, _ := domain.NewOID(strings.Repeat("b", 64))
	repo, _ := domain.NewRepositoryIdentifier("owner/repo")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	policyID1, _ := domain.NewAccessPolicyID(1)
	policyID2, _ := domain.NewAccessPolicyID(2)

	tests := []struct {
		name   string
		policy *domain.AccessPolicy
		want   domain.OID
	}{
		{
			name:   "正常系: OIDが正しく取得できる",
			policy: domain.NewAccessPolicy(policyID1, oid1, repo, fixedTime),
			want:   oid1,
		},
		{
			name:   "正常系: 異なるOIDも正しく取得できる",
			policy: domain.NewAccessPolicy(policyID2, oid2, repo, fixedTime),
			want:   oid2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.OID()
			if diff := cmp.Diff(tt.want.String(), got.String()); diff != "" {
				t.Errorf("OID() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAccessPolicy_Repository(t *testing.T) {
	oid, _ := domain.NewOID(strings.Repeat("a", 64))
	repo1, _ := domain.NewRepositoryIdentifier("owner1/repo1")
	repo2, _ := domain.NewRepositoryIdentifier("owner2/repo2")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	policyID1, _ := domain.NewAccessPolicyID(1)
	policyID2, _ := domain.NewAccessPolicyID(2)

	tests := []struct {
		name   string
		policy *domain.AccessPolicy
		want   *domain.RepositoryIdentifier
	}{
		{
			name:   "正常系: Repositoryが正しく取得できる",
			policy: domain.NewAccessPolicy(policyID1, oid, repo1, fixedTime),
			want:   repo1,
		},
		{
			name:   "正常系: 異なるRepositoryも正しく取得できる",
			policy: domain.NewAccessPolicy(policyID2, oid, repo2, fixedTime),
			want:   repo2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.Repository()
			if !got.Equals(tt.want) {
				t.Errorf("Repository() = %v, want %v", got.FullName(), tt.want.FullName())
			}
		})
	}
}

func TestAccessPolicy_CreatedAt(t *testing.T) {
	oid, _ := domain.NewOID(strings.Repeat("a", 64))
	repo, _ := domain.NewRepositoryIdentifier("owner/repo")
	time1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	policyID1, _ := domain.NewAccessPolicyID(1)
	policyID2, _ := domain.NewAccessPolicyID(2)

	tests := []struct {
		name   string
		policy *domain.AccessPolicy
		want   time.Time
	}{
		{
			name:   "正常系: CreatedAtが正しく取得できる",
			policy: domain.NewAccessPolicy(policyID1, oid, repo, time1),
			want:   time1,
		},
		{
			name:   "正常系: 異なるCreatedAtも正しく取得できる",
			policy: domain.NewAccessPolicy(policyID2, oid, repo, time2),
			want:   time2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.CreatedAt()
			if !got.Equal(tt.want) {
				t.Errorf("CreatedAt() = %v, want %v", got, tt.want)
			}
		})
	}
}
