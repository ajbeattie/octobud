# Load DATABASE_URL from .env for migration commands
# Try root .env first (../.env), then backend/.env for backward compatibility
# Go commands use godotenv and load .env automatically
ifneq (,$(wildcard ../.env))
	include ../.env
	export
else ifneq (,$(wildcard ./.env))
	include .env
	export
endif

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/octobud?sslmode=disable

# Get Go bin directory for tools installed via go install
# GOBIN takes precedence over GOPATH/bin if set
GOBIN := $(shell go env GOBIN)
GOPATH_BIN := $(shell go env GOPATH)/bin
GO_TOOL_BIN := $(if $(GOBIN),$(GOBIN),$(GOPATH_BIN))
SQLC := $(GO_TOOL_BIN)/sqlc
MOCKGEN := $(GO_TOOL_BIN)/mockgen
GOOSE := $(GO_TOOL_BIN)/goose
GOLANGCI_LINT := $(GO_TOOL_BIN)/golangci-lint

.PHONY: install-tools dev dev-prompt worker worker-prompt migrate-up migrate-down psql test lint format generate-sqlc build-frontend

# Pinned tool versions - update these when upgrading tools
GOOSE_VERSION := v3.26.0
SQLC_VERSION := v1.30.0
MOCKGEN_VERSION := v0.6.0
GOLANGCI_LINT_VERSION := v2.6.2

install-tools:
	@echo "Installing Go tooling..."
	go install github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION)
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

dev:
	go run ./cmd/server

dev-prompt:
	go run ./cmd/server --prompt-token

worker:
	go run ./cmd/worker

worker-prompt:
	go run ./cmd/worker --prompt-token

migrate-up:
	$(GOOSE) -dir ./migrations postgres "$(DATABASE_URL)" up

migrate-down:
	$(GOOSE) -dir ./migrations postgres "$(DATABASE_URL)" down

psql:
	docker compose -f ../docker-compose.dev.yaml exec postgres psql -U postgres -d octobud

test:
	@mkdir -p web/dist
	@touch web/dist/.gitkeep
	go test ./...

lint:
	@echo "Running golangci-lint..."
	@if ! command -v $(GOLANGCI_LINT) > /dev/null; then \
		echo "Error: golangci-lint is not installed. Run 'make install-tools' first."; \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run
	@echo "✓ Linting complete"

format:
	@echo "Formatting with golangci-lint --fix..."
	@if ! command -v $(GOLANGCI_LINT) > /dev/null; then \
		echo "Error: golangci-lint is not installed. Run 'make install-tools' first."; \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run --fix
	@echo "✓ Formatting complete"

generate-sqlc:
	$(SQLC) generate

generate-mocks:
	@echo "Generating mocks..."
	PATH="$(GO_TOOL_BIN):$$PATH" go generate

generate: generate-sqlc generate-mocks
	@echo "Code generation complete"

# Build frontend server binary with embedded assets
build-frontend:
	@echo "Building frontend assets..."
	cd ../frontend && npm run build
	@echo "Copying frontend build to web/dist..."
	rm -rf web/dist
	mkdir -p web/dist
	cp -r ../frontend/build/* web/dist/
	@echo "Building frontend server binary..."
	mkdir -p bin
	CGO_ENABLED=0 go build -o bin/octobud-web ./cmd/frontend
	@echo "✓ Built: bin/octobud-web"
	@echo ""
	@echo "Run with: ./bin/octobud-web"
	@echo "The backend API server must be running (default: localhost:8080)"

