# Confident Programming (Binding Policy for Agents)

## 0) One-page summary (operator rules)

- Validate **only at edges** (I/O, HTTP, CLI, DB adapters). Core assumes valid domain objects.
- **Make illegal states unrepresentable** via types and smart constructors.
- **Fail fast** (dev) and **wrap errors with context** at boundaries (prod).
- **Narrow interfaces**: accept domain types, not loose primitives, when a domain type exists.
- **No duplicate checks** in core; once validated, don’t re-validate.
- Tests target **contracts/invariants**, not defensive branches.
- Reference domain model and executor contracts: [docs/cli_design.md §8](docs/cli_design.md#8-repository-domain-model-and-executor-contracts-gx-403--gx-406).

---

## A. Hard rules (agent MUST follow)

1. Edge-only validation; core **assumes** valid domain objects.
2. All domain entities created via **smart constructors**; exporting “zero-but-invalid” is **forbidden**.
3. **No duplicated validation** inside core modules—remove it when found.
4. **Narrow interfaces**: functions accept domain types (not `string`, `any`, bare `float64`, etc., when a domain type exists).
5. Errors are **explicit and contextual**. Choices:

   - **Dev**: assert/panic for impossible states.
   - **Prod boundary**: **wrap** with operation + subject + stable code.

6. **No silent fallbacks** or “best-effort” paths unless a product requirement is cited in the commit.

---

## B. Invariants & contracts (declare per module)

- **Preconditions**: truth required at entry.
- **Postconditions**: guarantees on success.
- **Invariants**: properties that can never be false for the type/module.
- Each constructor MUST reject invalid state with a **typed/explicit** error.

---

## C. Language targets (agent MUST implement)

### Go

- Smart constructors returning `(Type, error)`; **no exported invalid zero-values**.
- Typed/sentinel errors; wrap using `fmt.Errorf("%w: context", ErrX)`.
- CI gates MUST pass: `go vet ./... && staticcheck ./... && ineffassign ./...`.

### Python

- `@dataclass(frozen=True)` or **Pydantic if already present**.
- Validate in `__post_init__` (or Pydantic validators); raise `ValueError` subclasses.
- CI gate MUST pass: `mypy --strict domain service` (or `pyright` equivalent).

### JavaScript (vanilla ESM + JSDoc; no build step)

- `// @ts-check` at top of every new/edited file.
- JSDoc typedefs; factory functions **throw** on invalid input.
- CI gate MUST pass: `tsc --noEmit` (type-checking only).

---

## D. Prohibited patterns (auto-reject)

- Exporting partially initialized/invalid structs/classes.
- Swallowing errors (`catch {}` with no action; `if err == nil { /* ignore */ }`).
- Re-validating a domain object already built by a smart constructor.
- Adding “best-effort” fallback without a cited product requirement.
- Boolean/flag parameters that conflate behaviors when a sum-type or distinct API is clearer.

---

## E. Agent patching protocol (order of operations)

1. **Introduce or reuse** a domain type before changing service logic.
2. **Move validation** from core into the nearest edge (handler/CLI/repo adapter).
3. Replace ambiguous flags with **sum-type** style (Go: newtype + constants; Python/JS: literal unions).
4. **Wrap errors** with operation + subject + stable code; do not couple core to HTTP/log formatting.
5. **Delete redundant checks**; note removal in commit message: “removed interior validation (edge-validated)”.

---

## F. CI gates (must wire or respect)

- **Go:** `go vet ./... && staticcheck ./... && ineffassign ./...`
- **Python:** `mypy --strict domain service` (or `pyright`)
- **JS:** `tsc --noEmit` with `// @ts-check` present in edited files

> Failing any gate = patch is not acceptable.

---

## G. PR template checks (agent MUST tick)

- [ ] Validation moved to edges; core free of re-checks
- [ ] Domain types created/extended via smart constructors
- [ ] Errors wrapped with operation + subject + stable code
- [ ] No zero-but-invalid exports
- [ ] Language CI gates pass (Go/Py/JS as applicable)

---

## H. Agent self-check rubric (0/1 each; **must score 6/6**)

1. All external inputs validated **exactly once** at edges.
2. All core function params are **domain types**, not loose primitives.
3. No defensive re-checks remain inside core.
4. Every constructor rejects invalid state with a typed/explicit error.
5. Every error path includes **operation + subject + stable code**.
6. CI gates configured or confirmed passing for edited languages.

If score < 6, agent MUST continue patching until score = 6.

---

## I. Minimal boilerplate the agent MAY reuse

### Go

```go
package domain

import (
	"errors"
	"fmt"
)

var ErrInvalidExample = errors.New("invalid_example")

type ExampleID string

func NewExampleID(rawInput string) (ExampleID, error) {
	if rawInput == "" {
		return "", fmt.Errorf("%w: empty id", ErrInvalidExample)
	}
	return ExampleID(rawInput), nil
}
```

### Python

```python
from dataclasses import dataclass

class InvalidExample(ValueError):
    pass

@dataclass(frozen=True)
class ExampleId:
    value: str
    def __post_init__(self) -> None:
        if not self.value:
            raise InvalidExample("empty id")
```

### JavaScript (ESM + JSDoc)

/_ @ts-check _/

```js
/**
 * @typedef {{ value: string }} ExampleId
 */

/**
 * @param {string} rawInput
 * @returns {ExampleId}
 */
export function createExampleId(rawInput) {
  const normalized = String(rawInput).trim();
  if (!normalized) throw new Error("invalid_example: empty id");
  return { value: normalized };
}
```

---

## J. Commit message template (agent MUST use)

```
feat(domain): introduce {DomainType} smart constructor; move validation to edge

- add {DomainType} with invariants: {list}
- adapt handlers/adapters to construct at edges
- remove interior defensive checks (validated once at boundary)
- wrap errors with operation+subject+stable code (e.g., user.create.invalid_email)
- CI: {go vet/staticcheck | ineffassign | mypy/pyright | tsc --noEmit} passing
```

---

## K. Example edge→core flow (reference)

1. **Edge** (HTTP/CLI/Repo): parse → validate → construct domain types.
2. **Core** (services/domain): operate only on domain types; no re-validation.
3. **Boundary**: map domain/infra errors to stable codes and user-facing messages.

---

### Notes for agents

- Prefer **static guarantees** over runtime checks.
- If you must catch and continue, include a **product requirement citation** and justification.
- Do not introduce new dependencies unless an existing project standard already includes them.

---

### CI snippets

**Go (Makefile)**

```
.PHONY: verify
verify:
	go vet ./...
	staticcheck ./...
	ineffassign ./...
```

**Python (Makefile)**

```
.PHONY: verify
verify:
	mypy --strict domain service
```

**JS (package.json)**

```json
{
  "scripts": {
    "verify": "tsc --noEmit"
  }
}
```
