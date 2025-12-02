# Security Documentation

Octobud is designed as a **single-user, self-hosted application** for local deployment, home labs, or VPN-accessible deployments. This directory documents security features and best practices for deployments beyond localhost.

## Security Features

Octobud includes the following security features:

- **JWT-based authentication** with token expiration and refresh
- **Password hashing** with bcrypt
- **CSRF protection** using double-submit cookie pattern
- **Rate limiting** on login attempts (5 per minute)
- **Request body size limits** (1MB max)
- **Security headers** (CSP, HSTS, XSS protection, etc.)
- **Automatic HTTPS detection** for secure cookies

## Documentation

- **[Authentication](authentication.md)** - Password requirements, JWT tokens, CSRF protection
- **[Rate Limiting](rate-limiting.md)** - Brute-force and DoS attack prevention
- **[Secure Cookies](secure-cookies.md)** - Automatic HTTPS detection for cookies
- **[Security Headers](security-headers.md)** - HTTP security headers and browser protection

## Security Checklist

When deploying Octobud outside of localhost:

- [ ] **Change default credentials** - Update the default `octobud:octobud` user immediately
- [ ] **Use HTTPS** - Set up a reverse proxy (nginx/Caddy) with TLS certificates
- [ ] **Change database credentials** - Replace default `postgres:postgres` credentials
- [ ] **Set strong JWT secret** - Auto-generated is fine, or use `openssl rand -hex 32`
- [ ] **Configure CORS** - Set `CORS_ALLOWED_ORIGINS` if frontend/backend are on different origins
- [ ] **Network access controls** - Use firewall rules or VPN to limit access

## Deployment Risk Levels

| Scenario | Risk | Key Actions |
|----------|------|-------------|
| **Local Development** | Low | Default credentials acceptable, HTTP fine |
| **Home Lab / Private Network** | Medium | Change credentials, use HTTPS if externally accessible |
| **VPS / Cloud / Public** | High | All security measures required, consider additional authentication |

## Reporting Security Issues

If you discover a security vulnerability:

- **Do NOT** open a public GitHub issue
- Report via [GitHub Security Advisories](https://github.com/ajbeattie/octobud/security/advisories/new)
- See [SECURITY.md](../../SECURITY.md) for details

## Related

- [Deployment Guide](../deployment.md) - Complete deployment instructions
