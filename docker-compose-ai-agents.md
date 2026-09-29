# gHTTP Docker Compose Integration (for AI agents)

This document describes how automation and AI agents should integrate the `ghttp` file server into `docker-compose` stacks. The examples assume the published container image and the current CLI behavior of `ghttp`.

The goals are:

- Produce working, repeatable `docker-compose.yml` snippets.
- Keep configuration declarative via environment variables.
- Avoid assumptions that do not hold for the distroless runtime image.

## Quick reference

- Image: `ghcr.io/tyemirov/ghttp:latest`
- Entrypoint: `/app/ghttp`
- Default listen port: `8000` for HTTP, `8443` when `serve.https` is true
- Container metadata: `EXPOSE 8000` in the Dockerfile
- Default serve directory: current working directory (`.` inside `/app`)
- Configuration sources (highest precedence first):
  - CLI flags (e.g. `--directory`, positional `PORT`)
  - Environment variables (`GHTTP_*`)
  - Configuration file in the user config directory (`config.yaml`)
  - Built-in defaults

For most Compose use cases, prefer environment variables over configuration files. Use CLI flags only when something cannot be expressed via environment variables.

## Minimal service template

This pattern serves static content from a host directory using HTTP on port `8000`:

```yaml
services:
  ghttp:
    image: ghcr.io/tyemirov/ghttp:latest
    restart: unless-stopped
    ports:
      - "8000:8000"
    volumes:
      - ./public:/data:ro
    environment:
      GHTTP_SERVE_DIRECTORY: /data
      GHTTP_SERVE_PORT: "8000"
      GHTTP_SERVE_LOGGING_TYPE: JSON
```

Behavior:

- The container runs `ghttp` as PID 1 with working directory `/app`.
- `GHTTP_SERVE_DIRECTORY=/data` instructs gHTTP to serve the mounted directory instead of `/app`.
- `GHTTP_SERVE_PORT=8000` makes gHTTP listen on container port `8000`, matching the Dockerfile `EXPOSE` directive and the Compose port mapping.
- `GHTTP_SERVE_LOGGING_TYPE=JSON` produces structured logs on stdout, which is better for log collection and automated analysis.

## Configuration via environment variables

The CLI uses Viper with an `env` prefix of `GHTTP` and replaces dots in configuration keys with underscores. The most common configuration keys and their environment variable forms are:

- `config.file` → `GHTTP_CONFIG_FILE`
  - Path to a specific configuration file (overrides the default lookup in the user config directory).
- `serve.directory` → `GHTTP_SERVE_DIRECTORY`
  - Directory path inside the container to serve.
  - Default: `"."` (the process working directory).
- `serve.port` → `GHTTP_SERVE_PORT`
  - TCP port that gHTTP listens on inside the container.
  - Default: `"8000"` for HTTP, `"8443"` when `serve.https` is true and no port is provided.
- `serve.bind_address` → `GHTTP_SERVE_BIND_ADDRESS`
  - Bind address for the HTTP listener.
  - Default: `""` (equivalent to all interfaces).
  - For Compose, leaving this empty is usually sufficient; `0.0.0.0` is also safe when explicit binding is desired.
- `serve.protocol` → `GHTTP_SERVE_PROTOCOL`
  - HTTP protocol version: `"HTTP/1.0"` or `"HTTP/1.1"`.
  - Default: `"HTTP/1.1"`.
- `serve.no_markdown` → `GHTTP_SERVE_NO_MARKDOWN`
  - Boolean; when truthy, disables Markdown rendering and serves `.md` files as plain assets.
  - Default: `false`.
- `serve.browse` → `GHTTP_SERVE_BROWSE`
  - Boolean; when truthy, folder URLs always return a directory listing even if index.html or README.md exists.
  - Direct file requests use the regular file-serving pipeline with no filename preference (including index files); `.md` requests still render when Markdown rendering is enabled.
  - Default: `false`.
- `serve.logging_type` → `GHTTP_SERVE_LOGGING_TYPE`
  - Logging format: `"CONSOLE"` (human-oriented) or `"JSON"` (machine-oriented).
  - Default: `"CONSOLE"`.
- `serve.proxies` → `GHTTP_SERVE_PROXIES`
  - Comma-separated list of from=to mappings (for example, `/api=http://backend:8081`).
  - Use multiple entries to proxy multiple path prefixes.
- `serve.proxy_path_prefix` → `GHTTP_SERVE_PROXY_PATH_PREFIX`
  - Legacy single mapping (from-path prefix); requires `serve.proxy_backend`.
  - Only used when `serve.proxies` is empty.
- `serve.proxy_backend` → `GHTTP_SERVE_PROXY_BACKEND`
  - Legacy single mapping (to-backend URL); requires `serve.proxy_path_prefix`.
  - Only used when `serve.proxies` is empty.
- `serve.tls_certificate` → `GHTTP_SERVE_TLS_CERTIFICATE`
  - Path inside the container to a PEM-encoded TLS certificate.
  - Used together with `GHTTP_SERVE_TLS_PRIVATE_KEY`.
- `serve.tls_private_key` → `GHTTP_SERVE_TLS_PRIVATE_KEY`
  - Path inside the container to a PEM-encoded private key for the certificate.
- `serve.https` → `GHTTP_SERVE_HTTPS`
  - Boolean; when truthy, enables dynamic HTTPS using a self-signed development certificate authority.
  - In most containerized deployments, prefer manual TLS termination via `GHTTP_SERVE_TLS_CERTIFICATE` and `GHTTP_SERVE_TLS_PRIVATE_KEY` or terminate TLS at a separate reverse proxy.
- `https.certificate_directory` → `GHTTP_HTTPS_CERTIFICATE_DIRECTORY`
  - Directory where dynamic HTTPS certificates are generated and stored.
  - Default: a `certs` directory under the user config directory.
- `https.hosts` → `GHTTP_HTTPS_HOSTS`
  - Comma-separated list of hostnames/IPs included in the dynamic HTTPS certificate.
  - Defaults include `localhost`, `127.0.0.1`, and `::1`.

Additional directory listing control:

- `GHTTPD_DISABLE_DIR_INDEX`
  - When set to `"1"`, disables automatic directory listings (returns HTTP 403 for directory roots).
  - This is evaluated independently of the `GHTTP_` configuration and can be set alongside the other environment variables.

### Boolean values

Boolean configuration values use the same truthiness semantics as Viper and Go’s standard library. AI agents should use `"true"` or `"false"` for clarity. Acceptable truthy values include:

- `"1"`, `"true"`, `"yes"`, `"on"` (case-insensitive)

## Using CLI flags from Compose

Environment variables cover almost all use cases. When CLI flags are required, supply them via the Compose `command` field. The binary already serves as the entrypoint, so the `command` array corresponds to CLI arguments of `ghttp`.

To set directory and port via CLI flags instead of environment variables:

```yaml
services:
  ghttp:
    image: ghcr.io/tyemirov/ghttp:latest
    ports:
      - "8000:8000"
    volumes:
      - ./public:/data:ro
    command:
      - --directory
      - /data
      - "8000"
```

Notes for agents:

- Positional arguments after flags are interpreted as the port value.
- Do not prepend `ghttp` in the `command` list; the entrypoint is already configured in the image.
- For deterministic behavior across environments, prefer environment variables when both options are available.

## HTTPS and TLS in Compose

For container-based deployments, the recommended HTTPS strategy is to treat gHTTP as a plain HTTP file server and terminate TLS at an ingress or reverse-proxy layer. When direct HTTPS from gHTTP is required, use one of the following patterns.

### Manual TLS with existing certificates

When certificates are managed outside the container (for example, by a certificate management system or a volume with pre-provisioned credentials), mount them read-only and configure gHTTP via environment variables:

```yaml
services:
  ghttp:
    image: ghcr.io/tyemirov/ghttp:latest
    ports:
      - "8443:8443"
    volumes:
      - ./public:/data:ro
      - ./certs:/certs:ro
    environment:
      GHTTP_SERVE_DIRECTORY: /data
      GHTTP_SERVE_PORT: "8443"
      GHTTP_SERVE_TLS_CERTIFICATE: /certs/server.pem
      GHTTP_SERVE_TLS_PRIVATE_KEY: /certs/server-key.pem
      GHTTP_SERVE_LOGGING_TYPE: JSON
```

This configuration:

- Binds host port `8443` to container port `8443`.
- Serves static content from `/data` (mounted from `./public`).
- Uses the mounted certificate and key from `/certs`.

### Dynamic HTTPS inside the container

Dynamic HTTPS (`GHTTP_SERVE_HTTPS=true` or the `--https` flag) generates a self-signed development certificate authority and installs it into the container’s trust store. This is primarily useful for local development rather than production.

AI agents should treat dynamic HTTPS as an advanced option and only use it when:

- The container has permission to run trust-store modification commands for the target OS.
- The generated certificates do not need to persist beyond the lifetime of the container, or a dedicated volume is provided via `GHTTP_HTTPS_CERTIFICATE_DIRECTORY`.

In automated Compose deployments, it is usually safer to:

- Run gHTTP with plain HTTP inside the container.
- Terminate TLS at a reverse proxy (for example, Nginx, Caddy, Traefik, or a platform ingress).

## Common integration patterns

### 1. Standalone static file server

Use gHTTP as a standalone static file server fronted by a single port. This is the simplest pattern and requires only:

- A host directory with static assets, mounted into the container (for example, `./public:/data:ro`).
- `GHTTP_SERVE_DIRECTORY=/data`.
- A port mapping from host to container (for example, `"8000:8000"` with `GHTTP_SERVE_PORT=8000`).

### 2. Sidecar for another service

When another service generates files that should be exposed via HTTP, run gHTTP as a sidecar in the same Compose project:

- Mount the same volume into both containers (for example, `./shared-data:/data`).
- Configure gHTTP with `GHTTP_SERVE_DIRECTORY=/data`.
- Put both services on the same user-defined network; other services can reach gHTTP at `http://ghttp:PORT`.

This pattern keeps the file-serving concerns separate from the main application container while providing a simple static hosting solution.

### 3. Local development helper

For local development workflows, gHTTP can mimic `python -m http.server` with structured logging and Markdown rendering:

- Mount the current project directory: `.:/data`.
- Set `GHTTP_SERVE_DIRECTORY=/data`.
- Use `GHTTP_SERVE_LOGGING_TYPE=CONSOLE` for human-readable logs.
- Optionally set `GHTTP_SERVE_NO_MARKDOWN=true` when raw Markdown assets are desired.

## Distroless image constraints (important for AI agents)

The runtime image is based on `gcr.io/distroless/base-debian12`. This has several implications that AI agents must respect when generating Compose files:

- There is no shell such as `/bin/sh` or `/bin/bash` in the container.
  - Do not use `command: ["sh", "-c", "..."]` or similar patterns.
- Common utilities like `curl`, `wget`, or `bash` are not available.
  - Healthchecks that rely on shell commands will fail unless you build a custom image with those tools.
- The only supported entrypoint process is the `ghttp` binary itself.
  - Compose `command` should contain only gHTTP CLI flags and positional arguments.

When health checks or additional tooling are required:

- Use an external probe from the host or a dedicated utility sidecar container.
- Avoid mutating the official gHTTP image unless absolutely necessary; if customization is required, document the changes clearly in your own project.

## Recommendations for automated tooling

For robust integration, AI agents should:

- Prefer environment variables (`GHTTP_*`) instead of CLI flags where possible.
- Explicitly set `GHTTP_SERVE_DIRECTORY` and `GHTTP_SERVE_PORT` in Compose services.
- Use JSON logging (`GHTTP_SERVE_LOGGING_TYPE=JSON`) in production-like environments.
- Avoid dynamic HTTPS inside containers unless a human has explicitly requested it and provided suitable volumes and permissions.
- Keep volume mounts read-only (`:ro`) for static content, unless there is a documented need for write access.

Following these guidelines will produce predictable and maintainable Docker Compose configurations that work correctly with the current gHTTP container image and CLI behavior.
