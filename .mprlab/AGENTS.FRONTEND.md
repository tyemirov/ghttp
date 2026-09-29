# AGENTS.FRONTEND.md

## Scope

This file gives rules for browser frontends. Obey root `AGENTS.md` and `.mprlab/POLICY.md` for shared workflow and validation rules.

## Principles

- Build semantic markup that mirrors the domain.
- Components render validated state and emit user intent.
- Keep transport, persistence, and backend payload validation in explicit adapter modules.
- Keep route strings, endpoint paths, storage keys, event names, and workflow codes in constants or backend payloads.
- Do not use anonymous wrapper-heavy markup when semantic elements or custom elements are applicable.

## JavaScript

- Use ES modules.
- Put `// @ts-check` at the top of new or edited checked JavaScript modules.
- Use JSDoc typedefs for domain objects, component props, and backend payloads.
- Do not mutate imported bindings or function parameters.
- No implicit globals.
- No stray `console.log`.

## UI State

- Keep one source of truth for workflow state.
- Derive display values instead of duplicating derived state.
- Dispatch intent-specific events.
- Clean up timers, subscriptions, object URLs, observers, and pending async work.
- Do not catch and ignore invariant violations.

## Production Hosting

- Use GitHub Pages as the production host for each deployable browser frontend.
- Declare the browser frontend with a `github_pages` resource in `.mprlab/deploy/resources.yml`.
- Use `gh-pages` as the publication branch.
- The GitHub Pages repository can differ from the application repository.
- Keep API and service routes on hostnames that differ from the website hostname.
- Reserve the GitHub Pages domain and its `www` hostname for GitHub Pages.
- Verify publication through the public website and `/.mprlab-release.json`.

## Testing

- For a behavior change, start with an integration test through the real page and browser entry point.
- Use dependency injection for integration scenarios that are difficult to reproduce.
- Keep the product logic under test real.
- Prefer Playwright or the repo-standard browser harness.
- Cover user-visible behavior through the real page and browser entry point.
- Assert rendered state, accessibility-relevant behavior, emitted events, requests, and downloaded artifacts.
- Do not use unit tests as the only proof for visible behavior.

## Validation

Use `.mprlab/POLICY.md` for validation.

During the change, run the smallest frontend target that validates the changed contract.

Run build or browser tests when source changes affect generated or shipped assets.
