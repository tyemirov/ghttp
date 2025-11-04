# Notes

## Role

You are a staff level full stack engineer. Your task is to **re-evaluate and refactor the TAuth repository** according to the coding standards already written in **AGENTS.md**.  
**Read-only:** Keep operational notes only. Record all issues in `ISSUES.md`. Track changes in the `CHANGELOG.md`

## Context

- AGENTS.md defines all rules: naming, state/event principles, structure, testing, accessibility, performance, and security.
- The repo uses Alpine.js, CDN scripts only, no bundlers.
- Event-scoped architecture: components communicate via `$dispatch`/`$listen`; prefer DOM-scoped events; `Alpine.store` only for true shared domain state.
- The backend uses Go language ecosystem

## Your tasks

1. **Read AGENTS.md first** → treat it as the _authoritative style guide_.
2. **Scan the codebase** → identify violations (inline handlers, globals, duplicated strings, lack of constants, cross-component state leakage, etc.).
3. **Generate PLAN.md** → bullet list of problems and refactors needed, scoped by file. PLAN.md is a part of PR metadata. It's a transient document outlining the work on a given issue. Do not commit PLAN.md; copy its content into the PR description.
4. **Refactor in small commits** →
   Front-end:
   - Inline → Alpine `x-on:`
   - Buttons → standardized Alpine factories/events
   - Notifications → event-scoped listeners (DOM-scoped preferred)
   - Strings → move to `constants.js`
   - Utilities → extract into `/js/utils/`
   - Composition → normalize `/js/app.js` as Alpine composition root
     Backend:
   - Use "object-oreinted" stye of functions attached to structs
   - Prioritize data-driven solutions over imperative approach
   - Design and use shared components
5. **Tests** → Add/adjust Puppeteer tests for key flows (button → event → notification; cross-panel isolation). Prioritize end-2-end and integration tests.
6. **Docs** → Update README and CHANGELOG.md with new event contracts, removed globals, and developer instructions.
7. **Timeouts** Prepend every CLI command with `timeout -k <N>s -s SIGKILL <N>s <command>`. This is mandatory for all commands (local dev, CI, docs, scripts). Pick `<N>` appropriate to the operation; avoid indefinite waits. The Node test harness enforces per-test budgets but the shell-level timeout remains required.
   7a. Any individual test or command must be terminated in 30s. The only long running command is a full test, which must be terminated in 350s. There are no exception to this rule, and no extension of time: each individual test or command must finish under 30s.

## Output requirements

- Always follow AGENTS.md rules (do not restate them, do not invent new ones).
- Output a **PLAN.md** first, then refactor step-by-step.
- Only modify necessary files.
- Treat `NOTES.md` as read-only; never edit it during an implementation cycle.
- Only touch the following markdown files while delivering work: `ISSUES.md` (append-only status log), `PLAN.md` (local, untracked scratchpad), and `CHANGELOG.md` (post-completion history).
- If `PLAN.md` becomes tracked, remove it from history with `git filter-repo --path PLAN.md --invert-paths` before continuing.
- Descriptive identifiers, no single-letter names.
- End with a short summary of changed files and new event contracts.

**Begin by reading AGENTS.md and generating PLAN.md now.**

## Rules of engagement

Review the backlog in `ISSUES.md`. Make a plan for autonomously fixing every item under Features, BugFixes, Improvements, Maintenance. Ensure no regressions. Ensure adding tests. Lean into integration tests. Fix every issue. Document the changes directly in `ISSUES.md`. Continue cycling through the backlog without pausing for additional confirmation until every marked item is complete.

Fix issues one by one, working sequentially.

1. The production git branch is called `master`. The `main` branch does not exist.
2. Before making any changes, create a new git branch with a descriptive name (e.g., `bugfix/GN-58-editor-duplicate-preview`) and branch from the previous issue’s branch. Use the taxonomy prefixes improvement/, feature/, bugfix/, maintenace/ followed by the issue ID and a short description. Respect branch name limits.
3. On that branch, describe the issue through tests.
   3a. Add comprehensive regression coverage that initially fails on the branch prior to implementing the fix (run the suite to observe the failure before proceeding).
   3b. Ensure AGENTS.md coding standards are checked and test names/descriptions reflect those rules.
4. Fix the issue
5. Rerun the tests
6. Repeat pp 2-4 untill the issue is fixed:
   6a. old and new comprehensive tests are passing
   6b. Confirm black-box contract aligns with event-driven architecture (frontend) or data-driven logic (backend).
   6c. If an issue can not be resolved after 3 carefull iterations, - mark the issue as [Blocked]. - document the reason for the bockage. - commit the changes into a separate branch called "blocked/<issue-id>". - work on the next issue from the divergence point of the previous issue.
7. Write a nice comprehensive commit message AFTER EACH issue is fixed and tested and covered with tests.
8. Optional: update the README in case the changes warrant updated documentation (e.g. have user-facing consequences)
9. Optional: ipdate the PRD in case the changes warrant updated product requirements (e.g. change product undestanding)
10. Optional: update the code examples in case the changes warrant updated code examples
11. Mark an issue as done ([X]) in `ISSUES.md` after the issue is fixed: New and existing tests are passing without regressions
12. After each issue-level commit, push the local branch to the remote with `git push -u origin <branch>` so the branch tracks its remote counterpart. Subsequent pushes should use `git push` only. Never push to arbitrary remotes or untracked branch names.
13. Repeat the entire cycle immediately for the next issue, continuing until all backlog items are resolved. Do not wait for additional prompts between issues.

Do not work on all issues at once. Work at one issue at a time sequntially.

Working with git bracnhes you are forbidden from using --force, rebase or cherry-pick operations. Any changes in history are strictly and explcitly forbidden, The git branches only move up, and any issues are fixed in the next sequential commit. Only merges and sequential progression of changes.

Leave Features, BugFixes, Improvements, Maintenance sections empty when all fixes are implemented but don't delete the sections themselves.

## Pre-finish Checklist

1. Update `PLAN.md` for the active issue, then clear it before starting working on the next issue.
2. Ensure the issue entry in `ISSUES.md` is marked `[x]` and includes an appended resolution note.
3. Run tests, whether `go test ./...` or `npm test` or the relevant suite and resolve all failures.
4. Commit only the intended changes and push the branch to origin. Esnure that the local branch is tracking the remote.
5. Verify no required steps were skipped; if anything cannot be completed, stop and ask before proceeding.

## Issue Tracking

All feature, improvement, bugfix, and maintenance backlog entries now live in `ISSUES.md`. This file remains append-only for process notes.

_Use `PLAN.md` (ignored by git) as a scratchpad for the single active issue; do not commit it._

## Action Items

The deliverables are code changes. Sequentially open PRs use `gh` utility after finishing your autonomous work. Present a list of opened PRs at the end for reviews

    1. Read the files that guide the development: README.md , PRD.md  , AGENTS.md , NOTES.md , ARCHITECTURE.md .
    2. Run the tests
    3. Plan the required changes to close the open issues. If issues are missing based on analysis of the code, add them and plan to fix them.
    4. Use PLAN.md for an individual issue to plan the fix
    5. Read the documentation of gthe 3rd party libraries before implementing changes
