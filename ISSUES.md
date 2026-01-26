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

### 101: Repeatable proxy mappings via --proxy
**Status:** Unresolved

Add repeatable proxy mappings with explicit from/to semantics (`--proxy /api=http://backend:8081`) and support `GHTTP_SERVE_PROXIES=/api=http://...,/ws=http://...` for configuration. Keep from/to behavior explicit and allow multiple mappings.

## Improvements (200–299)

### 201: Config file env var
**Status:** Resolved

Add a `GHTTP_CONFIG_FILE` environment variable counterpart to the `--config` flag so the configuration file path can be set via environment.

## BugFixes (300–399)

## Maintenance (400–499)

### 401: Remove manual HTTPS CLI workflow
**Status:** Resolved

Drop the `ghttp https` CLI subcommands so `--https` is the single self-managed certificate path. Keep internal HTTPS plumbing for automated flow.

### 402: Refresh docker-compose integration doc accuracy
**Status:** Resolved

Align `docs/docker-compose-ai-agents.md` with the current image name, configuration precedence, config file environment variable, legacy proxy settings, and browse behavior.

## Planning
