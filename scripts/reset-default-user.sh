#!/bin/bash
# Reset the default user credentials by deleting the user from the database.
# The server will automatically recreate the default octobud:octobud user on next startup.
#
# This script should be run from the project root directory.

set -e

# Ensure we're in the project root (check for backend/ directory)
if [ ! -d "backend" ]; then
	echo "Error: This script must be run from the project root directory" >&2
	exit 1
fi

# Load DATABASE_URL from .env if it exists (check root .env first, then backend/.env for backward compatibility)
if [ -f .env ]; then
	export $(grep -v '^#' .env | grep DATABASE_URL | xargs)
elif [ -f backend/.env ]; then
	export $(grep -v '^#' backend/.env | grep DATABASE_URL | xargs)
fi

# Default DATABASE_URL if not set
DATABASE_URL=${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/octobud?sslmode=disable}

echo "Resetting default user credentials..."
echo "This will delete the current user. The server will recreate octobud:octobud on next startup."
echo ""

# Ask for confirmation
read -p "Are you sure you want to delete the current user? [y/N] " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
	echo "Cancelled. No changes made."
	exit 0
fi
echo ""

# Parse DATABASE_URL to extract connection details
# Format: postgres://user:password@host:port/database?sslmode=mode
if [[ ! "$DATABASE_URL" =~ postgres://([^:]+):([^@]+)@([^/]+)/([^?]+) ]]; then
	echo "Error: Invalid DATABASE_URL format" >&2
	echo "Expected format: postgres://user:password@host:port/database?sslmode=mode" >&2
	exit 1
fi

DB_USER="${BASH_REMATCH[1]}"
DB_PASS="${BASH_REMATCH[2]}"
DB_HOST_PORT="${BASH_REMATCH[3]}"
DB_NAME="${BASH_REMATCH[4]}"

# Split host:port
if [[ "$DB_HOST_PORT" =~ ^([^:]+):([0-9]+)$ ]]; then
	DB_HOST="${BASH_REMATCH[1]}"
	DB_PORT="${BASH_REMATCH[2]}"
else
	DB_HOST="$DB_HOST_PORT"
	DB_PORT="5432"
fi

# Check if we're using Docker (host is "postgres" which is the Docker service name)
if [ "$DB_HOST" = "postgres" ]; then
	# Try to use docker compose if available
	if command -v docker >/dev/null 2>&1; then
		# Try dev compose first
		if docker compose -f docker-compose.dev.yaml ps postgres 2>/dev/null | grep -q "running"; then
			echo "Using Docker Compose (dev) to connect to database..."
			echo "Deleting user from database..."
			docker compose -f docker-compose.dev.yaml exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" <<EOF
DELETE FROM users;
SELECT 'Default user deleted. Restart the server to recreate octobud:octobud user.' as message;
EOF
			exit 0
		# Try production compose
		elif docker compose ps postgres 2>/dev/null | grep -q "running"; then
			echo "Using Docker Compose (production) to connect to database..."
			echo "Deleting user from database..."
			docker compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" <<EOF
DELETE FROM users;
SELECT 'Default user deleted. Restart the server to recreate octobud:octobud user.' as message;
EOF
			exit 0
		else
			echo "Warning: DATABASE_URL points to 'postgres' but no running Docker container found." >&2
			echo "Falling back to direct psql connection..." >&2
		fi
	fi
fi

# Fall back to direct psql connection
if ! command -v psql >/dev/null 2>&1; then
	echo "Error: psql command not found. Please install PostgreSQL client tools." >&2
	exit 1
fi

echo "Connecting to database at $DB_HOST:$DB_PORT..."
echo "Deleting user from database..."
PGPASSWORD="$DB_PASS" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
DELETE FROM users;
SELECT 'Default user deleted. Restart the server to recreate octobud:octobud user.' as message;
EOF

echo ""
echo "✓ Default user has been deleted."
echo "  Restart the server to automatically recreate the octobud:octobud user."

