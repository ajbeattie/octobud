# Getting Started with Octobud

This guide will help you get Octobud up and running on your machine.

## Prerequisites

Before you begin, ensure you have the following installed:

**For Docker (recommended - easiest way to run Octobud):**
- **Docker** and **Docker Compose**
- A **GitHub Personal Access Token** (PAT)

**For local development (without Docker):**
- **Go** 1.24 or later
- **Node.js** 24 or later (with npm)
- **Docker** and **Docker Compose** (for PostgreSQL only)
- A **GitHub Personal Access Token** (PAT)

### Creating a GitHub PAT

1. Go to [GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)](https://github.com/settings/tokens)
2. Click **Generate new token (classic)**
3. Give it a descriptive name like "Octobud"
4. Select the following scopes:
   - `notifications` — Required for reading notifications
   - `repo` — Required for accessing repository details
5. Click **Generate token**
6. Copy the token immediately (you won't see it again!)

---

## Option A: Docker Setup (Recommended)

**Best for:** Quick start, production deployments, minimal setup

Docker handles all dependencies and configuration automatically. No Go or Node.js installation needed.

### 1. Clone the Repository

```bash
git clone https://github.com/ajbeattie/octobud.git
cd octobud
```

### 2. Configure Environment Variables (Optional)

Copy the example environment file and add your GitHub token:

```bash
cp .env.example .env
```

Edit `.env` and set your token:

```env
GH_TOKEN=ghp_your_token_here
# Optional: JWT_SECRET=your_secure_random_secret_key_here  # Auto-generated if not set
# Optional: JWT_EXPIRY=168h  # Default is 7 days (168h). Examples: 720h (30 days), 2160h (90 days)
```

**Note:** You can also pass `GH_TOKEN` directly to the make commands without creating a `.env` file. JWT_SECRET is automatically generated if not set.

### 3. Start Services

**Development with hot reload (port 5173):**
```bash
# JWT_SECRET is automatically generated if not set
GH_TOKEN=your_token make docker-up-dev
```

Open http://localhost:5173 in your browser.

**Production build (port 3000):**
```bash
# JWT_SECRET is automatically generated if not set
GH_TOKEN=your_token make docker-up
```

Open http://localhost:3000 in your browser.

**What Docker starts:**
- PostgreSQL database
- Backend API server
- Frontend server
- Background sync worker
- All services configured and connected automatically

**Stop all services:**
```bash
make docker-down
```

---

## Option B: Local Development Setup

**Best for:** Active development, code changes, faster iteration

Run services locally for faster development cycles when making code changes.

### 1. Clone the Repository

```bash
git clone https://github.com/ajbeattie/octobud.git
cd octobud
```

### 2. Install Dependencies and Setup

The setup command will install all dependencies and start the database:

```bash
make setup
```

This command:
- Installs Go tools (goose, sqlc, mockgen)
- Installs frontend npm dependencies
- Starts PostgreSQL in Docker
- Runs database migrations

### 3. Configure Environment Variables

Copy the example environment file and add your GitHub token:

```bash
cp .env.example .env
```

Edit `.env` and set your token and JWT secret:

```env
GH_TOKEN=ghp_your_token_here
JWT_SECRET=your_secure_random_secret_key_here  # Generate with: openssl rand -hex 32
# Optional: JWT_EXPIRY=168h  # Default is 7 days (168h). Examples: 720h (30 days), 2160h (90 days)
```

**JWT Secret:**
- **Required for local development** (unlike Docker, it's not auto-generated)
- Generate one: `openssl rand -hex 32`
- Or run `make ensure-jwt-secret` to generate it automatically

**JWT Token Expiration:**
- Default expiration: 7 days (168 hours)
- Active users stay logged in automatically via token refresh
- Service worker continues working even after long periods of tab inactivity
- Inactive users (no activity for 7+ days) will need to log in again
- Customize via `JWT_EXPIRY` environment variable (e.g., `JWT_EXPIRY=720h` for 30 days)

### 4. Start Services (in separate terminals)

**Terminal 1 — Backend API:**
```bash
make backend-dev
```

**Terminal 2 — Frontend:**
```bash
make frontend-dev
```

**Terminal 3 — Worker (syncs notifications):**
```bash
make worker
```

Open http://localhost:5173 in your browser.

**Note:** PostgreSQL runs in Docker even for local development. Manage it with:
```bash
make db-start  # Start database
make db-stop   # Stop database
make db-migrate # Run migrations
```

---

## First Steps

After starting Octobud (using either Docker or local setup):

1. **Login** — Use the default credentials:
   - Username: `octobud`
   - Password: `octobud`
   - **Important:** Change these credentials immediately after first login via the profile avatar dropdown (top right) → "Update credentials"

2. **Wait for Sync** — The worker will automatically start syncing your GitHub notifications (this may take a few minutes depending on how many notifications you have)

3. **View the Inbox** — Your notifications appear in the default inbox view

4. **Try Keyboard Shortcuts** — Press `h` to see available shortcuts

5. **Create a View** — Click the "+" in the sidebar to create a custom filtered view

6. **Update Credentials** — Click your profile avatar (top right) → "Update credentials" to change your username and password

7. **Learn the Basics** — Check out the [Basic Usage Guide](basic-usage.md) to learn about actions, queries, views, tags, rules, and keyboard shortcuts

---

## Quick Reference: Docker vs Local Setup

| Aspect | Docker Setup | Local Development Setup |
|--------|--------------|-------------------------|
| **Prerequisites** | Docker, Docker Compose | Go 1.24+, Node.js 24+, Docker (for PostgreSQL only) |
| **Setup Required** | None - just clone and run | `make setup` to install dependencies |
| **JWT_SECRET** | Auto-generated | Must set manually (or use `make ensure-jwt-secret`) |
| **Environment Config** | Optional (can pass GH_TOKEN to make command) | Required (must create `.env` file) |
| **Starting Services** | Single command (`make docker-up-dev`) | Three separate terminals |
| **Best For** | Quick start, production, minimal setup | Active development, code changes |
| **Database** | Managed by Docker Compose | Runs in Docker (separate from app) |
| **Hot Reload** | Yes (dev mode) | Yes (all services) |

## Available Make Commands

### Setup & Installation

| Command | Description |
|---------|-------------|
| `make setup` | Complete first-time setup (install tools, deps, start DB, run migrations) |
| `make install-tools` | Install Go tools (goose, sqlc, mockgen) |
| `make frontend-install` | Install frontend npm dependencies |
| `make generate` | Generate code (sqlc queries, mocks) |

### Docker

| Command | Description |
|---------|-------------|
| `make docker-up` | Run production stack (port 3000) |
| `make docker-up-dev` | Run dev stack with hot reload (port 5173) |
| `make docker-up-1password` | Run production stack with 1Password token |
| `make docker-up-dev-1password` | Run dev stack with 1Password token |
| `make docker-down` | Stop all containers |

### Local Development

| Command | Description |
|---------|-------------|
| `make backend-dev` | Run backend server locally |
| `make frontend-dev` | Run frontend dev server locally |
| `make worker` | Run GitHub sync worker locally |
| `make dev` | Run backend server (alias for backend-dev) |
| `make dev-prompt` | Run backend server with interactive token prompt |
| `make worker-prompt` | Run worker with interactive token prompt |

### Database

| Command | Description |
|---------|-------------|
| `make db-start` | Start PostgreSQL in Docker |
| `make db-stop` | Stop PostgreSQL |
| `make db-migrate` | Run database migrations |
| `make psql` | Open PostgreSQL shell |

### Utilities

| Command | Description |
|---------|-------------|
| `make format` | Format Go code |
| `make test` | Run all tests (backend and frontend) |
| `make test-backend` | Run backend tests only |
| `make test-frontend` | Run frontend tests only |
| `make clean` | Remove built binaries |

### 1Password Integration

If you use 1Password for secrets management:

```bash
# Production stack
OP_GH_TOKEN='op://Private/GitHub PAT/token' make docker-up-1password

# Development stack with hot reload
OP_GH_TOKEN='op://Private/GitHub PAT/token' make docker-up-dev-1password
```

This starts the stack with your GitHub token fetched from 1Password.

## Next Steps

- [Basic Usage](basic-usage.md) — Learn the core features and concepts
- [Query Syntax](features/query-syntax.md) — Master the query language
- [Keyboard Shortcuts](features/keyboard-shortcuts.md) — Navigate efficiently
- [Views & Rules](features/views-and-rules.md) — Organize your inbox

## Troubleshooting

### Database Connection Issues

```bash
# Check if PostgreSQL is running
docker compose -f docker-compose.dev.yaml ps

# Restart the database
make db-stop
make db-start

# Re-run migrations
make db-migrate
```

### Port Already in Use

If port 5173, 3000, or 8080 is already in use:

```bash
# Find what's using the port
lsof -i :5173

# Stop all containers
make docker-down
```

### Notifications Not Syncing

1. Check your `GH_TOKEN` is set correctly in `.env`
2. Ensure the worker is running (`make worker`)
3. Check the worker logs for errors

### Authentication Issues

1. **Server won't start:** `JWT_SECRET` is automatically generated by Docker targets or `make ensure-jwt-secret`. If running locally without Docker, ensure `JWT_SECRET` is set in `.env` or run `make ensure-jwt-secret`
2. **Can't login:** Default credentials are `octobud` / `octobud` (change these after first login via profile avatar dropdown)
3. **Token expired:** JWT tokens expire after 7 days by default. Active users stay logged in automatically via token refresh. Inactive users will need to log in again after expiration.
4. **Forgot password:** If you've lost access, you can reset by deleting the user from the database and restarting the server (it will recreate the default octobud user)

### Frontend Build Issues

```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
```

