# Deployment Guide

This guide covers deploying Octobud for single-user, self-hosted scenarios.

## Deployment Scenarios

Octobud is designed as a single-user application. Common deployment scenarios include:

- **Local Development** — Running on your local machine for development or personal usage
- **Home Lab** — Running on a server in your home network
- **VPS/Cloud** — Running on a virtual private server or cloud instance

## Security Considerations

> **📚 Security Documentation**  
> For detailed information about all security features, see the [Security Documentation](security/README.md), including:
> - [Security Headers](security/security-headers.md)
> - [Authentication & Password Security](security/authentication.md)
> - [Secure Cookie Auto-Detection](security/secure-cookie-auto-detection.md)
> - [Rate Limiting & DoS Protection](security/rate-limiting.md)

### Default Database Credentials

Octobud uses default PostgreSQL credentials (`postgres:postgres`) by default. These are **fine for local development/usage** but **should be changed** for any network-accessible deployment.

**When to change credentials:**
- Your deployment is accessible from a network (not just localhost)
- The database port (5432) is exposed beyond localhost
- You're deploying on a VPS, cloud instance, or home lab server
- Multiple users have access to the deployment environment

**How to change credentials:**

1. Update `POSTGRES_USER` and `POSTGRES_PASSWORD` in your docker-compose file or environment
2. Update `DATABASE_URL` in all services to use the new credentials:
   ```bash
   DATABASE_URL=postgres://username:strongpassword@host:5432/octobud?sslmode=disable
   ```

The application will log warnings at startup if default credentials are detected, especially if the server is bound to a non-localhost interface.

### Environment Variables

Use environment variables for all sensitive configuration:

```bash
# Required
GH_TOKEN=your_github_token_here
DATABASE_URL=postgres://user:pass@host:5432/octobud?sslmode=disable

# Optional
JWT_SECRET=your_secure_random_secret_key_here  # Auto-generated if not set (or generate with: openssl rand -hex 32)
SERVER_ADDR=:8080
CORS_ALLOWED_ORIGINS=http://yourdomain.com,https://yourdomain.com
SYNC_INTERVAL=20s
JWT_EXPIRY=168h  # Default is 7 days. Examples: 720h (30 days), 2160h (90 days)
```

**JWT Secret:**
- **Automatically generated** when using Docker targets (`make docker-up`, `make docker-up-dev`, etc.) or `make ensure-jwt-secret`
- **Only set manually** if you want to supply your own secret (generate with: `openssl rand -hex 32`)
- The `JWT_SECRET` is required for authentication - the server will not start without it

### Authentication

Octobud uses JWT-based authentication. On first startup, a default user is automatically created:
- **Username:** `octobud`
- **Password:** `octobud`

**Security Best Practices:**
1. **Change default credentials immediately** after first login via the profile avatar dropdown (top right) → "Update credentials"
2. **JWT_SECRET is automatically generated** - Docker targets and `make ensure-jwt-secret` handle this automatically. If setting manually, use a strong random string (at least 32 characters)
3. **Use HTTPS** in production - JWT tokens are sent in HTTP headers and should be encrypted in transit
4. **Store JWT_SECRET securely** - use a secrets manager or environment variable injection (not in code or config files)

**Updating Credentials:**
- Click your profile avatar in the top right corner
- Select "Update credentials" from the dropdown menu
- Enter your current password
- Optionally update username and/or password
- At least one of username or password must be provided

**Token Expiration & Refresh:**
- JWT tokens expire after 7 days by default (configurable via `JWT_EXPIRY` environment variable)
- Active users stay logged in automatically - tokens are refreshed in the background when needed
- Service worker continues working even after long periods of tab inactivity by requesting token refresh
- Inactive users (no activity for 7+ days) will need to log in again
- Tokens are stored in browser localStorage and IndexedDB (for service worker access)
- Customize expiration: `JWT_EXPIRY=720h` (30 days), `JWT_EXPIRY=2160h` (90 days), etc.

## CORS Configuration

CORS (Cross-Origin Resource Sharing) is configured by default to allow:
- Same-origin requests (frontend and backend on same domain)
- Localhost development ports (5173, 3000, 8080)

### Configuring CORS for Production

If your frontend and backend are served from different origins, configure CORS using the `CORS_ALLOWED_ORIGINS` environment variable:

```bash
# Single origin
CORS_ALLOWED_ORIGINS=https://octobud.example.com

# Multiple origins (comma-separated)
CORS_ALLOWED_ORIGINS=https://octobud.example.com,https://www.example.com
```

**Note:** If frontend and backend are served from the same origin (same domain and port), CORS configuration is not needed.

**Request Body Size Limits:**
- Maximum request body size is limited to 1MB to prevent DoS attacks
- This limit is sufficient for all normal Octobud operations
- See [Rate Limiting & DoS Protection](security/rate-limiting.md) for details

## Reverse Proxy Setup

For production deployments, use a reverse proxy (nginx, Caddy, Traefik) to:
- Provide HTTPS/TLS encryption
- Handle domain routing
- Serve static files efficiently

### Example: Caddy Configuration

```caddy
octobud.example.com {
    # Proxy API requests to backend
    reverse_proxy /api/* localhost:8080
    
    # Serve frontend static files
    file_server {
        root /path/to/frontend/build
        try_files {path} /index.html
    }
}
```

### Example: Nginx Configuration

```nginx
server {
    listen 443 ssl http2;
    server_name octobud.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # Proxy API requests
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # Serve frontend
    location / {
        root /path/to/frontend/build;
        try_files $uri $uri/ /index.html;
    }
}
```

## Docker Deployment

### Production Stack

```bash
# Set your GitHub token
export GH_TOKEN=your_token_here

# Update database credentials in docker-compose.yaml
# Then start the stack
make docker-up
```

### Custom Configuration

Create a `.env` file or set environment variables:

```bash
GH_TOKEN=your_token
DATABASE_URL=postgres://user:pass@postgres:5432/octobud?sslmode=disable
CORS_ALLOWED_ORIGINS=https://yourdomain.com
```

## Single-User Considerations

Octobud is designed for single-user use:

- **Single User Account** — Only one user account exists (created automatically)
- **No Multi-Tenancy** — All data belongs to a single user
- **Built-in Authentication** — JWT-based authentication with password protection (see [Authentication & Password Security](security/authentication.md))
- **Network Security** — Additional network-level security (firewall, VPN, etc.) is recommended for network-accessible deployments
- **Access Control** — Consider reverse proxy authentication or network-level restrictions for additional security layers

For network-accessible deployments, consider:
- Using a VPN to access your deployment
- Implementing reverse proxy authentication (Basic Auth, OAuth, etc.)
- Restricting access via firewall rules
- Using SSH tunneling for remote access

## Monitoring and Logging

### Logs

The application logs to stdout/stderr. For production:

- Use Docker logging drivers
- Configure log rotation
- Consider centralized logging (e.g., Loki, ELK stack)

### Health Checks

The application provides a health check endpoint:

```bash
curl http://localhost:8080/healthz
# Returns: ok
```

Use this endpoint for:
- Docker health checks
- Load balancer health checks
- Monitoring systems

## Backup and Recovery

### Database Backups

Regularly backup your PostgreSQL database:

```bash
# Using pg_dump
pg_dump -U postgres octobud > backup.sql

# Using Docker
docker exec postgres pg_dump -U postgres octobud > backup.sql
```

### Restore

```bash
# Restore from backup
psql -U postgres octobud < backup.sql

# Using Docker
docker exec -i postgres psql -U postgres octobud < backup.sql
```

## Troubleshooting

### Database Connection Issues

- Verify `DATABASE_URL` is correct
- Check PostgreSQL is running and accessible
- Verify network connectivity between services

### CORS Errors

- Check `CORS_ALLOWED_ORIGINS` includes your frontend origin
- Verify frontend is making requests to the correct backend URL
- Check browser console for specific CORS error messages

### Sync Not Working

- Verify `GH_TOKEN` is set and valid
- Check worker logs for errors
- Ensure worker process is running
- Verify GitHub token has required scopes (`notifications`, `repo`)

## Next Steps

- [Getting Started](getting-started.md) — Initial setup
- [Query Syntax](features/query-syntax.md) — Using the query language
- [Keyboard Shortcuts](features/keyboard-shortcuts.md) — Navigation tips

