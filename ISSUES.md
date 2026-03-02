# ISSUES (Open Only)

- [ ] [G001] Add response-cache and stream-delivery controls for app shells and SSE.
  Summary:
  ghttp needs first-class runtime controls for frontend cache policy and long-lived event streams so apps can avoid stale JS/CSS shells while preserving efficient caching for static assets and stable SSE delivery.
  Requested controls:
  - Per-route Cache-Control configuration (`no-store`, `no-cache`, and immutable/static profiles).
  - ETag and conditional request controls (`If-None-Match` handling toggles).
  - SSE-friendly streaming mode (disable buffering/compression where needed, immediate flush semantics).
  - Simple config surface (CLI flags and/or config file) so container entrypoints do not need custom reverse-proxy templates for these basics.
