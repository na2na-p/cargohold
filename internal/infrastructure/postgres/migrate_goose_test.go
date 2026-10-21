package postgres_test

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgconn"
)

type parsedConnection struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
	Database string `json:"database"`
}

func TestMigrateGoose_UsesEnvironmentConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		user     string
		password string
		exitCode int
	}{
		{name: "IAMユーザー名と空パスワードを解釈できる", user: "hades-cargohold@na2na-platform.iam"},
		{name: "予約文字を含むパスワードを解釈できる", user: "cargohold", password: "p@ss:word?&"},
		{name: "gooseの終了コードを伝播する", user: "hades-cargohold@na2na-platform.iam", exitCode: 23},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			_, sourceFile, _, ok := runtime.Caller(0)
			if !ok {
				t.Fatal("runtime.Caller failed")
			}
			migrationsDir := filepath.Join(tmpDir, "migrations")
			if err := os.Mkdir(migrationsDir, 0o755); err != nil {
				t.Fatal(err)
			}
			testBinary := os.Args[0]
			goosePath := filepath.Join(tmpDir, "goose")
			gooseScript := "#!/bin/sh\nexec " + strconv.Quote(testBinary) + " -test.run=TestGooseHelper -- \"$@\"\n"
			if err := os.WriteFile(goosePath, []byte(gooseScript), 0o755); err != nil {
				t.Fatal(err)
			}

			outputPath := filepath.Join(tmpDir, "parsed.json")
			env := append(os.Environ(),
				"PATH="+tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"MIGRATIONS_DIR="+migrationsDir,
				"DATABASE_HOST=db.example.test",
				"DATABASE_PORT=15432",
				"DATABASE_USER="+tt.user,
				"DATABASE_PASSWORD="+tt.password,
				"DATABASE_DBNAME=cargohold",
				"DATABASE_SSLMODE=disable",
				"GOOSE_HELPER_OUTPUT="+outputPath,
				"GOOSE_HELPER_EXIT="+strconv.Itoa(tt.exitCode),
			)
			cmd := exec.Command("sh", "scripts/migrate-goose.sh", "up")
			cmd.Dir = filepath.Join(filepath.Dir(sourceFile), "..", "..", "..")
			cmd.Env = env
			err := cmd.Run()
			if tt.exitCode == 0 {
				if err != nil {
					t.Fatalf("migrate-goose.sh: %v", err)
				}
				contents, readErr := os.ReadFile(outputPath)
				if readErr != nil {
					t.Fatal(readErr)
				}
				var got parsedConnection
				if unmarshalErr := json.Unmarshal(contents, &got); unmarshalErr != nil {
					t.Fatal(unmarshalErr)
				}
				want := parsedConnection{
					User: tt.user, Password: tt.password,
					Host: "db.example.test", Port: 15432, Database: "cargohold",
				}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Fatalf("parsed connection mismatch (-want +got):\n%s", diff)
				}
				return
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != tt.exitCode {
				t.Fatalf("exit error = %v, want exit code %d", err, tt.exitCode)
			}
		})
	}
}

func TestGooseHelper(t *testing.T) {
	if os.Getenv("GOOSE_HELPER_OUTPUT") == "" {
		return
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) < 4 {
		t.Fatalf("unexpected goose arguments: %v", os.Args)
	}
	if args[2] != "" {
		t.Fatalf("goose DSN = %q, want empty DSN", args[2])
	}
	config, err := pgconn.ParseConfig(args[2])
	if err != nil {
		t.Fatalf("pgconn.ParseConfig: %v", err)
	}
	parsed, err := json.Marshal(parsedConnection{
		User: config.User, Password: config.Password, Host: config.Host,
		Port: config.Port, Database: config.Database,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("GOOSE_HELPER_OUTPUT"), parsed, 0o600); err != nil {
		t.Fatal(err)
	}
	if exitCode, err := strconv.Atoi(os.Getenv("GOOSE_HELPER_EXIT")); err == nil && exitCode != 0 {
		os.Exit(exitCode)
	}
}
