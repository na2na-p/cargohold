package domain

import "fmt"

type CredentialCommand struct {
	value string
}

func NewCredentialCommand(shellType ShellType, host, sessionID string) CredentialCommand {
	switch shellType {
	case ShellTypePowerShell:
		return CredentialCommand{
			value: fmt.Sprintf(`@"
protocol=https
host=%s
username=x-session
password=%s
"@ | git credential approve`, host, sessionID),
		}
	default:
		return CredentialCommand{
			value: fmt.Sprintf(`git credential approve <<EOF
protocol=https
host=%s
username=x-session
password=%s
EOF`, host, sessionID),
		}
	}
}

func (c CredentialCommand) String() string {
	return c.value
}
