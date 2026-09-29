# AGENTS.API.md

## Scope

This file gives design and implementation rules for repository-owned HTTP and gRPC interfaces.

Obey root `AGENTS.md`, `.mprlab/POLICY.md`, and the applicable language guide.

This guide applies when the repository declares or implements an API contract. An outbound HTTP client alone does not make the repository the owner of that API's design.

## Protocol Contract

- Design HTTP APIs as resource-oriented REST APIs. Use standard HTTP methods, status codes, headers, and semantics.
- Design gRPC APIs as protobuf services with typed RPC methods. REST constraints do not apply to gRPC.
- Do not expose action-shaped HTTP RPC endpoints when the operation can be modeled as a resource or state transition.
- Do not add parallel REST, gRPC, JSON gateway, or compatibility interfaces unless each interface is an explicit current product requirement.

## REST Constraints

- Keep the client and server concerns separate. Clients consume representations and do not depend on handler, database, or deployment internals.
- Keep requests stateless. Every request carries the authentication, resource identity, preconditions, and other context necessary to process it.
- Make cacheability explicit with `Cache-Control`, validators such as `ETag` or `Last-Modified`, and `Vary` where representations depend on request headers.
- Maintain one uniform resource interface across the API instead of endpoint-specific transport conventions.
- Support gateways, proxies, and other layers without making clients depend on the server's internal hop topology.
- Represent related resources and available state transitions with links when runtime discoverability is part of the public contract.

## REST Resources And URLs

- Model domain resources, not controller actions or UI workflows.
- Use nouns for resources, plural collection paths, and stable opaque identifiers.
- Use lowercase, hyphenated path segments consistently.
- Nest paths only for true ownership or containment, and keep nesting shallow.
- Use query parameters for filtering, sorting, field selection, and pagination instead of inventing route variants.
- Model long-running work as an operation resource. For example, create an export with `POST /exports`, return its resource, and read its status with `GET /exports/{export_id}`.
- Do not create verb routes such as `/get-user`, `/createReport`, `/orders/{id}/cancel`, or `:run` endpoints.

## HTTP Methods

- `GET` reads a resource or collection and never mutates server state.
- `POST` creates a subordinate resource or operation. Use an idempotency key for retry-sensitive creation.
- `PUT` completely replaces the resource at a client-selected URI and is idempotent.
- `PATCH` performs a documented partial update with an explicit patch schema.
- `DELETE` makes the target resource absent and remains idempotent across retries.
- `HEAD` returns the same metadata as `GET` without a response body. `OPTIONS` advertises supported interaction semantics when the interface exposes it.
- Preserve the safety and idempotency semantics of standard methods. Return `405 Method Not Allowed` with `Allow` when a resource exists but the method is unsupported.

## Status Codes And Representations

- Return `200 OK` for a successful read or update that includes a representation.
- Return `201 Created` with `Location` for synchronous creation.
- Return `202 Accepted` for queued work.
- Return `204 No Content` when a successful response has no body.
- Return `400` for a malformed request and `401` for missing or invalid authentication.
- Return `403` for denied authorization and `404` for an absent resource.
- Return `409` for a state conflict and `412` for a failed precondition.
- Return `422` for semantically invalid input.
- Do not return success failures inside `2xx` envelopes or successful results with `4xx` or `5xx` status codes.
- Use one documented media type and field-naming convention per interface. Send the correct `Content-Type` and reject unsupported request media types.
- Honor `Accept` when the API offers multiple representations.
- Keep identifiers opaque and serialize timestamps in UTC with RFC 3339.
- Return errors in one typed shape. Include a stable code, message, optional details, and request or trace identifier.
- Do not expose stack traces or internal implementation details.

## Collections, Retries, And Concurrency

- Use one consistent filtering and sorting vocabulary across collections.
- Use cursor pagination for large or mutable collections. Document the order and cursor behavior.
- Bound page sizes and return the next cursor in a consistent response field or header.
- Use idempotency keys for non-idempotent requests that callers must safely retry.
- Use entity tags, version fields, or explicit preconditions when concurrent updates can overwrite current state.
- Honor cancellation and propagate request deadlines through downstream work.

## Contract Ownership

- Keep one canonical current contract. Prefer a machine-readable schema such as OpenAPI.
- When code owns routes, generate or validate the schema from the same definitions.
- Centralize route paths, operation identifiers, schemas, and error codes. Do not duplicate them across handlers, clients, tests, and documentation.
- Validate path, query, header, and body inputs once at the transport boundary before constructing domain values.
- Authenticate at the transport boundary and authorize every operation against the addressed resource. Do not put credentials or secrets in URLs, logs, or error responses.
- Update the server, schema, repository-owned clients, documentation, and contract tests atomically when the API changes.
- Delete obsolete routes, payload shapes, versions, aliases, dual reads, and translation layers. Do not retain deprecated API surfaces as compatibility shims.

## gRPC

- Treat `.proto` files as the canonical service contract and generate transport types from them.
- Use cohesive services, explicit RPC names, and typed request and response messages.
- Use standard gRPC status codes and structured error details.
- Propagate deadlines and cancellation. Use streaming only when the interaction requires streaming semantics.
- Remove obsolete services and RPCs instead of retaining deprecated aliases. Reserve removed protobuf field numbers and names so they cannot be reused unsafely.
- Do not add REST-shaped paths or an HTTP/JSON mirror to a gRPC-only interface.

## Testing And Validation

- For a behavior change, start with an integration test through the real HTTP listener or public gRPC service entry point.
- Use dependency injection for integration scenarios that are difficult to reproduce.
- Keep the product logic under test real.
- Exercise REST behavior through a real HTTP listener and gRPC behavior through the real public service entrypoint.
- Assert methods or RPCs, resource identifiers, status codes, headers or metadata, response bodies, typed errors, and externally visible state changes.
- Cover method safety, idempotent retries, validation failures, authorization boundaries, pagination, conflicts, and cancellation where those behaviors apply.
- Validate or lint the canonical API contract and regenerate derived artifacts with the repository-owned command.
- Use `.mprlab/POLICY.md` for validation.
- During the change, run the smallest API target that validates the changed contract.
