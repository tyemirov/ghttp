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

### 301: Parse comma-delimited proxy mappings in env
**Status:** Resolved

Normalize `GHTTP_SERVE_PROXIES` so comma-delimited values are split into individual mappings for the proxy route parser.

### 302: Browse mode cannot open index.html from directory listing
**Status:** Unresolved

When running `ghttp --browse`, clicking `index.html` from the rendered directory listing redirects to `./` instead of returning the file content.

Observed behavior:
- `GET /` returns directory listing in browse mode (expected)
- `GET /index.html` returns `301 Location: ./` (unexpected for direct file navigation from listing link)

Repro:
- Create a directory with `index.html`
- Run `ghttp --directory <dir> --browse`
- Open `/<index.html>` from listing or request it directly
- Observe redirect loop back to root listing

Likely cause:
- Handler chain delegates direct file requests to `http.FileServer`, which canonicalizes `/index.html` to `/` by redirect.
- Browse listing currently emits direct `index.html` links, so users cannot view that file in browse mode.

Expected:
- In `--browse` mode, direct file request from listing should render file content, including `index.html`, or listing should avoid emitting links that resolve into canonical redirect loops.

### 303: Serve direct index files in browse mode without canonical redirect
**Status:** Resolved

Add browse-handler interception for direct requests to directory index file names (`index.html`, `index.htm`) so those requests are served as file content instead of being redirected to the directory listing by `http.FileServer`.

Validation:
- Added integration coverage for `GET /example/index.html` in browse mode to assert `200` response with direct file content and no `Location` redirect header.
- Ran `go vet ./...` and `go test ./...`.

## Maintenance (400–499)

### 401: Remove manual HTTPS CLI workflow
**Status:** Resolved

Drop the `ghttp https` CLI subcommands so `--https` is the single self-managed certificate path. Keep internal HTTPS plumbing for automated flow.

### 402: Refresh docker-compose integration doc accuracy
**Status:** Resolved

Align `docs/docker-compose-ai-agents.md` with the current image name, configuration precedence, config file environment variable, legacy proxy settings, and browse behavior.

## Planning
