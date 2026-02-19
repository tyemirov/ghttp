# Changelog

## v0.4.3 — 2026-02-18

### Features ✨
- _No changes._

### Improvements ⚙️
- Clarified and improved folder browsing behavior for the `--browse` flag: folder URLs now always return directory listings, even if index.html or README.md exist. Direct file requests use the standard file-serving pipeline without filename preference; Markdown files still render when Markdown rendering is enabled.
- Updated documentation to reflect the new `--browse` flag behavior.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- Added comprehensive tests covering browse handler behavior, including index.html handling and root index.html direct requests in browse mode.
- Improved test coverage for browse mode directory listing and direct file serving.
- Added integration tests verifying browse mode behavior with various index.html file scenarios.

### Docs 📚
- Updated README and docker-compose-ai-agents.md docs to accurately describe the `--browse` flag behavior changes.

## v0.4.2 — 2026-02-18 

### Features ✨
- _No changes._

### Improvements ⚙️
- Skip Docker image builds for documentation-only changes to optimize CI workflows.
- Add GitHub Pages workflow to automate documentation site deployment.
- Generalize browse mode file serving to serve direct non-directory files uniformly.

### Bug Fixes 🐛
- Fix browse mode to serve direct index files without triggering canonical redirects.
- Correct browse mode to serve direct file requests from directory listing links, preventing redirect loops.

### Testing 🧪
- Add integration tests to validate browse mode serving of direct index and HTML files.

### Docs 📚
- Add comprehensive GitHub Pages landing site with enhanced styling and quickstart content.
- Include new 404 error page with Google Analytics tracking.
- Add CNAME file for custom domain.
- Update documentation to show quickstart card content and Google Analytics integration.

## v0.4.1 — 2026-01-26

### Features ✨
- Introduce proxy mapping normalization for `GHTTP_SERVE_PROXIES`.
- Move proxy configuration validation to the edge.

### Improvements ⚙️
- Update Docker Compose documentation accuracy for image references, configuration precedence, and environment variables.

### Bug Fixes 🐛
- Normalize `GHTTP_SERVE_PROXIES` to correctly parse comma-delimited proxy mappings.

### Testing 🧪
- Add test to verify splitting of comma-delimited proxy mappings in proxy configuration.

### Docs 📚
- Enhance documentation on proxy configuration and Docker Compose integration settings.
- Add `ISSUES.md` entries for proxy mapping normalization and documentation updates.

## v0.4.0 — 2026-01-26

### Features ✨
- Introduce repeatable proxy mappings with explicit from/to semantics via `--proxy` flag and `GHTTP_SERVE_PROXIES` environment variable.
- Add environment variable support for specifying configuration file path (`GHTTP_CONFIG_FILE`).
- Default HTTPS port selection integrated into serve command; manual HTTPS CLI workflow removed.

### Improvements ⚙️
- Align default port documentation and move validation logic to the edge.
- Update Dockerfile to expose port 8000 to match default HTTP port.
- Clarify bind address logging in documentation.
- Refine HTTPS flags and add coverage tests.
- Enhance proxy routing and add coverage for proxy mappings.
- Update README examples to reflect default port 8000 for HTTP and 8443 for HTTPS.
- Add detailed flags and environment variables documentation.

### Bug Fixes 🐛
- Remove manual HTTPS CLI subcommands, consolidating HTTPS management under a single flag.
- Fix legacy proxy flags to coexist properly with new repeatable proxy mappings.

### Testing 🧪
- Add extensive tests for proxy routes, proxy handler, serve command, and HTTPS flag behaviors.

### Docs 📚
- Clarify environment variable mappings for CLI flags in README.
- Update Docker Compose examples and documentation to default port 8000.
- Expand ISSUES.md with planning entries for repeatable proxies and HTTPS CLI removal.
- Document proxy configuration and behavior improvements.

## v0.3.2 — 2026-01-12

### Features ✨
- Added full handler chain method for production-equivalent integration tests.
- Replaced all unit tests for proxy with comprehensive integration tests.

### Improvements ⚙️
- Implemented http.Hijacker on statusRecorder to support WebSocket connections through logging middleware.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- Removed proxy unit tests; introduced extensive proxy integration tests covering:
  - Backend forwarding and fallback.
  - Query parameter and request body preservation.
  - Handling backend errors and invalid URLs.
  - WebSocket upgrade handling through full handler stack and logging middleware.

### Docs 📚
- _No changes._

## v0.3.1 — 2026-01-12

### Features ✨
- _No changes._

### Improvements ⚙️
- Validate proxy backend URLs to fail fast on malformed URLs, ensuring only http or https schemes with valid hosts.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- Add tests covering rejection of invalid proxy backend URLs including invalid schemes and missing hosts.

### Docs 📚
- _No changes._

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
