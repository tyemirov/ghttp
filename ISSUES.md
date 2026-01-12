# ISSUES (Append-only Log)

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

## Features (100–199)

### 100: Reverse proxy with WebSocket support
**Status:** Done

Add reverse proxy capability to ghttp with full WebSocket support. This enables ghttp to serve static files while proxying API requests (including WebSocket connections) to a backend server.

**Configuration:**
- `GHTTP_SERVE_PROXY_BACKEND` - Backend URL (e.g., `http://backend:8001`)
- `GHTTP_SERVE_PROXY_PATH_PREFIX` - Path prefix to proxy (e.g., `/api/`)

Or via flags:
- `--proxy-backend` - Backend URL to proxy requests to
- `--proxy-path` - Path prefix to proxy

**Implementation:**
- Proxy handler with automatic WebSocket upgrade detection
- HTTP reverse proxy using httputil.ReverseProxy
- WebSocket proxying via TCP connection hijacking
- Falls back to static file serving for non-matching paths

## Improvements (200–299)

## BugFixes (300–399)

## Maintenance (400–499)

## Planning
