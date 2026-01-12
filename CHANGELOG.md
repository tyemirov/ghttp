# Changelog

## v0.3.0 — 2026-01-12

### Features ✨
- Add reverse proxy capability with full WebSocket support, enabling proxying API requests alongside static file serving.
- Support configuration of reverse proxy via environment variables and CLI flags.

### Improvements ⚙️
- Reformat Go files for consistency.
- Refactor module namespace following owner rename.
- Update dependencies including cobra, goldmark, zap, and gorilla/websocket.
- Enhance Docker Compose integration documentation for AI agents.
- Update README to reflect module rename and updated image paths.
- Improve logging options with JSON format support.
- Add environment variable configurations for TLS certificates and HTTPS.

### Bug Fixes 🐛
- Fix old import paths preventing compilation.
- Use TLS protocol for WebSocket connections to HTTPS backends.

### Testing 🧪
- Add extensive tests for proxy handler including WebSocket support.
- Minor test updates across root and serve command tests, and cert installer tests.

### Docs 📚
- Add detailed Docker Compose integration guide tailored for AI agents, including environment variable usage, HTTPS handling, and configuration patterns.
- Update README with new release URLs and usage instructions.
- Add issues documentation for new reverse proxy feature.

## v0.2.4 — 2025-10-10

### Features ✨
- Add integration test for static asset MIME types
- Add multi-platform Docker image build support
- Add comprehensive documentation for Docker workflows

### Improvements ⚙️
- Consolidate Docker publishing workflow
- Publish multi-platform Docker images
- Streamline Docker distribution targets
- Hardcode Docker base images for stability
- Update instructions for autonomous coding flow
- Update README and documentation for autonomous flow

### Bug Fixes 🐛
- Stabilize Docker integration tests
- Correct branch reference in Docker workflows
- Skip Docker integration tests when Docker is missing
- Ignore service files in distribution

### Testing 🧪
- Add extensive integration tests for Docker and distribution
- Use temporary directories and table-driven tests for integration
- Mock external dependencies and focus on black-box API testing

### Docs 📚
- Update README.md with usage and project info
- Add comprehensive Docker publishing and CI documentation
- Expand AGENTS.md with coding standards and policies
- Improve autonomous flow documentation

## v0.2.3 — 2025-10-10

### Fixed
- `--browse` flag now forces directory listings even when default index files are present.

## v0.2.2 — 2025-10-10

### Added
- `--browse` flag to force directory listings while still rendering markdown and HTML files when explicitly requested.
- Positional argument support for serving a specific HTML or Markdown file directly (for example, `ghttp cat.html`).

### Fixed
- `--no-md` flag now serves Markdown files without HTML conversion and honors `index.html` before `README.md` when both exist.
- `--browse` flag now forces directory listings even when default index files are present.

## v0.2.1 — 2025-10-10

### Added
- Scaffolding for GitHub releases using GitHub actions
- CI pipeline for GitHub
- Makefile to abstract the CI logic from the commands

### Changed
- Tests refactored into unit tests and integration tests

## v0.2.0 — 2025-10-09

### Added
- Published a reusable `pkg/logging` service with console and JSON encoders, typed field helpers, and dedicated tests so other binaries can share gHTTP's logging stack.

### Changed
- Rewired the CLI, HTTPS workflow, and file server to emit all events through the centralized logging service, keeping request and lifecycle logs consistent across JSON and console modes.
- Moved the logging implementation into `pkg/logging` to make the abstraction importable by external consumers without reaching into `internal/`.
- Adjusted HTTPS certificate provisioning to install CA material into user-level trust stores on macOS, Linux, and Windows, removing the need for sudo escalation during install or uninstall.

### Fixed
- Eliminated repeated password prompts during certificate setup by targeting user-owned keychains/anchors and cleaning them up without elevated privileges.

## v0.1.2 — 2025-10-08

### Fixed
- Corrected the Go module path to `github.com/temirov/ghttp`, aligning imports across the project.

## v0.1.1 — 2025-10-07

### Added
- Published contributor operating guidelines in `AGENTS.md` covering coding standards, testing policy, and delivery requirements.

### Changed
- Normalized the server listening address reported in logs to favor `localhost` for wildcard and loopback binds, backed by a dedicated formatter and table-driven tests.
- Expanded README guidance with installation prerequisites, usage scenarios, and refreshed licensing details.

## v0.1.0 — 2025-08-19

### Added
- Introduced the `ghttpd` CLI as a minimal file server compatible with `python -m http.server` flags for port, bind address, directory, and protocol selection.
- Enabled optional TLS via `--tls-cert` and `--tls-key`, enforcing presence checks for both files before starting the server.
- Implemented structured request logging with latency reporting and graceful shutdown handling for `SIGINT` and `SIGTERM`.
- Added the `GHTTPD_DISABLE_DIR_INDEX` environment toggle to block directory listings while still serving files.
- Bootstrapped the project scaffolding with licensing, documentation, and ignore rules.
