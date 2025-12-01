# Contributing to Octobud

Thank you for your interest in contributing to Octobud! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
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

- **Bug fixes** — Fix issues in the codebase
- **Features** — Add new functionality
- **Documentation** — Improve or add documentation
- **Tests** — Improve test coverage
- **Bug reports** — Report issues you encounter
- **Feature requests** — Suggest new features

### Before You Start

1. Check if there's an existing issue for what you want to work on
2. For significant changes, open an issue first to discuss the approach
3. For bugs, check if it's already been reported

## Development Setup

### Prerequisites

- Go 1.24+
- Node.js 24+
- Docker and Docker Compose
- PostgreSQL (via Docker or local)

### Setup

```bash
# Clone the repository
git clone https://github.com/ajbeattie/octobud.git
cd octobud

# Run the setup script
make setup

# Copy and configure environment
cp .env.example .env
# Add your GitHub token to .env

# Start development servers
make backend-dev    # Terminal 1
make frontend-dev   # Terminal 2
make worker         # Terminal 3 (optional, for syncing)
```

### Database

```bash
# Start PostgreSQL
make db-start

# Run migrations
make db-migrate

# Connect to database
make psql
```

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

1. **What** — Brief description of the change
2. **Why** — Motivation for the change
3. **How** — High-level approach (if not obvious)
4. **Testing** — How you tested the changes
5. **Screenshots** — For UI changes

### Review Process

1. A maintainer will review your PR
2. Address any feedback
3. Once approved, a maintainer will merge

## Code Style

### Go (Backend)

- Follow standard Go conventions
- Run `gofmt` before committing
- Use meaningful variable and function names
- Add comments for exported functions
- Handle errors explicitly

```bash
# Format Go code
make format
```

### TypeScript/Svelte (Frontend)

- Follow the existing code style
- Use TypeScript for type safety
- Prefer composition over inheritance
- Keep components focused and small

```bash
# Lint frontend code
cd frontend && npm run lint
```

### General

- Keep functions small and focused
- Prefer clarity over cleverness
- Add comments for complex logic
- Remove commented-out code

## Testing

### Backend Tests

```bash
cd backend
go test ./...           # Run all tests
go test -v ./...        # Verbose output
go test -cover ./...    # With coverage
```

### Frontend Tests

```bash
cd frontend
npm test              # Run tests
npm run test:watch    # Watch mode
```

### Test Guidelines

- Write tests for new features
- Add regression tests for bug fixes
- Aim for meaningful coverage, not 100%
- Use table-driven tests in Go where appropriate

## Documentation

### When to Update Docs

- Adding new features
- Changing existing behavior
- Adding configuration options
- Changing API endpoints

### Documentation Structure

```
docs/
├── getting-started.md     # Setup and first steps
├── features/              # Feature documentation
│   ├── query-syntax.md
│   ├── keyboard-shortcuts.md
│   └── views-and-rules.md
└── architecture/          # Technical deep-dives
    ├── query_engine.md
    └── sync.md
```

## Questions?

- Open a [Discussion](https://github.com/ajbeattie/octobud/discussions) for questions
- Join our community chat (if available)
- Check existing issues and discussions

Thank you for contributing! 🎉

