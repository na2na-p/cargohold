package domain_test

import (
	"strings"
	"testing"

	"github.com/na2na-p/cargohold/internal/domain"
)

func TestNewCredentialCommand(t *testing.T) {
	tests := []struct {
		name      string
		shellType domain.ShellType
		host      string
		sessionID string
		wantParts []string
	}{
		{
			name:      "正常系: bashの場合はheredoc構文が使用される",
			shellType: domain.ShellTypeBash,
			host:      "example.com",
			sessionID: "session-123",
			wantParts: []string{
				"git credential approve <<EOF",
				"protocol=https",
				"host=example.com",
				"username=x-session",
				"password=session-123",
				"EOF",
			},
		},
		{
			name:      "正常系: zshの場合はheredoc構文が使用される",
			shellType: domain.ShellTypeZsh,
			host:      "example.com",
			sessionID: "session-123",
			wantParts: []string{
				"git credential approve <<EOF",
				"protocol=https",
				"host=example.com",
				"username=x-session",
				"password=session-123",
				"EOF",
			},
		},
		{
			name:      "正常系: powershellの場合はhere-string構文が使用される",
			shellType: domain.ShellTypePowerShell,
			host:      "example.com",
			sessionID: "session-123",
			wantParts: []string{
				`@"`,
				"protocol=https",
				"host=example.com",
				"username=x-session",
				"password=session-123",
				`"@ | git credential approve`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.NewCredentialCommand(tt.shellType, tt.host, tt.sessionID).String()
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("NewCredentialCommand().String() missing %q.\nGot: %s", part, got)
				}
			}
		})
	}
}
