ROOT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

.PHONY: help setup install-tools backend-install-tools frontend-install backend-dev frontend-dev dev dev-prompt worker worker-prompt ensure-jwt-secret reset-default-user docker-up docker-up-dev docker-up-1password docker-up-dev-1password docker-down db-start db-stop db-migrate psql fmt lint lint-backend lint-frontend test test-backend test-frontend clean generate build-frontend

.DEFAULT_GOAL := help

# Default target - show help
help:
	@echo "Octobud - Available Make Targets"
	@echo ""
	@echo "Setup & Installation:"
	@echo "  make setup              - Complete first-time setup (install, start DB, migrate)"
	@echo "  make install-tools      - Install Go tools (goose, sqlc, mockgen)"
	@echo "  make generate           - Generate code (sqlc, mocks)"
	@echo "  make frontend-install   - Install frontend npm dependencies"
	@echo "  make ensure-jwt-secret  - Generate JWT_SECRET if missing (auto-run with docker targets)"
	@echo "  make reset-default-user - Reset default user to octobud:octobud (deletes current user)"
	@echo ""
	@echo "Development (local):"
	@echo "  make backend-dev        - Run backend server locally"
	@echo "  make frontend-dev       - Run frontend dev server locally"
	@echo "  make worker             - Run GitHub sync worker locally"
	@echo "  make dev                - Run backend server (alias)"
	@echo "  make dev-prompt         - Run backend server with token prompt"
	@echo "  make worker-prompt      - Run GitHub sync worker with token prompt"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up              - Run production stack (port 3000)"
	@echo "  make docker-up-dev          - Run dev stack with hot reload (port 5173)"
	@echo "  make docker-up-1password    - Run production stack with 1Password token"
	@echo "  make docker-up-dev-1password - Run dev stack with 1Password token"
	@echo "  make docker-down            - Stop all containers"
	@echo ""
	@echo "Build:"
	@echo "  make build-frontend     - Build frontend server with embedded assets"
	@echo ""
	@echo "Database:"
	@echo "  make db-start           - Start PostgreSQL in Docker"
	@echo "  make db-stop            - Stop PostgreSQL"
	@echo "  make db-migrate         - Run database migrations"
	@echo "  make psql               - Open PostgreSQL shell"
	@echo ""
	@echo "Utilities:"
	@echo "  make format                - Format Go code"
	@echo "  make lint               - Lint backend and frontend"
	@echo "  make lint-backend       - Lint backend only"
	@echo "  make lint-frontend      - Lint frontend only"
	@echo "  make test               - Run backend and frontend tests"
	@echo "  make test-backend       - Run backend tests only"
	@echo "  make test-frontend      - Run frontend tests only"
	@echo "  make clean              - Remove built binaries"

# Complete first-time setup
setup: install-tools frontend-install ensure-jwt-secret db-start db-migrate
	@mkdir -p backend/web/dist
	@touch backend/web/dist/.gitkeep
	@echo ""
	@echo "✓ Setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. (Optional) Add your GitHub token to .env"
	@echo "     cp .env.example .env"
	@echo "     # Edit .env and add your GH_TOKEN"
	@echo "     # Note: JWT_SECRET has been automatically generated"
	@echo ""
	@echo "  2. Start development servers:"
	@echo "     Terminal 1: make backend-dev"
	@echo "     Terminal 2: make frontend-dev"
	@echo ""
	@echo "  3. Open http://localhost:5173"
	@echo ""

install-tools: backend-install-tools

backend-install-tools:
	@echo "Installing Go tools..."
	$(MAKE) -C backend install-tools
	@echo "✓ Go tools installed"

frontend-install:
	@echo "Installing frontend dependencies..."
	@cd frontend && npm install
	@echo "✓ Frontend dependencies installed"

# Start PostgreSQL database (for local development)
db-start:
	@echo "Starting PostgreSQL..."
	@docker compose -f docker-compose.dev.yaml up -d postgres
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 3
	@docker compose -f docker-compose.dev.yaml exec postgres pg_isready -U postgres || (echo "Waiting for database..." && sleep 5)
	@echo "✓ PostgreSQL is running on localhost:5432"

# Stop PostgreSQL database
db-stop:
	@echo "Stopping PostgreSQL..."
	@docker compose -f docker-compose.dev.yaml stop postgres
	@echo "✓ PostgreSQL stopped"

# Run database migrations
db-migrate:
	@echo "Running database migrations..."
	@cd backend && $(MAKE) migrate-up
	@echo "✓ Migrations complete"

backend-dev:
	cd backend && go run ./cmd/server

frontend-dev:
	cd frontend && npm run dev

dev: backend-dev

dev-prompt:
	$(MAKE) -C backend dev-prompt

worker:
	$(MAKE) -C backend worker

worker-prompt:
	$(MAKE) -C backend worker-prompt

# Ensure JWT_SECRET exists in .env (for both local dev and docker)
ensure-jwt-secret:
	@./scripts/ensure-jwt-secret.sh

# Reset default user credentials (deletes user, server will recreate octobud:octobud on restart)
reset-default-user:
	@./scripts/reset-default-user.sh

# Run production stack with Docker
docker-up: ensure-jwt-secret
	@if [ -z "$$GH_TOKEN" ]; then \
		echo "Warning: GH_TOKEN not set. Sync will not work."; \
		echo "Set GH_TOKEN in .env file (cp .env.example .env)"; \
	fi
	docker compose up --build -d
	@echo ""
	@echo "✓ Services started"
	@echo "  Open http://localhost:3000"

# Run development stack with hot reload
docker-up-dev: ensure-jwt-secret
	@if [ -z "$$GH_TOKEN" ]; then \
		echo "Warning: GH_TOKEN not set. Sync will not work."; \
		echo "Set GH_TOKEN in .env file (cp .env.example .env)"; \
	fi
	docker compose -f docker-compose.dev.yaml up --build -d
	@echo ""
	@echo "✓ Dev services started"
	@echo "  Open http://localhost:5173"

# Stop all Docker containers
docker-down:
	docker compose down
	docker compose -f docker-compose.dev.yaml down 2>/dev/null || true
	@echo "✓ All containers stopped"

# Run production stack with 1Password token injection
docker-up-1password: ensure-jwt-secret
	@if ! command -v op &> /dev/null; then \
		echo "Error: 1Password CLI (op) is not installed."; \
		echo "Install it from: https://1password.com/downloads/command-line/"; \
		exit 1; \
	fi
	@if [ -z "$$OP_GH_TOKEN" ]; then \
		echo "Error: OP_GH_TOKEN environment variable is not set."; \
		echo ""; \
		echo "Usage:"; \
		echo "  OP_GH_TOKEN='op://Vault/Item/field' make docker-up-1password"; \
		echo ""; \
		echo "Example:"; \
		echo "  OP_GH_TOKEN='op://Private/GitHub PAT/token' make docker-up-1password"; \
		exit 1; \
	fi
	@echo "Starting production stack with 1Password token..."
	@bash -c 'op run --env-file=<(echo "GH_TOKEN=$$OP_GH_TOKEN") -- docker compose up --build -d'
	@echo ""
	@echo "✓ Services started"
	@echo "  Open http://localhost:3000"

# Run dev stack with 1Password token injection
docker-up-dev-1password: ensure-jwt-secret
	@if ! command -v op &> /dev/null; then \
		echo "Error: 1Password CLI (op) is not installed."; \
		echo "Install it from: https://1password.com/downloads/command-line/"; \
		exit 1; \
	fi
	@if [ -z "$$OP_GH_TOKEN" ]; then \
		echo "Error: OP_GH_TOKEN environment variable is not set."; \
		echo ""; \
		echo "Usage:"; \
		echo "  OP_GH_TOKEN='op://Vault/Item/field' make docker-up-dev-1password"; \
		echo ""; \
		echo "Example:"; \
		echo "  OP_GH_TOKEN='op://Private/GitHub PAT/token' make docker-up-dev-1password"; \
		exit 1; \
	fi
	@echo "Starting dev stack with 1Password token..."
	@bash -c 'op run --env-file=<(echo "GH_TOKEN=$$OP_GH_TOKEN") -- docker compose -f docker-compose.dev.yaml up --build -d'
	@echo ""
	@echo "✓ Dev services started"
	@echo "  Open http://localhost:5173"

psql:
	docker compose -f docker-compose.dev.yaml exec postgres psql -U postgres -d octobud

format: format-backend format-frontend
	@echo ""
	@echo "✓ All formatting complete!"

format-backend:
	@echo "Formatting backend..."
	@cd backend && $(MAKE) format

format-frontend:
	@echo "Formatting frontend..."
	@cd frontend && npm run format
	@echo "✓ Frontend formatting complete"

lint: lint-backend lint-frontend
	@echo ""
	@echo "✓ All linting complete!"

lint-backend:
	@echo "Linting backend..."
	cd backend && $(MAKE) lint

lint-frontend:
	@echo "Linting frontend..."
	@cd frontend && npm run check
	@cd frontend && npm run lint || (echo "⚠ ESLint found issues" && exit 1)
	@echo "✓ Frontend linting complete"

test: test-backend test-frontend
	@echo ""
	@echo "✓ All tests passed!"

test-backend:
	@echo "Running backend tests..."
	cd backend && $(MAKE) test

test-frontend:
	@echo "Running frontend tests..."
	cd frontend && npm test

clean:
	rm -f backend/server backend/worker backend/goose-migrations
	rm -rf backend/bin
	find backend/web/dist -mindepth 1 ! -name '.gitkeep' -delete 2>/dev/null || true
	@echo "✓ Cleaned build artifacts"

# Build frontend server with embedded assets
build-frontend:
	@echo "Building frontend server with embedded assets..."
	$(MAKE) -C backend build-frontend
	@echo ""
	@echo "Binary location: backend/bin/octobud-web"

generate:
	@echo "Generating code..."
	$(MAKE) -C backend generate
	@echo "✓ Code generation complete"

