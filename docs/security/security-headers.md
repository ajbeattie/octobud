# Security Headers

Octobud automatically adds security headers to all HTTP responses to protect against common web vulnerabilities. These headers are applied to all responses from both the API server and frontend server.

## Overview

Security headers provide defense-in-depth protection by instructing browsers how to handle content and connections. Octobud includes security headers enabled by default.

## Headers Included

### X-Content-Type-Options: nosniff

Prevents browsers from MIME-type sniffing, which could lead to XSS attacks.

**What it does:** Tells browsers to respect the declared `Content-Type` header and not try to guess the content type.

**Impact:** Prevents browsers from treating text files as executable scripts if the content type is incorrectly declared.

### X-Frame-Options: SAMEORIGIN

Prevents clickjacking attacks while allowing same-origin embedding.

**What it does:** Controls whether the page can be embedded in frames/iframes.

**Value:** `SAMEORIGIN` - allows embedding only from the same origin (good for local/home lab deployments that might embed the app).

**Impact:** Prevents malicious sites from embedding your Octobud instance in a frame to trick users into clicking on hidden elements.

### X-XSS-Protection: 1; mode=block

Enables XSS filtering in older browsers.

**What it does:** Activates built-in XSS protection in browsers that support it (primarily older browsers).

**Impact:** Provides additional protection against reflected XSS attacks in legacy browsers.

### Content-Security-Policy (CSP)

Restricts resources that can be loaded and executed.

**What it does:** Defines which sources are allowed for scripts, styles, images, fonts, and other resources.

**Octobud's CSP Policy:**
```
default-src 'self';
script-src 'self' 'unsafe-inline' 'unsafe-eval';
style-src 'self' 'unsafe-inline';
img-src 'self' data: https://github.com https://avatars.githubusercontent.com https://*.githubusercontent.com https://dependabot-badges.githubapp.com;
font-src 'self' data:;
connect-src 'self';
frame-ancestors 'self';
base-uri 'self';
form-action 'self'
```

**Notes:**
- `unsafe-inline` and `unsafe-eval` are required for SvelteKit's hydration mechanism
- Allows data URIs for images/icons (common in SvelteKit)
- Allows GitHub domains for images via `<img>` tags:
  - `github.com` - for `github.com/username.png` format (redirects to avatar)
  - `avatars.githubusercontent.com` - direct avatar URLs from GitHub API
  - `*.githubusercontent.com` - wildcard for any other GitHub CDN subdomains
  - `dependabot-badges.githubapp.com` - Dependabot status badges in markdown content
- `connect-src` restricts to same-origin only (API calls and service worker polling)
- Prevents framing from other origins

**Impact:** Significantly reduces the attack surface by restricting resource loading. While `unsafe-inline` is less restrictive, it's necessary for SvelteKit to function correctly.

### Strict-Transport-Security (HSTS)

Forces browsers to use HTTPS for future connections.

**What it does:** Tells browsers to only connect via HTTPS for a specified duration.

**Behavior:**
- Only set when HTTPS is detected (auto-detected from request)
- Set to `max-age=31536000; includeSubDomains` (1 year)
- Prevents accidental use on HTTP-only deployments

**Impact:** Once a browser has seen this header over HTTPS, it will only connect via HTTPS for the next year, even if the user types `http://`.

**When it's set:**
- Direct HTTPS connections (server handles TLS)
- Behind reverse proxy with `X-Forwarded-Proto: https` header
- Behind reverse proxy with `X-Forwarded-Ssl: on` header
- Behind reverse proxy with RFC 7239 `Forwarded` header

### Referrer-Policy: strict-origin-when-cross-origin

Controls referrer information sent with requests.

**What it does:** Limits what referrer information is included when navigating to other sites.

**Value:** `strict-origin-when-cross-origin`
- Same-origin requests: Send full URL
- Cross-origin HTTPS: Send only origin
- Cross-origin HTTP: Send nothing

**Impact:** Prevents leaking sensitive path information to external sites while maintaining functionality for same-origin navigation.

### Permissions-Policy

Disables access to browser features.

**What it does:** Prevents the application from accessing various browser features.

**Disabled features:**
- Geolocation
- Microphone
- Camera
- Payment APIs
- USB
- Magnetometer
- Gyroscope
- Speaker

**Impact:** Prevents the application from accessing features it doesn't need, reducing the attack surface.

## Automatic HTTPS Detection

Security headers that depend on HTTPS (like HSTS) are automatically enabled when HTTPS is detected. Detection works by checking:

1. `X-Forwarded-Proto: https` header (from reverse proxies)
2. Direct TLS connection
3. `X-Forwarded-Ssl: on` header
4. RFC 7239 `Forwarded` header

This means headers work correctly even when Octobud is behind a reverse proxy that terminates TLS.

## Configuration

Security headers are **enabled by default** and require no configuration. They are automatically applied to all HTTP responses.

### Behind Reverse Proxy

When using a reverse proxy (nginx, Caddy, etc.), ensure your proxy forwards the protocol information:

**Nginx:**
```nginx
proxy_set_header X-Forwarded-Proto $scheme;
```

**Caddy:**
```caddy
# Automatically adds X-Forwarded-Proto
reverse_proxy localhost:8080
```

With proper proxy configuration, security headers (including HSTS) will be set correctly.

## Verification

### Checking Headers

You can verify security headers are being set using:

**Command line:**
```bash
curl -I https://your-domain.com/api/healthz
```

**Browser DevTools:**
1. Open browser DevTools (F12)
2. Go to Network tab
3. Make a request
4. Click on the request
5. View Response Headers

### Expected Headers

For HTTPS requests, you should see:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: SAMEORIGIN`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy: ...`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: ...`

For HTTP requests (development), you won't see `Strict-Transport-Security`, but all other headers will be present.

## Security Notes

- **HTTPS is required** for HSTS to take effect
- Security headers provide defense-in-depth - they complement but don't replace good application security practices
- CSP is configured to allow SvelteKit to function - `unsafe-inline` is necessary for hydration
- Headers are automatically applied - no configuration needed in most cases

## Related

- [Secure Cookies](secure-cookies.md) - How cookies are secured
- [Deployment Guide](../deployment.md) - Full deployment instructions

