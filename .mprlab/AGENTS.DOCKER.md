# AGENTS.DOCKER.md

## Scope

Docker and container guidance for this repository. Use this guide only when Dockerfiles, Compose files, container images, or container deployment manifests are present.

## Rules

- Containers run as `root` unless the repo has a current explicit requirement otherwise.
- Prefer multi-stage Dockerfiles.
- Copy only necessary runtime artifacts into final images.
- Keep environment configuration centralized in documented env files or deployment manifests.
- Do not bake secrets into images.
- Development images must exercise current unmerged source when used for validation.
- Production images must document their base image and publish path.
- For Go builds in Docker, set `CGO_ENABLED=0` unless CGO is necessary and documented.

## Orchestration Boundaries

- Treat local orchestration and production orchestration as different contracts.
- MPR Lab has no current plan to put these contracts together.
- Use the permanent versionless selected-manifest contract for production.
- Reject a selected manifest that contains `schema_version`.
- Each later gateway must accept every manifest that an earlier versionless gateway accepted.
- A field added to an existing shape must be optional and have one canonical default.
- The gateway must normalize that default before manifest identity calculation.
- Keep app-owned local orchestration during a versionless selected-manifest cutover.
- Keep local Compose files, configuration, scripts, `make up`, `make down`, and local tests during a production contract change.
- A local topology can be different from the production topology.
- Keep local orchestration files outside `.mprlab/deploy/`.
- Do not use local orchestration files as production lifecycle inputs.
- For a deployable application, use `.mprlab/deploy/resources.yml` as the only
  tracked production deployment file.
- Permit the ignored `.mprlab/deploy/.env` file as the canonical private deployment input.
- A container can create a GitHub Pages artifact. The container does not own the public website hostname.
- Publish the container output through the `github_pages` resource.
- Keep each `caddy_route` on a hostname that differs from the GitHub Pages domain.

## Validation

- For a behavior change, start with an integration test through the real container entry point.
- Use dependency injection for integration scenarios that are difficult to reproduce.
- Keep the product logic under test real.
- Use `.mprlab/POLICY.md` for validation.
- During the change, run the smallest container target that validates the changed contract.
- Build the image locally when Dockerfile changes affect runtime behavior.
- Run the service or container smoke test when available.
- Confirm exposed ports, health checks, volumes, and env variables match the documented deployment contract.
