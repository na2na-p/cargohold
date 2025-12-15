#!/bin/sh

set -e

# Environment variables with defaults
POSTGRES_HOST="${DATABASE_HOST:-localhost}"
POSTGRES_PORT="${DATABASE_PORT:-5432}"
POSTGRES_USER="${DATABASE_USER:-cargohold}"
POSTGRES_PASSWORD="${DATABASE_PASSWORD:-}"
POSTGRES_DB="${DATABASE_DBNAME:-cargohold}"
SSL_MODE="${DATABASE_SSLMODE:-disable}"

MIGRATIONS_DIR="${MIGRATIONS_DIR:-/app/migrations}"

# Build database connection string
# Format: postgres://user:password@host:port/dbname?sslmode=disable
if [ -n "$POSTGRES_PASSWORD" ]; then
	DBSTRING="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=${SSL_MODE}"
else
	DBSTRING="postgres://${POSTGRES_USER}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=${SSL_MODE}"
fi

# Check if goose is available
if ! command -v goose >/dev/null 2>&1; then
	echo "Error: goose command not found"
	exit 1
fi

# Check if migrations directory exists
if [ ! -d "$MIGRATIONS_DIR" ]; then
	echo "Error: migrations directory not found: $MIGRATIONS_DIR"
	exit 1
fi

# Execute goose command
# Usage: migrate-goose.sh [up|down|status|version]
COMMAND="${1:-up}"

echo "Running goose $COMMAND..."
echo "Migrations directory: $MIGRATIONS_DIR"
echo "Database: ${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}"

cd "$MIGRATIONS_DIR"

case "$COMMAND" in
	up)
		goose postgres "$DBSTRING" up
		echo "Migrations applied successfully"
		;;
	down)
		goose postgres "$DBSTRING" down
		echo "Migration rolled back successfully"
		;;
	status)
		goose postgres "$DBSTRING" status
		;;
	version)
		goose postgres "$DBSTRING" version
		;;
	*)
		echo "Usage: $0 {up|down|status|version}"
		echo ""
		echo "Commands:"
		echo "  up      Apply all pending migrations"
		echo "  down    Roll back the last migration"
		echo "  status  Show migration status"
		echo "  version Display current migration version"
		exit 1
		;;
esac
