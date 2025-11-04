# AGENTS.md

## TAuth

Single-origin authentication service for Goole Identy Service designed for stand-alone front-end apps. See README.md for details

## Document Roles

- NOTES.md: Read-only process/journal. Append-only when closing work; do not retroactively edit history.
- ISSUES.md: Append-only log of newly discovered requests and changes. No instructive sections live here; each entry records what changed or what was discovered.
- PLAN.md: Working plan for one concrete change/issue; ephemeral and replaced per change.

### Issue Status Terms

- Resolved: Completed and verified; no further action.
- Unresolved: Needs decision and/or implementation.
- Blocked: Requires an external dependency or policy decision.

### Validation & Confidence Policy

All rules for validation, error handling, invariants, and “confident programming” (no defensive checks, edge-only validation, smart constructors, CI gates) are defined in POLICY.md. Treat that document as binding; this file does not restate them.

## Front-End Coding Standards (Browser ES Modules with Alpine.js + Vanilla CSS)

### 1. Naming & Identifiers

- No single-letter or non-descriptive names.
- **camelCase** → variables & functions.
- **PascalCase** → Alpine factories / classes.
- **SCREAMING_SNAKE_CASE** → constants.
- Event handlers named by behavior (`handleSpinButtonClick`, not `onClick`).

### 2. State & Events

- **Local by default**: `x-data` owns its own state.
- **Shared state** only via `Alpine.store` when truly necessary.
- **Events for communication**: use `$dispatch` / `$listen` to link components.
- Prefer **DOM-scoped events** (bubbling inside a panel) over `.window`. Use scope IDs only if DOM hierarchy forces it.
- Notifications, modals, and similar components must be event-driven; they cannot show unless triggered by a defined event.

### 3. Dead Code & Duplication

- No unused variables, imports, or exports.
- No duplicate logic; extract helpers.
- One source of truth for constants or repeated transforms.

### 4. Strings & Enums

- All user-facing strings live in `constants.js`.
- Use `Object.freeze` or symbols for enums.
- Map keys must be constants, not arbitrary strings.

### 5. Code Style & Structure

- ES modules (`type="module"`), strict mode.
- Pure functions for transforms; Alpine factories (`function Foo() { return {…} }`) for stateful components.
- No mutation of imports; no parameter mutation.
- DOM logic in `ui/`; domain logic in `core/`; utilities in `utils/`.

### 6. Dependencies & Organization

- CDN-hosted dependencies only; no bundlers.
- Node tooling is permitted for **tests only**.
- Layout:

  ```
  /assets/{css,img,audio}  # optional, create when needed
  /data/*.json             # optional, create when needed
  /js/
    constants.js
    types.d.js
    utils/
    core/
    ui/
    app.js   # composition root
  index.html
  ```

- the MDE editor is used [text](MDE.v2.19.0.md). Follow the documentation to ensure proper API usage and avoid reimplementing the functionality available through MDE API
- marked.js documentation is available at [text](marked.js.md). Follow the documentation to ensure proper API usage and avoid reimplementing the functionality available through marked.js API

### Dependencies & Versions

- Alpine.js: `3.13.5` via `https://cdn.jsdelivr.net/npm/alpinejs@3.13.5/dist/module.esm.js`
- EasyMDE: `2.19.0`
- marked.js: `12.0.2`
- DOMPurify: `3.1.7`
- Google Identity Services: `https://accounts.google.com/gsi/client`
- Loopaware widget: `https://loopaware.mprlab.com/widget.js` (allowed per Security policy below)

### 7. Testing

- Puppeteer permitted; Playwright forbidden.
- Node test harness (`npm test`) runs browser automation.
- Use table-driven test cases.
- Black-box tests only: public APIs and DOM.
- `tests/assert.js` provides `assertEqual`, `assertDeepEqual`, `assertThrows`.

### 8. Documentation

- JSDoc required for public functions, Alpine factories.
- `// @ts-check` at file top.
- `types.d.js` holds typedefs (`Note`, `NoteClassification`, etc.).
- Each domain module has a `doc.md` or `README.md`.
- Before changing integrations with third-party libraries (EasyMDE, marked.js, DOMPurify, etc.), read the companion docs in-repo (`MDE.v2.19.0.md`, `marked.js.md`, …) to ensure we're using the supported APIs instead of re-implementing them.

### 9. Refactors

- Plan changes; write bullet plan in PR description.
- Split files >300–400 lines.
- `app.js` wires dependencies, registers Alpine components, stores, and event bridges.

### 10. Error Handling & Logging

- Throw `Error`, never raw strings.
- Catch errors at user entry points (button actions, init).
- `utils/logging.js` wraps logging; no stray `console.log`.

### 11. Performance & UX

- Use `.debounce` modifiers for inputs.
- Batch DOM writes with `requestAnimationFrame`.
- Lazy-init heavy components (on intersection or first interaction).
- Cache selectors and avoid forced reflows.
- Animations must be async; no blocking waits.

### 12. Linting & Formatting

- ESLint run manually (Dockerized).
- Prettier only on explicit trigger, never autosave.
- Core enforced rules:

  - `no-unused-vars`
  - `no-implicit-globals`
  - `no-var`
  - `prefer-const`
  - `eqeqeq`
  - `no-magic-numbers` (allow 0,1,-1,100,360).

### 13. Data > Logic

- Validate catalogs (JSON) at boot.
- Logic assumes valid data; fail fast on schema errors.

### 14. Security & Boundaries

- No `eval`, no inline `onclick`.
- CSP is optional and low priority for now; recommended for production hardening.
- Google Analytics snippet is the only sanctioned inline exception.
- All external calls go through `js/core/backendClient.js` and `js/core/classifier.js` (network boundaries), both mockable in tests. Do not call `fetch` directly from UI components.

## Backend (Go Language)

### Core Principles

- Reuse existing code first; extend or adapt before writing new code.
- Generalize existing implementations instead of duplicating them.
- Favor data structures (maps, registries, tables) over branching logic.
- Use composition, interfaces, and method sets (“object-oriented Go”).
- Depend on interfaces; return concrete types.
- Group behavior on receiver types with cohesive methods.
- Inject all external effects (I/O, network, time, randomness, OS).
- No hidden globals for behavior.
- Treat inputs as immutable; return new values instead of mutating.
- Separate pure logic from effectful layers.
- Keep units small and composable.
- Minimal public API surface.
- Provide only the best solution — no alternatives.

---

### Deliverables (for automation)

- Only changed files.
- No diffs, snippets, or examples.
- Must compile cleanly.
- Must pass `go fmt ./... && go vet ./... && go test ./...`.

---

### Code Style

- No single-letter identifiers.
- Long, descriptive names for all identifiers.
- No inline comments.
- Only GoDoc for modules and exported identifiers.
- No repeated inline string literals — lift to constants.
- Return `error`; wrap with `%w` or `errors.Join`.
- No panics in library code.
- Use zap for logging; no `fmt.Println`.
- Prefer channels and contexts over shared mutable state.
- Guard critical sections explicitly.

---

### Project Structure

- `cmd/` for CLI entrypoints.
- `internal/` for private packages.
- `pkg/` for reusable libraries.
- No package cycles.
- Respect existing layout and naming.

---

### Configuration & CLI

- Use Viper + Cobra.
- Flags optional when provided via config/env.
- Validate config in `PreRunE`.
- Read secrets from environment.

---

### Dependencies (Approved)

- Core: `spf13/viper`, `spf13/cobra`, `uber/zap`.
- HTTP: `gin-gonic/gin`, `gin-contrib/cors`.
- Data: `gorm.io/gorm`, `gorm.io/driver/postgres`, `jackc/pgx/v5`.
- Auth/Validation: `golang-jwt/jwt/v5`, `go-playground/validator/v10`.
- Testing: `stretchr/testify`.
- Optional: `joho/godotenv`, `prometheus/client_golang`, `robfig/cron/v3`.
- Prefer standard library whenever possible.

---

### Testing

- No filesystem pollution.
- Use `t.TempDir()` for temporary dirs.
- Dependency injection for I/O.
- Table-driven tests.
- Mock external boundaries via interfaces.
- Use real, integration tests with comprehensive coverage

---

### Web/UI

- Use Gin for routing.
- Middleware for CORS, auth, logging.
- No vanilla CSS; use either Bootstrap with Materia or Tailwind.
- Header fixed top; footer fixed bottom using CSS utilities.

---

### Performance & Reliability

- Measure before optimizing.
- Favor clarity first, optimize after.
- Use maps and indexes for hot paths.
- Always propagate `context.Context`.
- Backoff/retry as data-driven config.

---

### Security

- Secrets from env.
- Never log secrets or PII.
- Validate all inputs.
- Principle of least privilege.
- CSP-friendly ES modules. Allowed third-party scripts: Google Analytics snippet, Google Identity Services, Loopaware widget. When CSP is enabled, inline scripts must be limited to GA config or guarded by nonce/hash.

#### CSP Template (optional; use when enabling CSP)

- HTTP header (preferred):
  - `Content-Security-Policy: default-src 'self'; script-src 'self' https://cdn.jsdelivr.net https://accounts.google.com https://www.googletagmanager.com https://loopaware.mprlab.com 'nonce-<nonce-value>'; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data: blob:; connect-src 'self' https://llm-proxy.mprlab.com http://localhost:8080; font-src 'self' data:; frame-src https://accounts.google.com; base-uri 'self'; form-action 'self';`
- Meta tag (static hosting):
  - `<meta http-equiv="Content-Security-Policy" content="default-src 'self'; script-src 'self' https://cdn.jsdelivr.net https://accounts.google.com https://www.googletagmanager.com https://loopaware.mprlab.com 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data: blob:; connect-src 'self' https://llm-proxy.mprlab.com http://localhost:8080; font-src 'self' data:; frame-src https://accounts.google.com; base-uri 'self'; form-action 'self';">`
- Replace `connect-src` endpoints when running against different backends or proxies. Prefer nonces over `'unsafe-inline'` where a server can inject them.
- When using a local LLM proxy on a non-default port (e.g., `http://localhost:8081`), include it in `connect-src`.

### Assistant Workflow

- Read repo and scan existing code.
- Plan reuse and extension.
- Replace branching with data tables where appropriate.
- Implement minimal, cohesive types.
- Inject dependencies.
- Prove with table-driven tests.

---

### Review Checklist

- [ ] Reused/extended existing code.
- [ ] Replaced branching with data structures where appropriate.
- [ ] Minimal, cohesive public API.
- [ ] All side effects injected.
- [ ] No single-letter identifiers.
- [ ] Constants used for repeated strings.
- [ ] zap logging; contextual errors.
- [ ] Config via Viper; validated in `PreRunE`.
- [ ] Table-driven tests; no filesystem pollution.
- [ ] `go fmt`, `go vet`, `go test ./...` pass.

## Backend (Python)

### Core Principles

- Reuse existing modules first; extend or adapt before writing new code.
- Generalize existing implementations rather than duplicating logic.
- Favor **data-driven** solutions (maps, registries, configuration) over imperative branching.
- Encapsulate domain rules in **dataclasses** or dedicated classes with clear invariants.
- Keep functions small, pure, and composable; separate logic from I/O.
- Inject all external dependencies (files, network, randomness, time). No hidden globals.
- Treat inputs as immutable; always return new values instead of mutating.
- Minimal public API surface; expose only one clear solution.
- For validation, error handling, and invariants, follow **POLICY.md (Confident Programming)**.

---

### Code Style

- Descriptive identifiers only; no single-letter names.
- Use `@dataclass(frozen=True)` for immutable domain types.
- Validation happens in `__post_init__` or via Pydantic (if already in use).
- Raise `ValueError` subclasses for domain validation errors.
- Lift repeated string literals to constants.
- Module docstrings and class/function docstrings required; no inline comments.
- Use type hints everywhere; run `mypy --strict`.
- Logging through standard `logging` module; no stray `print`.

---

### Project Structure

- `app/` or `src/` as top-level application package.
- `domain/` for core business objects and invariants.
- `infrastructure/` for DB, network, and OS integration.
- `services/` for orchestration logic using domain + infra.
- `tests/` for unit and integration tests.

---

### Configuration & CLI

- Use `argparse` or `typer` for CLI.
- Read configuration from environment or `.env` files.
- Validate configuration up front (edge validation).

---

### Dependencies

- Prefer standard library; third-party libraries require explicit approval.
- Allowed: `dataclasses`, `typing`, `pydantic` (optional), `pytest`, `mypy`.

---

### Testing

- Use `pytest` with table-driven tests.
- Isolate side effects with fixtures.
- Use `tmp_path` for filesystem operations (no pollution).
- Black-box: test only public API contracts.
- CI gate: `pytest -q`, `mypy --strict domain service`.

---

### Review Checklist

- [ ] Reused/extended existing code.
- [ ] Domain objects created via smart constructors or dataclasses with invariants.
- [ ] No duplicated validation inside core.
- [ ] Constants used for repeated strings.
- [ ] Clear type hints, no single-letter identifiers.
- [ ] Config validated at startup.
- [ ] `pytest`, `mypy --strict` passing.
