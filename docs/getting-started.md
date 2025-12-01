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

### 1. Clone and Configure

```bash
git clone https://github.com/ajbeattie/octobud.git
cd octobud
```

### 2. Configure Environment (Optional)

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

### 1. Clone and Setup

```bash
git clone https://github.com/ajbeattie/octobud.git
cd octobud
```

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
- Default: 7 days. Active users stay logged in automatically
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

1. **Login** — Use default credentials `octobud` / `octobud` and change them immediately after first login (profile avatar → "Update credentials")

2. **Wait for Sync** — Notifications sync automatically (may take a few minutes initially)

3. **Get Started** — Press `h` for keyboard shortcuts, check out the [Basic Usage Guide](basic-usage.md) for core features

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

## Common Commands

**Docker:**
- `make docker-up-dev` — Start development stack (port 5173)
- `make docker-up` — Start production stack (port 3000)
- `make docker-down` — Stop all containers

**Local Development:**
- `make setup` — First-time setup (installs tools, dependencies, starts DB)
- `make backend-dev` — Run backend server
- `make frontend-dev` — Run frontend dev server
- `make worker` — Run sync worker

**1Password Integration:**
```bash
OP_GH_TOKEN='op://Private/GitHub PAT/token' make docker-up-dev-1password
```

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

- **Server won't start:** `JWT_SECRET` auto-generates with Docker or run `make ensure-jwt-secret` for local dev
- **Can't login:** Default credentials are `octobud` / `octobud`
- **Token expired:** Active users stay logged in automatically; inactive users (>7 days) need to log in again

### Frontend Build Issues

```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
```

