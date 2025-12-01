# Security Policy

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

**Please do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please use GitHub's Security Advisory feature:

1. Go to the [Security tab](https://github.com/ajbeattie/octobud/security) on the repository
2. Click **"Report a vulnerability"**
3. Fill out the security advisory form with details about the vulnerability

Alternatively, you can navigate directly to: https://github.com/ajbeattie/octobud/security/advisories/new

Include in your report:

1. **Description** — What is the vulnerability?
2. **Steps to Reproduce** — How can we reproduce it?
3. **Impact** — What is the potential impact?
4. **Suggested Fix** — If you have one (optional)

### What to Expect

- **Acknowledgment** — We'll acknowledge receipt within 48 hours
- **Updates** — We'll keep you informed of our progress
- **Resolution** — We aim to resolve critical issues within 7 days
- **Credit** — We'll credit you in the release notes (unless you prefer anonymity)

### Disclosure Policy

- We follow a coordinated disclosure process
- Please give us reasonable time to address the issue before public disclosure
- We typically aim for disclosure within 90 days or when a fix is released

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < 1.0   | :x:                |

## Security Best Practices

When self-hosting Octobud:

1. **Keep Updated** — Run the latest version
2. **Secure Your Token** — Protect your GitHub PAT; don't commit it to version control
3. **Database Security** — Use strong PostgreSQL credentials in production
4. **Network Security** — Use HTTPS in production; consider a reverse proxy
5. **Access Control** — Limit who can access your Octobud instance

## Known Security Considerations

- **GitHub Token Scope** — Octobud requires `notifications` and `repo` scopes, which grants read access to your repositories
- **Self-Hosted** — You are responsible for securing your own deployment
- **Built-in Authentication** — Octobud includes JWT-based authentication with password protection, CSRF protection, and rate limiting. See [Authentication & Password Security](docs/security/authentication.md) for details
- **Single-User Application** — Octobud is designed for single-user use. For network-accessible deployments, consider additional security layers (VPN, reverse proxy authentication, firewall rules)

Thank you for helping keep Octobud secure!

