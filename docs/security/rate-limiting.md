# Rate Limiting and DoS Protection

Octobud includes built-in protection against brute-force attacks and denial-of-service (DoS) attacks through rate limiting and request size limits.

## Login Rate Limiting

### Overview

Login attempts are rate-limited to prevent brute-force password guessing attacks. The rate limiter tracks attempts per username and blocks further attempts after the limit is reached.

### Limits

- **Maximum attempts:** 5 per username
- **Time window:** 1 minute (sliding window)
- **Scope:** Per username (different usernames have separate limits)

### Behavior

**Normal operation:**
1. First 5 login attempts are allowed
2. 6th attempt within the same minute is blocked
3. After 1 minute passes, attempts are allowed again
4. Successful login resets the counter immediately

**Rate limit exceeded:**
- HTTP 429 (Too Many Requests) status code
- Error message: "Too many login attempts. Please try again later."
- Counter resets after 1 minute OR on successful login

### How It Works

The rate limiter:
- Tracks attempt timestamps per username in memory
- Uses a sliding window (not fixed buckets)
- Automatically cleans up old entries
- Resets on successful authentication

**Example:**
```
10:00:00 - Attempt 1 (allowed)
10:00:15 - Attempt 2 (allowed)
10:00:30 - Attempt 3 (allowed)
10:00:45 - Attempt 4 (allowed)
10:01:00 - Attempt 5 (allowed)
10:01:15 - Attempt 6 (blocked - 429 error)
10:02:00 - Attempt 7 (allowed - 1 minute passed)
```

### Configuration

Rate limiting is **configured automatically** and requires no setup. The limits are:

- 5 attempts per minute per username
- In-memory storage (fast, resets on server restart)

**Note:** Rate limiting is per server instance. In a multi-instance deployment, each instance tracks separately. For local/home lab single-instance deployments, this is sufficient.

## Request Body Size Limits

### Overview

To prevent DoS attacks via extremely large request bodies, Octobud limits the maximum size of request bodies.

### Limits

- **Default maximum:** 1MB (1,048,576 bytes)
- **Applied to:** All API requests
- **Enforcement:** Request body is limited using Go's `http.MaxBytesReader`

### Behavior

**Normal requests:**
- Requests under 1MB are processed normally
- No performance impact

**Oversized requests:**
- Request is rejected before processing
- HTTP 413 (Payload Too Large) or connection error
- Prevents memory exhaustion attacks

### Use Cases

This limit prevents:
- Memory exhaustion attacks (sending huge JSON payloads)
- Slow request attacks (slowly sending large bodies)
- Resource exhaustion DoS attacks

**Typical request sizes:**
- Login: < 100 bytes
- API queries: < 10KB
- Bulk operations: < 100KB
- 1MB is sufficient for all normal operations

### Configuration

Body size limits are **automatically configured** at 1MB. This is sufficient for all normal Octobud operations.

## Rate Limiting Implementation

### In-Memory Storage

Rate limiting uses in-memory storage, which means:
- ✅ Fast performance (no database queries)
- ✅ Low overhead
- ✅ Automatic cleanup
- ⚠️ Resets on server restart
- ⚠️ Not shared across multiple instances

**For single-instance deployments** (local/home lab): This is ideal.

**For multi-instance deployments:** Each instance tracks separately. Consider external rate limiting (nginx, Cloudflare, etc.) at the reverse proxy level if needed.

### Automatic Cleanup

Old rate limit entries are automatically cleaned up:
- Runs every 5 minutes
- Removes attempts outside the time window
- Prevents memory leaks

### Reset on Success

When a user successfully logs in:
- Rate limit counter is immediately reset
- User can make another 5 attempts immediately if needed
- Prevents legitimate users from being blocked after a few typos

## Troubleshooting

### Rate Limit Errors

**Error:** "Too many login attempts. Please try again later."

**Solutions:**
1. Wait 1 minute for the rate limit to reset
2. Use the correct username/password to reset immediately
3. Check if multiple devices/apps are attempting login simultaneously

### Request Too Large Errors

**Error:** HTTP 413 or connection errors when sending requests

**Solutions:**
1. Check request size (should be under 1MB)
2. Reduce data in request body
3. Split large operations into smaller requests

**Note:** 1MB should be more than sufficient for all normal Octobud operations. If you're hitting this limit, you may be sending unnecessary data.

### Rate Limits Not Working

**Symptoms:**
- Can make unlimited login attempts
- Rate limiting appears ineffective

**Possible causes:**
1. Server restart reset the in-memory rate limits
2. Multiple server instances (each tracks separately)
3. Rate limiter not initialized (shouldn't happen)

**Solutions:**
1. Check server logs for rate limiter initialization
2. Verify rate limiting middleware is applied to login endpoint
3. Test with multiple rapid login attempts

## Security Considerations

### For Local Development

Rate limiting still applies in development:
- Helps prevent accidental account lockouts
- Tests rate limiting behavior
- Can wait 1 minute or use correct password to reset

### For Production

**Single-instance deployments:**
- In-memory rate limiting is sufficient
- Fast and efficient
- Automatically protects against brute-force attacks

**Multi-instance deployments:**
- Each instance tracks separately
- Consider additional rate limiting at reverse proxy level
- Or use external rate limiting service

**Network-level protection:**
- Consider firewall rules
- Use reverse proxy rate limiting (nginx, Caddy)
- Consider Cloudflare or similar for DDoS protection

## Best Practices

1. **Don't disable rate limiting** — It's a critical security feature
2. **Monitor rate limit hits** — Frequent hits may indicate attack attempts
3. **Use strong passwords** — Even with rate limiting, strong passwords are essential
4. **Monitor server logs** — Watch for patterns of failed login attempts
5. **Consider network-level controls** — For additional protection beyond application-level rate limiting

## Related Documentation

- [Authentication and Password Security](authentication.md) — Password requirements and authentication
- [Deployment Guide](../deployment.md) — Production deployment considerations
- [Security Overview](README.md) — Complete security documentation

