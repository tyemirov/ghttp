# gHTTP

[![GitHub release](https://img.shields.io/github/release/tyemirov/ghttp.svg)](https://github.com/tyemirov/ghttp/releases)

gHTTP is a Go-powered file server that mirrors the ergonomics of `python -m http.server`, adds structured zap-based request logging, renders mardown files as HTML, and provisions self-signed HTTPS certificates for local development.

*gHTTP is fast.*

## Installation

### Docker

Pull and run the latest Docker image:

```bash
docker pull ghcr.io/tyemirov/ghttp:latest
docker run -p 8000:8000 -v $(pwd):/data ghcr.io/tyemirov/ghttp:latest --directory /data
```

Custom port and directory examples:

```bash
# Serve current directory on port 9000
docker run -p 9000:9000 -v $(pwd):/data ghcr.io/tyemirov/ghttp:latest --directory /data 9000

# Serve with HTTPS (requires certificate setup)
docker run -p 8443:8443 -v $(pwd):/data -v ~/.config/ghttp:/root/.config/ghttp ghcr.io/tyemirov/ghttp:latest --directory /data --https 8443
```

### Releases

Download the latest binaries from the [Releases page](https://github.com/tyemirov/ghttp/releases).

### Go toolchain

Install gHTTP with the Go toolchain:

```
go install github.com/tyemirov/ghttp/cmd/ghttp@latest
```

Go 1.24.6 or newer is required, matching the minimum version declared in `go.mod`.

After installation the `ghttp` binary is placed in `$GOBIN` (or `$GOPATH/bin`). The root command accepts an optional positional `PORT` argument so existing workflows keep working.

### Usage examples

| Scenario | Example command | Notes |
| --- | --- | --- |
| Serve the current working directory on the default HTTP port 8000 | `ghttp` | Mirrors `python -m http.server` with structured logging. |
| Serve a specific directory on a chosen port | `ghttp --directory /srv/www 9000` | Exposes `/srv/www` at <http://localhost:9000>. |
| Bind to a specific interface | `ghttp --bind 192.168.1.5 8000` | Restricts listening to the provided IP address. |
| Serve HTTPS with an existing certificate | `ghttp --tls-cert cert.pem --tls-key key.pem 8443` | Keeps backwards-compatible manual TLS support. |
| Serve HTTPS with self-signed certificates | `ghttp --https` | Defaults to port 8443, installs the development CA, serves HTTPS, and removes credentials on exit. |
| Disable Markdown rendering | `ghttp --no-md` | Serves raw Markdown assets without HTML conversion. |
| Switch logging format | `ghttp --logging-type JSON` | Emits structured JSON logs instead of the default console view. |

### Key capabilities
* Choose between HTTP/1.0 and HTTP/1.1 with `--protocol`/`-p`; the server tunes keep-alive behaviour automatically.
* Provision a development certificate authority with `ghttp --https`, storing it at `~/.config/ghttp/certs` and installing it into macOS, Linux, or Windows trust stores using native tooling.
* Issue SAN-aware leaf certificates on demand whenever HTTPS is enabled, covering `localhost`, `127.0.0.1`, `::1`, and additional hosts supplied via repeated `--https-host` flags or Viper configuration.
* Render Markdown files (`*.md`) to HTML automatically, treat `README.md` as a directory landing page, and skip the feature entirely with `--no-md` or `serve.no_markdown: true` in configuration.
* When Firefox is installed, automatically configure its profiles to trust the generated certificates so browser warnings disappear on the next restart.
* Suppress automatic directory listings by exporting `GHTTPD_DISABLE_DIR_INDEX=1`; the handler returns HTTP 403 for directory roots.
* Configure every flag via `~/.config/ghttp/config.yaml` or environment variables prefixed with `GHTTP_` (for example, `GHTTP_SERVE_DIRECTORY=/srv/www`).

### Flags and environment variables
Flags map to Viper configuration keys. Environment variables use the `GHTTP_` prefix with dots replaced by underscores.

| Flag | Environment variable | Notes |
| --- | --- | --- |
| `PORT` (positional) | `GHTTP_SERVE_PORT` | Defaults to 8000 for HTTP and 8443 when `--https` is enabled. |
| `--config` | `GHTTP_CONFIG_FILE` | Overrides the default config lookup (`~/.config/ghttp/config.yaml`). |
| `--bind` | `GHTTP_SERVE_BIND_ADDRESS` | Empty means all interfaces; logs display `localhost` for empty/`0.0.0.0`/`127.0.0.1`. |
| `--directory` | `GHTTP_SERVE_DIRECTORY` | Directory to serve files from. Defaults to the working directory. |
| `--protocol` | `GHTTP_SERVE_PROTOCOL` | HTTP protocol version (use the full value, for example, `HTTP/1.0` or `HTTP/1.1`). |
| `--no-md` | `GHTTP_SERVE_NO_MARKDOWN` | Disables Markdown rendering. |
| `--browse` | `GHTTP_SERVE_BROWSE` | Folder URLs always return a directory listing, even if index.html or README.md exists. Direct file requests are handled by the same normal file pipeline with no filename preference (including index files); Markdown requests still render when Markdown rendering is enabled. Overrides `GHTTPD_DISABLE_DIR_INDEX`. |
| `--logging-type` | `GHTTP_SERVE_LOGGING_TYPE` | CONSOLE or JSON. |
| `--proxy` | `GHTTP_SERVE_PROXIES` | Enables reverse proxy. Repeatable from=to mapping (for example, `/api=http://backend:8081`); backend can be `http://` or `https://` regardless of frontend scheme; env uses comma-separated list. |
| `--proxy-path` | `GHTTP_SERVE_PROXY_PATH_PREFIX` | Legacy from-path prefix (for example, `/api`); requires `--proxy-backend`. |
| `--proxy-backend` | `GHTTP_SERVE_PROXY_BACKEND` | Legacy to-backend URL (for example, `http://backend:8081`); requires `--proxy-path`. |
| `--https` | `GHTTP_SERVE_HTTPS` | Enables self-signed HTTPS using the development certificate authority (SANs from `--https-host`); mutually exclusive with `--tls-cert` and `--tls-key`. |
| `--https-host` | `GHTTP_HTTPS_HOSTS` | Repeatable flag; env uses comma-separated list; only used with `--https` and included in generated HTTPS certificates. |
| `--tls-cert` | `GHTTP_SERVE_TLS_CERTIFICATE` | Provide with `--tls-key`; cannot combine with `--https`. |
| `--tls-key` | `GHTTP_SERVE_TLS_PRIVATE_KEY` | Provide with `--tls-cert`; cannot combine with `--https`. |

Legacy single mapping: `--proxy-path` (from) + `--proxy-backend` (to) remain supported when `--proxy`/`GHTTP_SERVE_PROXIES` are unset.

Positional port arguments map to `GHTTP_SERVE_PORT` for `ghttp`. When no port is provided, gHTTP defaults to 8000 for HTTP and 8443 when `--https` is enabled.


### Browser trust behaviour
| Browser | Trust source | Restart needed? | Notes |
| --- | --- | --- | --- |
| Safari (macOS) | System keychain | No | macOS keychain updates apply immediately to Safari and other WebKit clients. |
| Chrome / Edge | OS certificate store | No | Chromium-based browsers rely on the OS trust store and accept the CA on the next handshake. |
| Firefox | Firefox NSS store or enterprise roots | Yes | Profiles are updated automatically: if `certutil` is available the CA is imported, otherwise `security.enterprise_roots.enabled` is set via `user.js`. Restart Firefox to apply the change. |
| Other browsers | OS certificate store | No | Most modern browsers reuse the system trust store; no manual action required. |

## File Serving Behavior
The server delegates file handling to the Go standard library's `http.FileServer`,
initializing the handler with the target directory via `http.FileServer(http.Dir(...))`.
Because this handler reads content directly from disk for each request, file
changes are reflected immediately without requiring a filesystem watcher or
manual reload step.

Only two response headers are set by default: `Server: ghttpd` is always
emitted, and when HTTP/1.0 is negotiated the handler also sets
`Connection: close`. No cache-control or time-to-live directives are provided,
so clients and intermediate caches decide their own policies.

If you need custom caching semantics, wrap the file server handler with your own
`http.Handler` that sets `Cache-Control`, `ETag`, or other headers before
forwarding the request to the embedded `http.FileServer` instance.

## License
This project is distributed under the terms of the [MIT License](./LICENSE).
Copyright (c) 2025 Vadym Tyemirov. Refer to the license file for the complete text, including permissions and limitations.
