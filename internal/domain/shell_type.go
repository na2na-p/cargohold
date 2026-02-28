package domain

import (
	"errors"
	"fmt"
)

var ErrInvalidShellType = errors.New("invalid shell type")

type ShellType struct {
	value string
}

var (
	ShellTypeBash       = ShellType{value: "bash"}
	ShellTypeZsh        = ShellType{value: "zsh"}
	ShellTypePowerShell = ShellType{value: "powershell"}
)

func ParseShellType(s string) (ShellType, error) {
	switch s {
	case "", ShellTypeBash.value:
		return ShellTypeBash, nil
	case ShellTypeZsh.value:
		return ShellTypeZsh, nil
	case ShellTypePowerShell.value:
		return ShellTypePowerShell, nil
	default:
		return ShellType{}, fmt.Errorf("%w: %q", ErrInvalidShellType, s)
	}
}

func (s ShellType) String() string {
	return s.value
}

func (s ShellType) IsZero() bool {
	return s.value == ""
}

func (s ShellType) CredentialCommand(host, sessionID string) string {
	switch s {
	case ShellTypePowerShell:
		return fmt.Sprintf(`@"
protocol=https
host=%s
username=x-session
password=%s
"@ | git credential approve`, host, sessionID)
	default:
		return fmt.Sprintf(`git credential approve <<EOF
protocol=https
host=%s
username=x-session
password=%s
EOF`, host, sessionID)
	}
}
