#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  publish_container_artifacts.sh

Loads container archives prepared by make release, pushes platform images, and
creates the version and latest manifests. It never builds an image.
USAGE
}

if [[ $# -gt 0 ]]; then
  case "$1" in --help|-h) usage; exit 0 ;; *) echo "error: no arguments are supported" >&2; exit 1 ;; esac
fi

command -v docker >/dev/null 2>&1 || { echo "error: docker is required" >&2; exit 1; }
command -v gh >/dev/null 2>&1 || { echo "error: gh is required" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "error: python3 is required" >&2; exit 1; }
docker buildx version >/dev/null 2>&1 || { echo "error: docker buildx is required" >&2; exit 1; }

repo_root="$(git rev-parse --show-toplevel)"
artifact_dir="$(git rev-parse --git-path mprlab-release)"
[[ "${artifact_dir}" == /* ]] || artifact_dir="${repo_root}/${artifact_dir}"
helper="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/release_helper.py"
"${helper}" verify-release-artifact >/dev/null
release_version="$(python3 - "${artifact_dir}/manifest.json" <<'PY'
import json
import sys
print(json.load(open(sys.argv[1], encoding="utf-8"))["version"])
PY
)"
publish_timeout="${PUBLISH_CONTAINER_TIMEOUT_SECONDS:-1200}"
[[ "${publish_timeout}" =~ ^[1-9][0-9]*$ ]] || { echo "error: PUBLISH_CONTAINER_TIMEOUT_SECONDS must be a positive integer" >&2; exit 1; }

mapfile -t descriptors < <(find "${artifact_dir}/payloads/containers" -mindepth 2 -maxdepth 2 -name container.json -type f | LC_ALL=C sort)
[[ "${#descriptors[@]}" -gt 0 ]] || { echo "error: no prepared container artifacts found; run make release" >&2; exit 1; }

if python3 - "${descriptors[@]}" <<'PY'
import json
import sys
raise SystemExit(0 if any(json.load(open(path, encoding="utf-8"))["image"].startswith("ghcr.io/") for path in sys.argv[1:]) else 1)
PY
then
  registry_username="$(gh api user --jq .login)"
  registry_token="$(gh auth token)"
  printf '%s' "${registry_token}" | timeout -k 30s -s SIGKILL 30s docker login ghcr.io --username "${registry_username}" --password-stdin
  unset registry_token
fi

for descriptor in "${descriptors[@]}"; do
  metadata="$(python3 - "${descriptor}" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
if data.get("schema_version") != 1 or data.get("artifact_kind") != "mprlab.container":
    raise SystemExit("invalid container artifact descriptor")
print(data["name"])
print(data["image"])
print(data["version"])
for platform in data["platforms"]:
    print("\t".join([platform["platform"], platform["token"], platform["local_ref"], platform["image_id"], platform["archive"], platform["sha256"]]))
PY
)"
  name="$(sed -n '1p' <<<"${metadata}")"
  image="$(sed -n '2p' <<<"${metadata}")"
  version="$(sed -n '3p' <<<"${metadata}")"
  [[ "${version}" == "${release_version}" ]] || { echo "error: ${name} was prepared for ${version}, expected ${release_version}" >&2; exit 1; }
  publish_latest="true"
  if [[ "${version}" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+- ]]; then
    publish_latest="false"
  fi
  if [[ "${image}" == */* && "${image%%/*}" == *.* && "${image}" != ghcr.io/* ]]; then
    echo "error: unsupported explicit container registry for ${image}" >&2
    exit 1
  fi
  sources=()

  while IFS=$'\t' read -r platform token local_ref expected_image_id archive_relative expected_sha256; do
    [[ -n "${platform}" ]] || continue
    archive="${artifact_dir}/${archive_relative}"
    actual_sha256="$(shasum -a 256 "${archive}" | awk '{print $1}')"
    [[ "${actual_sha256}" == "${expected_sha256}" ]] || { echo "error: container archive hash mismatch: ${archive_relative}" >&2; exit 1; }
    timeout -k "${publish_timeout}s" -s SIGKILL "${publish_timeout}s" docker load --input "${archive}" >/dev/null
    actual_image_id="$(docker image inspect "${local_ref}" --format '{{.Id}}')"
    [[ "${actual_image_id}" == "${expected_image_id}" ]] || { echo "error: loaded image does not match prepared ${name} ${platform}" >&2; exit 1; }
    platform_ref="${image}:${version}-${token}"
    docker tag "${local_ref}" "${platform_ref}"
    echo "==> [publish] Pushing ${platform_ref}"
    timeout -k "${publish_timeout}s" -s SIGKILL "${publish_timeout}s" docker push "${platform_ref}"
    sources+=("${platform_ref}")
  done < <(tail -n +4 <<<"${metadata}")

  [[ "${#sources[@]}" -gt 0 ]] || { echo "error: ${name} has no prepared platforms" >&2; exit 1; }
  echo "==> [publish] Creating ${image}:${version}"
  timeout -k "${publish_timeout}s" -s SIGKILL "${publish_timeout}s" docker buildx imagetools create --tag "${image}:${version}" "${sources[@]}"
  version_digest="$(docker buildx imagetools inspect "${image}:${version}" | awk '/^Digest:/ {print $2; exit}')"
  [[ -n "${version_digest}" ]] || { echo "error: published version digest is missing for ${image}:${version}" >&2; exit 1; }
  if [[ "${publish_latest}" == "true" ]]; then
    echo "==> [publish] Updating ${image}:latest"
    timeout -k "${publish_timeout}s" -s SIGKILL "${publish_timeout}s" docker buildx imagetools create --tag "${image}:latest" "${sources[@]}"
    latest_digest="$(docker buildx imagetools inspect "${image}:latest" | awk '/^Digest:/ {print $2; exit}')"
    [[ "${version_digest}" == "${latest_digest}" ]] || { echo "error: published version and latest digests differ for ${image}" >&2; exit 1; }
  else
    echo "==> [publish] Leaving ${image}:latest unchanged for prerelease ${version}"
  fi
  echo "Published ${image}:${version} at ${version_digest}."
done
