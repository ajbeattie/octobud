# Secure Cookie Auto-Detection

Octobud automatically sets the `Secure` flag on authentication cookies based on whether requests are made over HTTPS. This ensures cookies are only sent over encrypted connections, protecting them from interception.

## What Are Secure Cookies?

The `Secure` flag on cookies tells browsers to **only send the cookie over HTTPS connections**, never over plain HTTP. This prevents cookie theft via man-in-the-middle attacks on unencrypted connections.

## How It Works

Octobud automatically detects HTTPS connections and sets the `Secure` flag accordingly:

- **HTTPS requests** → Cookies are marked as secure (only sent over HTTPS)
- **HTTP requests** → Cookies are not marked as secure (can be sent over HTTP)

The detection works automatically when your server is behind a reverse proxy (like nginx or Caddy) that terminates TLS. The proxy forwards information about the original HTTPS connection, and Octobud uses that to set cookies appropriately.

## Detection Methods

Octobud detects HTTPS connections by checking (in priority order):

1. **`X-Forwarded-Proto: https` header** - Set by most reverse proxies (nginx, Caddy, Traefik)
2. **Direct TLS connection** - When the Go server handles TLS directly
3. **`X-Forwarded-Ssl: on` header** - Some proxies use this format
4. **RFC 7239 `Forwarded` header** - Standardized proxy header

## Deployment Scenarios

### Local Development (HTTP)

When running locally on HTTP:

```
Client → Octobud Server (http://localhost:8080)
```

- Cookies work normally on HTTP
- No special configuration needed
- Suitable for local development and testing

### Production Behind Reverse Proxy (HTTPS)

When deployed behind a reverse proxy that terminates TLS:

```
Client (HTTPS) → Nginx/Caddy (terminates TLS) → Octobud (HTTP on localhost:8080)
```

**Requirements:**

1. Your reverse proxy must forward the protocol information. Most proxies do this automatically, but verify your configuration includes:

   **Nginx:**
   ```nginx
   proxy_set_header X-Forwarded-Proto $scheme;
   ```

   **Caddy:**
   ```caddy
   # Automatically adds X-Forwarded-Proto
   reverse_proxy localhost:8080
   ```

2. Octobud will automatically detect HTTPS and set secure cookies
3. No additional configuration needed

### Direct TLS (HTTPS)

When Octobud handles TLS directly:

```
Client → Octobud Server (HTTPS on port 8443)
```

- Octobud detects the TLS connection directly
- Secure cookies are set automatically
- No configuration needed

## Configuration

### Automatic Detection (Default)

By default, Octobud automatically detects HTTPS and sets secure cookies accordingly. **No configuration is required** in most deployments.

### Force Secure Cookies

If you need to force secure cookies even on HTTP connections (for example, during testing), set the environment variable:

```bash
SECURE_COOKIES=true
```

**When to use this:**
- Testing cookie behavior
- Forcing secure cookies in specific test environments
- Overriding auto-detection if needed

**Note:** In production, you typically don't need this setting because HTTPS detection works automatically behind reverse proxies.

## Troubleshooting

### Cookies Not Working Behind Reverse Proxy

If cookies aren't being set securely behind a reverse proxy:

1. **Verify proxy headers:**
   - Check that your reverse proxy forwards `X-Forwarded-Proto: https`
   - Most proxies do this automatically, but verify your configuration

2. **Test HTTPS detection:**
   - Make a request through your reverse proxy
   - Check server logs to see if HTTPS is detected
   - Verify cookies in browser DevTools have the `Secure` flag set

3. **Common proxy configurations:**

   **Nginx:**
   ```nginx
   location /api/ {
       proxy_pass http://localhost:8080;
       proxy_set_header Host $host;
       proxy_set_header X-Real-IP $remote_addr;
       proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
       proxy_set_header X-Forwarded-Proto $scheme;  # ← Required
   }
   ```

   **Caddy:**
   ```caddy
   reverse_proxy localhost:8080 {
       # Automatically forwards X-Forwarded-Proto
   }
   ```

### Cookies Not Working in Development

If you're testing locally on HTTP:

- This is expected behavior - cookies without the `Secure` flag work on HTTP
- Secure cookies will be set automatically when you deploy behind HTTPS
- For testing secure cookie behavior, use `SECURE_COOKIES=true` or deploy behind HTTPS

## Related

- [Deployment Guide](../deployment.md) - Full deployment instructions
- [Security Headers](security-headers.md) - HTTP security headers

