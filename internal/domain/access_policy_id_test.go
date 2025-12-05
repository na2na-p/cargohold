package domain_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewAccessPolicyID(t *testing.T) {
	type args struct {
		value int64
	}
	tests := []struct {
		name       string
		args       args
		want       int64
		wantErr    bool
		wantErrVal error
	}{
		{
			name: "正常系: 正の値",
			args: args{
				value: 1,
			},
			want:       1,
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "正常系: ゼロ",
			args: args{
				value: 0,
			},
			want:       0,
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "正常系: 大きな正の値",
			args: args{
				value: 9223372036854775807,
			},
			want:       9223372036854775807,
			wantErr:    false,
			wantErrVal: nil,
		},
		{
			name: "異常系: 負の値",
			args: args{
				value: -1,
			},
			want:       0,
			wantErr:    true,
			wantErrVal: domain.ErrInvalidAccessPolicyID,
		},
		{
			name: "異常系: 大きな負の値",
			args: args{
				value: -9223372036854775808,
			},
			want:       0,
			wantErr:    true,
			wantErrVal: domain.ErrInvalidAccessPolicyID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := domain.NewAccessPolicyID(tt.args.value)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewAccessPolicyID() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErrVal) {
					t.Errorf("NewAccessPolicyID() error = %v, want %v", err, tt.wantErrVal)
				}
			} else {
				if err != nil {
					t.Errorf("NewAccessPolicyID() error = %v, wantErr %v", err, tt.wantErr)
				}
				if diff := cmp.Diff(tt.want, id.Int64()); diff != "" {
					t.Errorf("AccessPolicyID.Int64() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestAccessPolicyID_Int64(t *testing.T) {
	tests := []struct {
		name  string
		value int64
		want  int64
	}{
		{
			name:  "正常系: 正の値が正しく取得できる",
			value: 123,
			want:  123,
		},
		{
			name:  "正常系: ゼロが正しく取得できる",
			value: 0,
			want:  0,
		},
		{
			name:  "正常系: 大きな値が正しく取得できる",
			value: 9223372036854775807,
			want:  9223372036854775807,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := domain.NewAccessPolicyID(tt.value)
			if err != nil {
				t.Fatalf("NewAccessPolicyID() error = %v", err)
			}
			got := id.Int64()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("AccessPolicyID.Int64() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAccessPolicyID_Immutability(t *testing.T) {
	originalValue := int64(42)
	id, err := domain.NewAccessPolicyID(originalValue)
	if err != nil {
		t.Fatalf("NewAccessPolicyID() error = %v", err)
	}

	firstGet := id.Int64()
	secondGet := id.Int64()

	if firstGet != secondGet {
		t.Errorf("AccessPolicyID should be immutable: first = %v, second = %v", firstGet, secondGet)
	}

	if firstGet != originalValue {
		t.Errorf("AccessPolicyID value should not change: got = %v, want = %v", firstGet, originalValue)
	}
}

func TestAccessPolicyID_ValueEquality(t *testing.T) {
	id1, err := domain.NewAccessPolicyID(100)
	if err != nil {
		t.Fatalf("NewAccessPolicyID() error = %v", err)
	}
	id2, err := domain.NewAccessPolicyID(100)
	if err != nil {
		t.Fatalf("NewAccessPolicyID() error = %v", err)
	}
	id3, err := domain.NewAccessPolicyID(200)
	if err != nil {
		t.Fatalf("NewAccessPolicyID() error = %v", err)
	}

	if id1.Int64() != id2.Int64() {
		t.Errorf("同じ値で生成したAccessPolicyIDは等しいべき: id1 = %v, id2 = %v", id1.Int64(), id2.Int64())
	}

	if id1.Int64() == id3.Int64() {
		t.Errorf("異なる値で生成したAccessPolicyIDは異なるべき: id1 = %v, id3 = %v", id1.Int64(), id3.Int64())
	}
}
