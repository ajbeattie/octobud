# Authentication and Password Security

Octobud uses JWT-based authentication with strong password security practices. This document covers authentication features and password requirements.

## Overview

- **Authentication Method:** JWT (JSON Web Tokens)
- **Password Hashing:** bcrypt with default cost factor
- **Token Storage:** Browser localStorage (frontend) and httpOnly cookies (CSRF tokens)
- **Default Credentials:** Created automatically on first startup

## Password Requirements

When creating or updating passwords, Octobud enforces the following requirements:

### Minimum Length
- **8 characters minimum** (counted as UTF-8 runes, supporting Unicode)
- Prevents weak passwords

### Maximum Length
- **128 characters maximum**
- Prevents DoS attacks via extremely long passwords

### Supported Characters
- Any UTF-8 characters (including Unicode like emoji, non-Latin scripts, etc.)
- Length is counted properly using UTF-8 rune counting

### Password Hashing

Passwords are hashed using **bcrypt** with the default cost factor (10), which provides strong protection against brute-force attacks even if the database is compromised.

**Features:**
- Salted hashes (bcrypt generates unique salts automatically)
- Adaptive cost factor (can be increased as computing power increases)
- Timing-attack resistant comparisons

## Default User

On first startup, Octobud automatically creates a default user:

- **Username:** `octobud`
- **Password:** `octobud`

**Important:** Change these credentials immediately after first login. See [Updating Credentials](#updating-credentials) below.

## Token-Based Authentication

### JWT Tokens

Authentication uses JWT tokens that are:
- Signed with HS256 (HMAC-SHA256)
- Include username and expiration time
- Stored in browser localStorage

### Token Expiration

**Default:** 7 days (configurable via `JWT_EXPIRY` environment variable)

**Behavior:**
- Active users stay logged in automatically - tokens are refreshed in the background
- Inactive users (no activity for 7+ days) will need to log in again
- Tokens can be revoked via logout

**Configuration:**
```bash
# Examples:
JWT_EXPIRY=168h   # 7 days (default)
JWT_EXPIRY=720h   # 30 days
JWT_EXPIRY=2160h  # 90 days
```

### Token Refresh

Tokens are automatically refreshed when:
- The user makes authenticated requests
- The token is approaching expiration
- The service worker needs to refresh the token

The refresh happens transparently without requiring the user to log in again.

### Token Revocation

When you log out:
- The current JWT token is added to a revocation list
- The CSRF token cookie is cleared
- Tokens are automatically cleaned up after expiration

## CSRF Protection

Octobud uses the **double-submit cookie pattern** for CSRF protection:

- CSRF token stored in an httpOnly cookie
- CSRF token also sent in `X-CSRF-Token` header
- Both must match for state-changing requests (POST, PUT, DELETE, PATCH)
- Tokens are compared using constant-time comparison (prevents timing attacks)

**How it works:**
1. On login, a CSRF token is generated and set in a cookie
2. The same token is returned in the login response
3. The frontend includes the token in `X-CSRF-Token` header for all state-changing requests
4. The server verifies the cookie and header match

See [Secure Cookies](secure-cookies.md) for how CSRF cookies are secured.

## Rate Limiting

Login attempts are rate-limited to prevent brute-force attacks:

- **Limit:** 5 attempts per minute per username
- **Window:** 1 minute sliding window
- **Reset:** Automatically reset on successful login

**Behavior:**
- After 5 failed login attempts, further attempts are blocked
- Different usernames have separate rate limits
- Rate limits are tracked in memory (per server instance)
- Old attempts are automatically cleaned up

**Response:** HTTP 429 (Too Many Requests) when rate limit is exceeded

## Updating Credentials

You can update your username and/or password after logging in:

1. Click your profile avatar (top right corner)
2. Select "Update credentials" from the dropdown menu
3. Enter your current password
4. Optionally update username and/or password
5. At least one of username or password must be provided

**Security:**
- Current password is always required
- New password must meet strength requirements (8-128 characters)
- Both username and password can be updated in one operation

## Security Features

### Timing Attack Protection

Password validation uses constant-time comparisons and dummy hash generation to prevent timing attacks that could reveal:
- Whether a username exists
- How much of a password is correct

**Implementation:**
- Always performs bcrypt comparison (even for invalid users)
- Uses dummy hashes to normalize timing
- Doesn't reveal whether username or password is incorrect (returns same error)

### Password Strength Validation

Password validation occurs:
- When updating password
- When updating credentials (if password is changed)
- Uses UTF-8 rune counting (supports Unicode properly)

### Secure Storage

- Passwords are **never stored in plain text**
- Only bcrypt hashes are stored in the database
- Passwords are never logged or transmitted except during authentication

## JWT Secret

The JWT secret is used to sign and verify authentication tokens. It's critical for security.

**Configuration:**
- **Auto-generated:** Automatically generated when using Docker or `make ensure-jwt-secret`
- **Manual:** Can be set via `JWT_SECRET` environment variable
- **Required:** Server will not start without a JWT secret

**Best Practices:**
- Use a strong random string (at least 32 characters)
- Generate with: `openssl rand -hex 32`
- Store securely (environment variables, secrets manager)
- Never commit to version control
- Rotate if compromised

## Security Considerations

### For Local Development
- Default credentials are acceptable
- HTTP connections are fine
- Rate limiting still applies

### For Network-Accessible Deployments
- **Change default credentials immediately**
- Use HTTPS (required for secure cookies)
- Use strong JWT secret
- Consider network-level access controls
- Use reverse proxy authentication if needed

### Token Security
- JWT tokens are stored in localStorage (vulnerable to XSS)
- CSRF protection mitigates token theft risks
- Tokens expire automatically
- Tokens can be revoked on logout

### Password Security
- Passwords are hashed with bcrypt
- Minimum 8 characters required
- No maximum complexity requirements (but longer is better)
- Supports Unicode characters

## Troubleshooting

### Can't Log In

1. **Check credentials:** Default is `octobud:octobud`
2. **Rate limiting:** Wait 1 minute if you've had 5 failed attempts
3. **Token expiration:** Clear browser storage and log in again
4. **Check server logs:** Look for authentication errors

### Password Update Fails

1. **Check current password:** Must be correct
2. **Check new password:** Must be 8-128 characters
3. **Check error message:** Will indicate what's wrong

### Token Issues

1. **Token expired:** Log in again (tokens expire after configured time)
2. **Token invalid:** Clear browser storage and log in again
3. **CSRF token missing:** Log out and log in again to get new tokens

## Related

- [Secure Cookies](secure-cookies.md) - How CSRF cookies are secured
- [Rate Limiting](rate-limiting.md) - Brute-force protection
- [Deployment Guide](../deployment.md) - Deployment security considerations

