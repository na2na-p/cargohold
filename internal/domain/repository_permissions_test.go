package domain_test

import (
	"testing"

	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewRepositoryPermissions(t *testing.T) {
	tests := []struct {
		name     string
		admin    bool
		push     bool
		pull     bool
		maintain bool
		triage   bool
	}{
		{
			name:     "正常系: 全てfalseで作成できる",
			admin:    false,
			push:     false,
			pull:     false,
			maintain: false,
			triage:   false,
		},
		{
			name:     "正常系: 全てtrueで作成できる",
			admin:    true,
			push:     true,
			pull:     true,
			maintain: true,
			triage:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.NewRepositoryPermissions(tt.admin, tt.push, tt.pull, tt.maintain, tt.triage)
			if got.CanUpload() != (tt.push || tt.admin || tt.maintain) {
				t.Errorf("CanUpload() = %v, want %v", got.CanUpload(), tt.push || tt.admin || tt.maintain)
			}
			if got.CanDownload() != (tt.pull || tt.push || tt.admin || tt.maintain || tt.triage) {
				t.Errorf("CanDownload() = %v, want %v", got.CanDownload(), tt.pull || tt.push || tt.admin || tt.maintain || tt.triage)
			}
		})
	}
}

func TestRepositoryPermissions_CanUpload(t *testing.T) {
	tests := []struct {
		name     string
		admin    bool
		push     bool
		pull     bool
		maintain bool
		triage   bool
		want     bool
	}{
		{
			name:     "正常系: push権限のみでアップロード可能",
			admin:    false,
			push:     true,
			pull:     false,
			maintain: false,
			triage:   false,
			want:     true,
		},
		{
			name:     "正常系: admin権限のみでアップロード可能",
			admin:    true,
			push:     false,
			pull:     false,
			maintain: false,
			triage:   false,
			want:     true,
		},
		{
			name:     "正常系: maintain権限のみでアップロード可能",
			admin:    false,
			push:     false,
			pull:     false,
			maintain: true,
			triage:   false,
			want:     true,
		},
		{
			name:     "正常系: pull権限のみではアップロード不可",
			admin:    false,
			push:     false,
			pull:     true,
			maintain: false,
			triage:   false,
			want:     false,
		},
		{
			name:     "正常系: triage権限のみではアップロード不可",
			admin:    false,
			push:     false,
			pull:     false,
			maintain: false,
			triage:   true,
			want:     false,
		},
		{
			name:     "正常系: 全て権限なしではアップロード不可",
			admin:    false,
			push:     false,
			pull:     false,
			maintain: false,
			triage:   false,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := domain.NewRepositoryPermissions(tt.admin, tt.push, tt.pull, tt.maintain, tt.triage)
			if got := p.CanUpload(); got != tt.want {
				t.Errorf("CanUpload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepositoryPermissions_CanDownload(t *testing.T) {
	tests := []struct {
		name     string
		admin    bool
		push     bool
		pull     bool
		maintain bool
		triage   bool
		want     bool
	}{
		{
			name:     "正常系: pull権限のみでダウンロード可能",
			admin:    false,
			push:     false,
			pull:     true,
			maintain: false,
			triage:   false,
			want:     true,
		},
		{
			name:     "正常系: push権限のみでダウンロード可能",
			admin:    false,
			push:     true,
			pull:     false,
			maintain: false,
			triage:   false,
			want:     true,
		},
		{
			name:     "正常系: admin権限のみでダウンロード可能",
			admin:    true,
			push:     false,
			pull:     false,
			maintain: false,
			triage:   false,
			want:     true,
		},
		{
			name:     "正常系: maintain権限のみでダウンロード可能",
			admin:    false,
			push:     false,
			pull:     false,
			maintain: true,
			triage:   false,
			want:     true,
		},
		{
			name:     "正常系: triage権限のみでダウンロード可能",
			admin:    false,
			push:     false,
			pull:     false,
			maintain: false,
			triage:   true,
			want:     true,
		},
		{
			name:     "正常系: 全て権限なしではダウンロード不可",
			admin:    false,
			push:     false,
			pull:     false,
			maintain: false,
			triage:   false,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := domain.NewRepositoryPermissions(tt.admin, tt.push, tt.pull, tt.maintain, tt.triage)
			if got := p.CanDownload(); got != tt.want {
				t.Errorf("CanDownload() = %v, want %v", got, tt.want)
			}
		})
	}
}
