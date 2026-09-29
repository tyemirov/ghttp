# AGENTS.GO.md

## Scope

This file gives backend rules for Go code. Obey root `AGENTS.md` and `.mprlab/POLICY.md` for shared workflow rules.

## Core Principles

- Reuse existing code first.
- Prefer data structures, registries, and cohesive types instead of branching logic.
- Inject external effects: I/O, network, time, randomness, and OS state.
- Keep core logic pure where practical.
- Accept domain types instead of loose primitives when invariants exist.
- Return errors and wrap them with context.
- Keep public API surface minimal.

## Code Style

- Use descriptive identifiers. No single-letter names except conventional tiny scopes where the repo already allows them.
- Lift repeated string literals into constants.
- Use GoDoc for exported identifiers.
- No panics in library code.
- Use structured logging when the repo has a logger.
- Propagate `context.Context` through effectful boundaries.

## Testing

- For a behavior change, start with an integration test through the real HTTP, CLI, or public package entry point.
- Use dependency injection for integration scenarios that are difficult to reproduce.
- Keep the product logic under test real.
- Prefer black-box integration tests through HTTP, CLI, or public package entry points.
- Use table-driven scenarios where they cover contract permutations.
- Use `t.TempDir()` for temporary filesystem work.
- Use focused unit tests for complex algorithms, calculations, and isolated logic when useful.
- Require integration coverage of public behavior for product acceptance.

## Validation

Use `.mprlab/POLICY.md` for validation.

During the change, run the smallest Go target that validates the changed contract.

When these tools are part of the repository contract, `make lint` must include `go vet`, `staticcheck`, and `ineffassign`.
