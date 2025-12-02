# Deployment Guide

This guide covers deploying Octobud for single-user, self-hosted scenarios.

## Quick Start with Docker

The fastest way to deploy Octobud:

```bash
git clone https://github.com/ajbeattie/octobud.git && cd octobud
cp .env.example .env
# Edit .env and set GH_TOKEN to your GitHub token
make docker-up
```

Open `http://localhost:3000` and login with `octobud`/`octobud`.

> [!IMPORTANT]
> Change the default credentials after first login (profile avatar → "Update credentials").

### Deployment Options

| Command | Description | Port |
|---------|-------------|------|
| `make docker-up` | Production build | 3000 |
| `make docker-up-dev` | Development with hot reload | 5173 |
| `make docker-up-1password` | Production with 1Password | 3000 |

## Configuration

### Environment Variables

Configure via `.env` file:

| Variable | Required | Description |
|----------|----------|-------------|
| `GH_TOKEN` | Yes | GitHub PAT with `notifications` and `repo` scopes |
| `JWT_SECRET` | No | Auto-generated if not set |
| `JWT_EXPIRY` | No | Token lifetime (default: `168h` = 7 days) |
| `DATABASE_URL` | No | PostgreSQL connection string (has default) |
| `CORS_ALLOWED_ORIGINS` | No | Comma-separated origins for CORS |
| `SYNC_INTERVAL` | No | How often to sync (default: `30s`) |
| `SERVER_ADDR` | No | Server bind address (default: `:8080`) |

### Database Credentials

Default PostgreSQL credentials are `postgres:postgres`. This is **fine for local use** but should be changed for network-accessible deployments.

**To change credentials:**
1. Update `POSTGRES_USER` and `POSTGRES_PASSWORD` in docker-compose.yaml
2. Update `DATABASE_URL` in `.env`:
   ```
   DATABASE_URL=postgres://newuser:newpass@postgres:5432/octobud?sslmode=disable
   ```

### CORS Configuration

CORS is pre-configured for localhost development. For custom domains:

```bash
# Single origin
CORS_ALLOWED_ORIGINS=https://octobud.example.com

# Multiple origins
CORS_ALLOWED_ORIGINS=https://octobud.example.com,https://www.example.com
```

Not needed if frontend and backend are served from the same origin (the default Docker setup).

## Security

For detailed security documentation, see [Security](security/README.md).

### Security Checklist

When deploying outside localhost:

- [ ] Change default user credentials (`octobud`/`octobud`)
- [ ] Change default database credentials (`postgres`/`postgres`)
- [ ] Use HTTPS via reverse proxy
- [ ] Restrict network access (firewall, VPN)

### Authentication

- Default user: `octobud` / `octobud` (change immediately)
- JWT tokens expire after 7 days (configurable via `JWT_EXPIRY`)
- Tokens refresh automatically for active users
- See [Authentication](security/authentication.md) for details

## Reverse Proxy Setup

For HTTPS and custom domains, use a reverse proxy in front of Octobud.

### Caddy (Recommended)

```caddy
octobud.example.com {
    reverse_proxy localhost:3000
}
```

Caddy automatically handles HTTPS certificates.

### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name octobud.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Operations

### Health Checks

```bash
curl http://localhost:3000/healthz
# Returns: ok
```

### Logs

The application logs to stdout/stderr. For production:
- Use Docker logging drivers
- Configure log rotation
- Consider centralized logging (Loki, ELK)

### Backups

```bash
# Backup
docker exec postgres pg_dump -U postgres octobud > backup.sql

# Restore
docker exec -i postgres psql -U postgres octobud < backup.sql
```

## Troubleshooting

### Sync Not Working

- Verify `GH_TOKEN` is set in `.env`
- Check worker logs: `docker logs octobud-worker-1`
- Verify token has `notifications` and `repo` scopes

### Database Connection Issues

- Verify PostgreSQL is running: `docker ps`
- Check `DATABASE_URL` matches your credentials
- Verify network connectivity between containers

### CORS Errors

- Set `CORS_ALLOWED_ORIGINS` to include your frontend origin
- Check browser console for specific error messages

## Related

- [Start Here](start-here.md) - Initial setup and core workflows
- [Security](security/README.md) - Security features and best practices
- [Contributing](CONTRIBUTING.md) - Development setup
