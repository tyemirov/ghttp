# AGENTS.PY.md

## Scope

This file gives backend rules for Python code. Obey root `AGENTS.md` and `.mprlab/POLICY.md` for shared workflow rules.

## Core Principles

- Reuse existing modules first.
- Prefer data-driven registries and explicit domain types instead of branching.
- Use `@dataclass(frozen=True)` or Pydantic when already in use for validated domain values.
- Keep logic small, typed, and testable through public entry points.
- Inject files, network, randomness, time, and environment access.
- Validate at CLI, HTTP, file, and adapter edges.

## Code Style

- Use type hints.
- Use descriptive identifiers.
- Lift repeated literals into constants.
- Use module, class, and function docstrings where they clarify public behavior.
- Use `logging`. Do not leave stray `print` calls in libraries.
- Raise explicit exceptions for domain validation failures.

## Testing

- For a behavior change, start with an integration test through the real HTTP, CLI, or public package entry point.
- Use dependency injection for integration scenarios that are difficult to reproduce.
- Keep the product logic under test real.
- Use pytest.
- Exercise public contracts through CLI, HTTP, or public package entry points.
- Use fixtures and `tmp_path` to isolate side effects.
- Use focused unit tests for complex algorithms, calculations, and isolated logic when useful.
- Require integration coverage of public behavior for product acceptance.

## Validation

Use `.mprlab/POLICY.md` for validation.

During the change, run the smallest Python target that validates the changed contract.

When the repository contract includes it, lint must run mypy or pyright for typed Python surfaces.
