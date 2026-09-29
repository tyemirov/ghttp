# AGENTS.md

## Forward-Only Contract Discipline

This repository follows a forward-only, confident programming paradigm. This is a binding agent contract: no fallbacks, no backward compatibility, no legacy support, and no compatibility shims. Do not spend design or implementation effort on backward compatibility considerations except for explicit one-off data migrations into the current canonical contract.

Repeat for emphasis because this rule is binding: no fallbacks, no backward compatibility, no legacy compatibility. Delete or reject obsolete code paths, stale schemas, deprecated config, and old persisted shapes instead of preserving them through compatibility layers, dual reads/writes, aliases, or best-effort recovery.

One-off data migrations are allowed only when they move existing persisted data into the current schema in a bounded operation. After migration, remove the bridge and keep only the current contract.

## TAuth

Single-origin authentication service for Goole Identy Service designed for stand-alone front-end apps. See README.md for details

## Document Roles

- `NOTES.md` contains read-only process history. Use `.mprlab/POLICY.md` and `.mprlab/PLANNING.md` for current execution rules.
- Use root `ISSUES.md` as the active issue tracker. Preserve issue history.
- Use `.mprlab/PLANNING.md` for temporary execution plans.

### Issue Status Terms

- Resolved: Completed and verified; no further action.
- Unresolved: Needs decision and/or implementation.
- Blocked: Requires an external dependency or policy decision.

### Validation & Confidence Policy

Use `.mprlab/POLICY.md` for validation, error handling, invariants, and confident programming.

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
- Use public API and browser integration tests for product acceptance.
- Use focused unit tests under `.mprlab/POLICY.md` when useful.
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
- Obey the integration-first sequence in `.mprlab/POLICY.md` before implementation.
- Inject dependencies for difficult integration scenarios.
- Use table-driven scenarios where applicable.

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
- For validation, error handling, and invariants, follow **.mprlab/POLICY.md (Confident Programming)**.

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
- Use public API integration tests for product acceptance.
- Use focused unit tests under `.mprlab/POLICY.md` when useful.
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

<!-- BEGIN MPRLAB-GOVERNANCE -->
## MPR Lab Governance

Root `AGENTS.md` is the agent entrypoint. Shared rules live under `.mprlab/`.

Read `.mprlab/POLICY.md` for every task.
Read the following files only when their condition applies.
Read each selected guide in full before its first applicable action.

- Before edits: `.mprlab/PLANNING.md`.
- For technical prose: `.mprlab/AGENTS.DOCS.md` and `.mprlab/TERMINOLOGY.md`.
- For issue work: the selected issue and its dependencies in `ISSUES.md`.
- For tracker edits: `.mprlab/issues-md-format.md`.
- For Git operations: `.mprlab/AGENTS.GIT.md`.
- For HTTP or gRPC API changes: `.mprlab/AGENTS.API.md`.
- For Go changes: `.mprlab/AGENTS.GO.md`.
- For Python changes: `.mprlab/AGENTS.PY.md`.
- For browser changes: `.mprlab/AGENTS.FRONTEND.md`.
- For container changes: `.mprlab/AGENTS.DOCKER.md`.

File permission modes are outside agent scope.
Never examine, validate, compare, require, change, or record a file permission mode.
Never use a file permission mode in acceptance, security, credential, execution, publication, deployment, or failure analysis.
The values `0600` and `7777` have no governance meaning.
This rule does not change service authorization or operation authority.

Always reference each issue by its ID, for example `B001` or `I027`.
Never use an `ISSUES.md` file path, line number, or `path:line` syntax as an issue reference.

Do not create `.mprlab/AGENTS.md`. Scoped guidance belongs in `.mprlab/AGENTS.*.md` files.
If guidance conflicts, obey `.mprlab/POLICY.md` first, then root `AGENTS.md`, then the applicable scoped guide.
<!-- END MPRLAB-GOVERNANCE -->
