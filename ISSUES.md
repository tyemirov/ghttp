# ISSUES (Open Only)

- [ ] [G001] Add response-cache and stream-delivery controls for app shells and SSE.
  Summary:
  ghttp needs first-class runtime controls for frontend cache policy and long-lived event streams so apps can avoid stale JS/CSS shells while preserving efficient caching for static assets and stable SSE delivery.
  Requested controls:
  - Per-route Cache-Control configuration (`no-store`, `no-cache`, and immutable/static profiles).
  - ETag and conditional request controls (`If-None-Match` handling toggles).
  - SSE-friendly streaming mode (disable buffering/compression where needed, immediate flush semantics).
  - Simple config surface (CLI flags and/or config file) so container entrypoints do not need custom reverse-proxy templates for these basics.

## Maintenance

### Recurring

- [ ] [M400R] (P2) Backlog hygiene and archive
  Goal:
  Keep the issue tracker reliable, readable, and focused on active work while preserving resolved history in the appropriate archive.

  Requirements:
  - Cadence: run weekly during active development and before each release cut.
  - Validate section names, identifier prefixes, recurrence suffixes, priority markers, dependencies, and duplicate IDs against the current `issues-md-format.md`.
  - Reconcile stale statuses, duplicate issues, broken references, obsolete instructions, and entries filed under the wrong section.
  - Move completed non-recurring history to the repository issue archive or durable documentation when the active tracker becomes noisy.
  - Keep active, blocked, planning, and recurring entries visible in `ISSUES.md`.

  Deliverables:
  - Normalized `ISSUES.md` structure and statuses.
  - Updated issue archive or docs when completed entries are removed from the active tracker.
  - A short `Last run:` note summarizing the cleanup and any follow-up issues filed.

  Validation:
  - Re-read `ISSUES.md` after edits and confirm every issue is under the right section with a unique section-aware ID.
  - Confirm recurring entries remain open and keep the `R` suffix.
  - Confirm no active, blocked, recurring, or planning work was archived.

- [ ] [M401R] (P2) Polish open issues
  Goal:
  Keep unresolved work executable by making each open issue concrete, ordered, and testable.

  Requirements:
  - Cadence: run weekly during active development and before handing a repo to automated execution.
  - Review every unresolved non-recurring issue for missing context, dependencies, repro steps, acceptance criteria, and validation expectations.
  - Make priorities concrete and ensure each open issue has actionable deliverables.
  - Merge duplicate open issues or add explicit dependency links when separate entries must remain.
  - Do not close or implement issues as part of this polish pass unless that work is separately requested.

  Deliverables:
  - Open issues with enough detail for a person or agent to execute without rediscovery.
  - New or updated dependency markers where ordering matters.
  - A short `Last run:` note listing the number of issues polished and any blockers found.

  Validation:
  - Sample the open entries after the pass and confirm each has clear next actions and validation expectations.
  - Confirm no recurring runbook was marked complete.
  - Confirm duplicates were merged or explicitly cross-referenced.

- [ ] [M402R] (P2) Architecture and policy review
  Goal:
  Catch architecture, policy, and workflow drift before it becomes hidden maintenance debt.

  Requirements:
  - Cadence: run monthly, before large refactors, and after major framework or runtime changes.
  - Review the codebase, docs, and workflow against `AGENTS.md`, `POLICY.md`, stack guides, and the current architecture notes.
  - Look for drift from forward-only contracts, edge-validation boundaries, smart-constructor usage, testing policy, and module ownership.
  - Record findings as new Maintenance issues with concrete scope, priority, and validation.
  - Close the pass with a no-action note only when the review finds no actionable drift.

  Deliverables:
  - New Maintenance issues for each actionable architecture or policy drift finding.
  - Updated notes on areas reviewed and areas intentionally left unchanged.
  - A short `Last run:` note with the review scope and outcome.

  Validation:
  - Confirm every finding is represented as an issue with owner-readable context and validation criteria.
  - Confirm no implementation changes were mixed into the review runbook unless separately requested.
  - Confirm all recurring runbooks remain open.

- [ ] [M403R] (P1) Dependency and security audit
  Goal:
  Keep third-party dependencies, runtime versions, and security-sensitive configuration within the current supported contract.

  Requirements:
  - Cadence: run weekly for active apps and before each release cut.
  - Inspect package managers, lockfiles, language toolchains, container bases, and generated clients for known vulnerabilities or stale direct dependencies.
  - Review auth, secret, CORS, CSP, SQL, network, and permission-sensitive configuration for drift from the current contract.
  - Prefer current supported dependencies; do not add compatibility shims for obsolete dependency behavior.
  - File separate Maintenance or BugFix issues for each actionable vulnerability, unsupported runtime, or security-contract gap.

  Deliverables:
  - Documented audit commands or data sources used for the pass.
  - Updated issues for each actionable dependency or security finding.
  - A short `Last run:` note with clean result or follow-up issue IDs.

  Validation:
  - Rerun the repository-native audit, lint, or dependency checks used for the pass.
  - Confirm every finding is either filed, fixed under a separate issue, or explicitly marked not applicable with evidence.
  - Confirm no secrets or private payloads were written into the tracker.

- [ ] [M404R] (P1) CI, release, and artifact health
  Goal:
  Keep the repository's validation, release, publication, and generated artifact surfaces trustworthy.

  Requirements:
  - Cadence: run before every release, publish, or deploy, and weekly for critical services.
  - Verify repository-native CI, lint, format, coverage, release, publish, Docker image, Pages, and artifact workflows still match the documented contract.
  - Check generated artifacts, release tags, published images, and Pages outputs for source-to-public drift.
  - File concrete follow-up issues for failing gates, stale artifacts, missing release prerequisites, or undocumented workflow changes.
  - Do not perform production deployment from this runbook unless the operator explicitly requests that deployment.

  Deliverables:
  - Recorded gate status and artifact surfaces inspected.
  - Follow-up issues for each reproducible CI, release, publish, or artifact drift problem.
  - A short `Last run:` note with commands run and any skipped surfaces.

  Validation:
  - Use repository-native `make` targets or documented release helpers for checks.
  - Confirm release and deployment ownership boundaries remain separate.
  - Confirm public or published artifacts match the intended source revision when that surface is inspected.

- [ ] [M405R] (P1) Code contract and static hygiene
  Goal:
  Keep source contracts explicit, current, and statically guarded against policy drift.

  Requirements:
  - Cadence: run monthly and before large refactors.
  - Scan for dead code, unused exports, duplicated literals, silent fallbacks, legacy aliases, compatibility reads, and zero-but-invalid domain states.
  - Check static analysis, coverage, schema, and contract guards that are supposed to prevent drift.
  - File focused Maintenance issues for each concrete violation instead of broad cleanup placeholders.
  - Keep the current canonical contract only; do not preserve obsolete behavior unless a product requirement explicitly says so.

  Deliverables:
  - Issue entries for each actionable static hygiene or contract violation.
  - Notes on static tools, searches, and contract guards used during the pass.
  - A short `Last run:` note with clean result or follow-up issue IDs.

  Validation:
  - Rerun the relevant static checks, contract tests, or repository searches used to identify drift.
  - Confirm every finding has a narrow follow-up issue and does not duplicate existing backlog work.
  - Confirm no implementation changes were mixed into the audit unless separately requested.

- [ ] [M406R] (P1) Production drift and health
  Goal:
  Detect when production, public, or scheduled runtime state has drifted from the intended repository contract.

  Requirements:
  - Cadence: run weekly for deployed services and after each publish or deploy.
  - Compare current source, runtime configuration, published images, public routes, scheduled jobs, and health checks for drift.
  - Inspect real operator-facing surfaces rather than assuming merged source is deployed.
  - File follow-up issues for stale images, stale Pages output, missing routes, failed monitors, invalid production config, or undocumented runtime differences.
  - Stop before production deploy or destructive operator actions unless the operator explicitly requests them.

  Deliverables:
  - Recorded source revision, public artifact, route, image, or health surfaces inspected.
  - Follow-up issues for each source-to-runtime drift finding.
  - A short `Last run:` note with evidence links or commands used.

  Validation:
  - Verify inspected production or public surfaces directly where access is available.
  - Confirm any deploy-required finding is filed with the exact publish/deploy boundary and owner.
  - Confirm no production state was changed by the audit unless explicitly requested.

- [ ] [M407R] (P2) Documentation and runbook hygiene
  Goal:
  Keep durable documentation and runbooks aligned with the current behavior users and operators actually rely on.

  Requirements:
  - Cadence: run before release cuts and after merge bursts that change user-facing or operator-facing behavior.
  - Review README, ARCHITECTURE, PRD, CHANGELOG, docs, runbooks, setup guides, and local workflow notes for stale behavior or missing new contracts.
  - Update docs when closed issues changed durable behavior, public APIs, operator workflows, release semantics, or deployment expectations.
  - Remove or rewrite stale instructions instead of preserving obsolete alternatives.
  - File separate issues for documentation gaps that require product or implementation decisions.

  Deliverables:
  - Updated documentation or filed follow-up issues for each gap.
  - A short `Last run:` note listing docs inspected and changes made.
  - Cross-references from archived issue history to durable docs when useful.

  Validation:
  - Check links, command names, paths, and public contract descriptions touched by the pass.
  - Confirm docs describe the current canonical path only.
  - Confirm issue archive and active tracker references remain consistent.
## 2026-07-10

- [ ] (P1) Repository-owned Git Release Pages and container toolchain
  Status: Unresolved pending validation.
  Release, publication, and Pages deployment executed mutable scripts from the sibling `agentSkills` repository. The current contract vendors the canonical scripts locally and requires black-box coverage for distinct source/release commits, empty `.nojekyll`, source-marker acceptance, and release-commit-marker rejection. Validation: `make test`, `make lint`, and `make ci`.
