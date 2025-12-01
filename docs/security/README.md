# Security Documentation

This directory contains security documentation for deploying Octobud outside of a local development environment.

## Quick Reference

Octobud is designed as a **single-user, self-hosted application** for local development, home labs, or VPN-accessible deployments. When deploying to a network-accessible environment, follow the security best practices documented here.

## Security Features

Octobud includes security features:

✅ **Authentication & Authorization**
- JWT-based authentication
- Password requirements (8-128 characters)
- bcrypt password hashing
- Token expiration and refresh
- Token revocation on logout

✅ **Protection Against Attacks**
- CSRF protection (double-submit cookie pattern)
- Rate limiting (5 login attempts per minute)
- Request body size limits (1MB max)
- Security headers (CSP, HSTS, XSS protection, etc.)

✅ **Secure Communications**
- Automatic HTTPS detection
- Secure cookie auto-detection
- CORS configuration
- HSTS (HTTP Strict Transport Security)

## Documentation Index

### Core Security Features

- **[Security Headers](security-headers.md)** — HTTP security headers and browser protection
- **[Secure Cookie Auto-Detection](secure-cookie-auto-detection.md)** — Automatic HTTPS detection for cookies
- **[Authentication and Password Security](authentication.md)** — Password requirements, JWT tokens, CSRF protection
- **[Rate Limiting and DoS Protection](rate-limiting.md)** — Brute-force and DoS attack prevention

### Deployment Security

- **[Deployment Guide](../deployment.md)** — Complete deployment instructions with security considerations
- **[Getting Started Guide](../getting-started.md)** — Initial setup and configuration

## Security Checklist for Network-Accessible Deployments

When deploying Octobud outside of localhost:

- [ ] **Change default credentials** — Update the default `octobud:octobud` user immediately
- [ ] **Use HTTPS** — Set up a reverse proxy (nginx/Caddy) with TLS certificates
- [ ] **Change database credentials** — Replace default `postgres:postgres` credentials
- [ ] **Set strong JWT secret** — Auto-generated is fine, or use `openssl rand -hex 32`
- [ ] **Configure CORS** — Set `CORS_ALLOWED_ORIGINS` if frontend/backend are on different origins
- [ ] **Network access controls** — Use firewall rules or VPN to limit access
- [ ] **Monitor logs** — Watch for failed login attempts and suspicious activity
- [ ] **Keep updated** — Run the latest version to receive security updates

## Deployment Scenarios

### Local Development (Low Risk)
- Default credentials acceptable
- HTTP connections fine
- No special configuration needed

### Home Lab / Private Network (Medium Risk)
- Change default credentials
- Use HTTPS if accessible outside your network
- Consider VPN access only
- Change database credentials

### VPS / Cloud / Public Access (High Risk)
- **All security measures required**
- Use HTTPS (required)
- Change all default credentials
- Use strong passwords and JWT secret
- Consider reverse proxy authentication
- Implement network-level access controls
- Regular security updates

## Security Architecture

### Authentication Flow

1. User submits username/password
2. Rate limiter checks attempt count
3. Password validated (with timing attack protection)
4. JWT token generated (expires in 7 days by default)
5. CSRF token generated and set in httpOnly cookie
6. Tokens returned to frontend

### Request Flow (Authenticated)

1. Request includes JWT token in `Authorization` header
2. JWT middleware validates token
3. Checks token revocation list
4. CSRF middleware validates token (for state-changing requests)
5. Request processed with authenticated user context

### Protection Layers

1. **Network Layer** — Firewall, VPN, reverse proxy
2. **Application Layer** — Rate limiting, authentication, authorization
3. **Transport Layer** — HTTPS/TLS encryption
4. **Browser Layer** — Security headers, secure cookies, CSP

## Known Security Considerations

### Single-User Application

Octobud is designed for single-user use:
- Only one user account exists
- No multi-user or multi-tenant support
- Authentication provides basic access control

### Token Storage

JWT tokens are stored in browser localStorage:
- Vulnerable to XSS attacks if the frontend is compromised
- CSRF protection mitigates some risks
- Tokens expire automatically
- Can be revoked on logout

**Mitigation:**
- Security headers (CSP) help prevent XSS
- CSRF protection prevents token theft from other sites
- Token expiration limits exposure window

### In-Memory Rate Limiting

Rate limiting uses in-memory storage:
- Fast and efficient
- Resets on server restart
- Not shared across multiple instances

**For single-instance deployments:** Ideal solution.

**For multi-instance:** Consider reverse proxy rate limiting.

### Default Credentials

Default credentials are created automatically:
- Username: `octobud`
- Password: `octobud`

**Must be changed** for any network-accessible deployment.

## Security Best Practices

### For All Deployments

1. **Keep updated** — Run the latest version
2. **Strong passwords** — Use unique, complex passwords
3. **Secure secrets** — Store JWT_SECRET and database credentials securely
4. **Monitor access** — Review logs regularly

### For Network-Accessible Deployments

1. **Use HTTPS** — Always use TLS/SSL encryption
2. **Reverse proxy** — Use nginx/Caddy for TLS termination
3. **Access controls** — Limit network access (firewall, VPN)
4. **Change defaults** — Update all default credentials
5. **Regular updates** — Keep application and dependencies updated

### For Public-Facing Deployments

1. **All of the above** plus:
2. **Additional authentication** — Consider reverse proxy authentication (Basic Auth, OAuth)
3. **DDoS protection** — Use Cloudflare or similar
4. **Intrusion detection** — Monitor for suspicious activity
5. **Backup strategy** — Regular database backups
6. **Incident response** — Plan for security incidents

## Getting Help

### Security Issues

If you discover a security vulnerability:
- **Do NOT** open a public GitHub issue
- Report via [GitHub Security Advisories](https://github.com/ajbeattie/octobud/security/advisories/new)
- See [SECURITY.md](../../SECURITY.md) for details

### Configuration Questions

For deployment and configuration questions:
- Check the [Deployment Guide](../deployment.md)
- Review the specific security documentation in this directory
- Open a GitHub discussion for questions

## Additional Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/) — Common web vulnerabilities
- [OWASP Secure Headers Project](https://owasp.org/www-project-secure-headers/) — Security headers reference
- [Mozilla Web Security Guidelines](https://infosec.mozilla.org/guidelines/web_security) — Web security best practices

## Version Information

This documentation applies to Octobud version 1.0 and later. Security features are continuously improved — check the changelog for updates.

