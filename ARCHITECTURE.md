# gHTTP Architecture

## Overview
gHTTP is a single-process Go server with one primary runtime path:
- parse CLI/configuration with Cobra + Viper
- resolve validated serve configuration
- compose HTTP handlers around `http.FileServer`
- optionally enable TLS and proxying
- run until process signal or context cancellation

The implementation lives mainly under:
- `cmd/ghttp` (entrypoint)
- `internal/app` (CLI, config, orchestration)
- `internal/server` (HTTP handler composition and server runtime)
- `internal/certificates` (dynamic HTTPS CA/cert install + cleanup)
- `pkg/logging` (zap-backed logging abstraction)

## Configuration and startup
Configuration is built in `internal/app` with this precedence:
1. defaults set in code
2. config file (`~/.config/ghttp/config.yaml` by default)
3. environment variables (`GHTTP_...`)
4. CLI flags and positional argument

Key points:
- `--config` and `GHTTP_CONFIG_FILE` select a specific config file path.
- The optional positional argument is interpreted as port when numeric; otherwise it is treated as an initial file to serve.
- `--https` (dynamic HTTPS) is mutually exclusive with `--tls-cert` + `--tls-key`.
- Reverse proxy config supports repeatable mappings via `--proxy` and `GHTTP_SERVE_PROXIES`; legacy single mapping (`--proxy-path` + `--proxy-backend`) remains supported.
- Route-scoped response headers are configured via repeatable `--response-header` mappings (`/path=Header-Name:Header-Value`).
- Route-scoped proxy streaming mode is configured via repeatable `--proxy-streaming` mappings (`/path=unbuffered|buffered`).

## Request pipeline
The runtime handler chain is assembled in `internal/server/file_server.go`.

Base:
- `http.FileServer(http.Dir(directory))`

Conditional wrappers (inside-out):
1. Markdown wrapper (`markdown_handler`) or directory guard (`directory_guard_handler`)
2. Browse wrapper (`browse_handler`) when `--browse` is enabled
3. Initial file wrapper (`initial_file_handler`) when a startup file path is provided and browse mode is off
4. Proxy wrapper (`proxy_handler`) when proxy routes are configured
5. Response headers wrapper (`Server: ghttpd`, plus `Connection: close` for HTTP/1.0)
6. Route response-policy wrapper (`route_response_policy_handler`) for path-scoped header overrides
7. Request logging wrapper (console or JSON)

Effectively, for active proxy routes the request enters:
`logging -> route response policy -> headers -> proxy -> local file pipeline`

## Core subsystems

### File and directory serving
- Primary file serving uses the Go standard library file server.
- Markdown mode renders `*.md` to HTML and can use `README.md` as a directory landing page.
- Browse mode always lists directories on trailing-slash paths and serves direct non-directory files (including `index.html`) without redirect loops.

### Reverse proxy
- Route mappings parse as `/from=http://backend` and are sorted by longest prefix for deterministic matching.
- Proxy handler forwards normal HTTP traffic through `httputil.ReverseProxy`.
- WebSocket upgrades are proxied via connection hijacking and bidirectional stream copy.
- HTTP proxy streaming behavior is selected per matched request path (`buffered` or `unbuffered` flush behavior).

### Route response policies
- Response header rules are resolved by path-prefix matching with deterministic specificity (more specific prefixes override broader ones).
- Policies are applied at response write time so route rules can enforce headers such as `Cache-Control` even when upstream handlers set their own values.

### TLS
- Manual TLS: provide `--tls-cert` and `--tls-key`.
- Dynamic HTTPS: `--https` provisions and installs a development CA/cert chain, serves HTTPS, then cleans up on exit.

### Logging
- `pkg/logging` wraps zap and supports `CONSOLE` and `JSON`.
- Console logging emits access-log style lines.
- JSON logging emits structured request start/completion entries and startup metadata.

### Integration and reliability
- Black-box integration tests under `tests/integration` validate process-level behavior across browse, HTTP/HTTPS, proxy, and WebSocket paths.
- CI includes end-to-end coverage aggregation gates for core runtime paths.

## Architectural changes reflected in this snapshot
- HTTPS lifecycle is consolidated into the main serve workflow (`--https`) rather than separate manual HTTPS command flows.
- Proxy configuration moved to explicit repeatable mappings with backwards-compatible legacy mapping support.
- Browse mode direct file handling was generalized to avoid index-only special cases.
- Route-scoped response policy and proxy streaming controls were added for SPA + API proxy deployments.
