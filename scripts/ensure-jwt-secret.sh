#!/bin/bash
# Ensure JWT_SECRET exists in .env (for both local dev and docker)
# This script checks for JWT_SECRET in the root .env file, generating
# a secure random secret if missing.
#
# This script should be run from the project root directory.

set -e

# Ensure we're in the project root (check for backend/ directory)
if [ ! -d "backend" ]; then
	echo "Error: This script must be run from the project root directory" >&2
	exit 1
fi

echo "Ensuring JWT_SECRET is configured..."

# Check and update root .env
if [ ! -f .env ]; then
	touch .env
fi

SECRET_LINE=$(grep "^JWT_SECRET=" .env 2>/dev/null || echo "")
SECRET_VALUE=$(echo "$SECRET_LINE" | cut -d'=' -f2- | tr -d ' ')

if [ -z "$SECRET_VALUE" ]; then
	echo "Generating JWT_SECRET for .env..."
	NEW_SECRET=$(openssl rand -hex 32)
	if [ -n "$SECRET_LINE" ]; then
		grep -v "^JWT_SECRET=" .env > .env.tmp 2>/dev/null || true
		mv .env.tmp .env
	fi
	echo "JWT_SECRET=$NEW_SECRET" >> .env
	echo "✓ JWT_SECRET added to .env"
else
	echo "✓ JWT_SECRET already configured in .env"
fi

echo "✓ JWT_SECRET configuration complete"

