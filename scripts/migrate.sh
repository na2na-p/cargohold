#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/.env"

if [ -f "${ENV_FILE}" ]; then
	set -a
	source "${ENV_FILE}"
	set +a
fi

POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-cargohold}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-cargohold_dev_password}"
POSTGRES_DB="${POSTGRES_DB:-cargohold}"

MIGRATIONS_PATH="${PROJECT_ROOT}/migrations"

DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"

if ! command -v migrate &> /dev/null; then
	echo "Error: migrate command not found"
	echo "Please install golang-migrate CLI tool"
	echo "See: https://github.com/golang-migrate/migrate"
	exit 1
fi

if [ ! -d "${MIGRATIONS_PATH}" ]; then
	echo "Error: migrations directory not found: ${MIGRATIONS_PATH}"
	exit 1
fi

case "$1" in
	up)
		echo "Applying migrations..."
		migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" up
		echo "Migrations applied successfully"
		;;
	down)
		echo "Rolling back migration (1 step)..."
		migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" down 1
		echo "Migration rolled back successfully"
		;;
	version)
		echo "Current migration version:"
		migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" version
		;;
	force)
		if [ -z "$2" ]; then
			echo "Error: version number required"
			echo "Usage: $0 force <version>"
			exit 1
		fi
		echo "Forcing migration version to $2..."
		migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" force "$2"
		echo "Migration version forced successfully"
		;;
	*)
		echo "Usage: $0 {up|down|version|force <version>}"
		echo ""
		echo "Commands:"
		echo "  up              Apply all pending migrations"
		echo "  down            Roll back the last migration"
		echo "  version         Display current migration version"
		echo "  force <version> Force set the migration version"
		exit 1
		;;
esac
