# Contributing to Octobud

Thank you for your interest in contributing to Octobud! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Submitting a Pull Request](#submitting-a-pull-request)
- [Code Style](#code-style)
- [Testing](#testing)
- [Documentation](#documentation)

## Code of Conduct

This project follows our [Code of Conduct](../CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Types of Contributions

We welcome many types of contributions:

- **Bug fixes** - Fix issues in the codebase
- **Features** - Add new functionality
- **Documentation** - Improve or add documentation
- **Tests** - Improve test coverage
- **Bug reports** - Report issues you encounter
- **Feature requests** - Suggest new features

### Before You Start

1. Check if there's an existing issue for what you want to work on
2. For significant changes, open an issue first to discuss the approach
3. For bugs, check if it's already been reported

## Project Structure

```
octobud/
├── backend/           # Go backend service
│   ├── cmd/          # Entry points (server, worker, migrations)
│   ├── internal/     # Internal packages
│   │   ├── api/      # HTTP handlers
│   │   ├── core/     # Business logic services
│   │   ├── db/       # Database queries (sqlc generated)
│   │   ├── github/   # GitHub API client
│   │   ├── jobs/     # Background jobs (River)
│   │   ├── sync/     # Sync service
│   │   └── query/    # Query language parser
│   └── migrations/   # Database migrations (goose)
├── frontend/         # Svelte 5 + SvelteKit frontend
│   └── src/
│       ├── lib/      # Components and utilities
│       └── routes/   # SvelteKit routes
└── docs/             # Documentation
```

## Development Setup

### Prerequisites

- **Go** 1.24+
- **Node.js** 24+ and npm
- **Docker** and Docker Compose (for PostgreSQL)
- **GitHub PAT** with `notifications` and `repo` scopes ([create one here](https://github.com/settings/tokens))

### Setup

```bash
# Clone and setup
git clone https://github.com/ajbeattie/octobud.git
cd octobud
make setup
```

This installs Go tools, npm dependencies, starts PostgreSQL, and runs migrations.

### Configure Environment

```bash
cp .env.example .env
```

Edit `.env` and set:
- `GH_TOKEN` - Your GitHub Personal Access Token
- `JWT_SECRET` - Generate with `openssl rand -hex 32` (or run `make ensure-jwt-secret`)

### Start Services

Run in separate terminals:

```bash
make backend-dev    # Terminal 1 - Backend API
make frontend-dev   # Terminal 2 - Frontend with hot reload
make worker         # Terminal 3 - Notification sync
```

Open http://localhost:5173 and login with `octobud` / `octobud`.

### Database Commands

```bash
make db-start    # Start PostgreSQL
make db-stop     # Stop PostgreSQL
make db-migrate  # Run migrations
make psql        # Connect to database
```

### All Available Commands

| Command | Description |
|---------|-------------|
| `make setup` | First-time setup (install tools, deps, DB, migrations) |
| `make install-tools` | Install Go tools (goose, sqlc, mockgen) |
| `make frontend-install` | Install frontend npm dependencies |
| `make generate` | Generate code (sqlc queries, mocks) |
| `make docker-up` | Run production stack (port 3000) |
| `make docker-up-dev` | Run dev stack with hot reload (port 5173) |
| `make docker-up-1password` | Run production stack with 1Password token |
| `make docker-up-dev-1password` | Run dev stack with 1Password token |
| `make docker-down` | Stop all Docker containers |
| `make backend-dev` | Run backend server locally |
| `make frontend-dev` | Run frontend dev server locally |
| `make worker` | Run notification sync worker |
| `make dev` | Run backend server (alias for backend-dev) |
| `make dev-prompt` | Run backend server with interactive token prompt |
| `make worker-prompt` | Run worker with interactive token prompt |
| `make format` | Format all code (backend and frontend) |
| `make lint` | Lint all code (backend and frontend) |
| `make test` | Run all tests (backend and frontend) |
| `make db-start` | Start PostgreSQL in Docker |
| `make db-stop` | Stop PostgreSQL |
| `make db-migrate` | Run database migrations |
| `make psql` | Open PostgreSQL shell |
| `make format-backend` | Format Go code only |
| `make lint-backend` | Lint Go code only |
| `make test-backend` | Run backend tests only |
| `make test-frontend` | Run frontend tests only |
| `make clean` | Remove built binaries |
| `make build-frontend` | Build frontend server with embedded assets |

## Making Changes

### Branch Naming

Use descriptive branch names:

- `feature/add-snooze-reminder`
- `fix/notification-sync-error`
- `docs/update-readme`
- `refactor/query-parser`

### Commit Messages

Write clear, concise commit messages:

```
Short summary (50 chars or less)

More detailed explanation if needed. Wrap at 72 characters.
Explain what and why, not how.

Fixes #123
```

### Code Changes

1. Create a branch from `main`
2. Make your changes
3. Add or update tests as needed
4. Ensure all tests pass
5. Update documentation if needed

## Submitting a Pull Request

### Before Submitting

- [ ] Code follows the style guidelines
- [ ] Tests pass locally (`make test`)
- [ ] New code has appropriate test coverage
- [ ] Documentation is updated if needed
- [ ] Commit messages are clear and descriptive

### PR Description

Include in your PR description:

1. **What** - Brief description of the change
2. **Why** - Motivation for the change
3. **How** - High-level approach (if not obvious)
4. **Testing** - How you tested the changes
5. **Screenshots** - For UI changes

### Review Process

1. A maintainer will review your PR
2. Address any feedback
3. Once approved, a maintainer will merge

## Code Style

Before committing, run from the project root:

```bash
make format    # Format all code
make lint      # Check for issues
```

### Go (Backend)

- Follow standard Go conventions
- Use meaningful variable and function names
- Add comments for exported functions
- Handle errors explicitly

### TypeScript/Svelte (Frontend)

- Follow the existing code style
- Use TypeScript for type safety
- Prefer composition over inheritance
- Keep components focused and small

### General

- Keep functions small and focused
- Prefer clarity over cleverness
- Add comments for complex logic
- Remove commented-out code

## Testing

Run all tests from the project root:

```bash
make test             # Run all tests (backend + frontend)
make test-backend     # Backend only
make test-frontend    # Frontend only
```

### Test Guidelines

- Write tests for new features
- Add regression tests for bug fixes
- Aim for meaningful coverage, not 100%
- Use table-driven tests in Go where appropriate
- Please do manual testing of new features and check for regressions if touching an existing feature ;)

## Documentation

### When to Update Docs

- Adding new features
- Changing existing behavior
- Adding configuration options
- Changing API endpoints

### Documentation Structure

```
docs/
├── README.md              # Documentation index
├── start-here.md          # Initial setup and core workflows
├── deployment.md          # Production deployment
├── guides/                # User guides and reference
│   ├── query-syntax.md
│   ├── keyboard-shortcuts.md
│   └── views-and-rules.md
├── concepts/              # How things work under the hood
│   ├── action-hints.md
│   ├── query-engine.md
│   └── sync.md
└── security/              # Security documentation
    └── ...
```

## Questions?

- Open a [Discussion](https://github.com/ajbeattie/octobud/discussions) for questions
- Join our community chat (if available)
- Check existing issues and discussions

Thank you for contributing! 🎉

