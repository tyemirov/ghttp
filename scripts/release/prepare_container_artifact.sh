#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  prepare_container_artifact.sh --name <name> --image <registry/repository> [options]

Builds one or more platform-specific container images into local Docker archives
under the active .git/mprlab-release staging area. It never logs in or pushes.

Options:
  --file <path>          Dockerfile path. Default: Dockerfile
  --context <path>       Docker build context. Default: .
  --platforms <list>     Comma-separated platforms. Default: linux/amd64,linux/arm64
  --build-arg <value>    Repeatable Docker build argument
  --build-context <val>  Repeatable named build context
  --secret <value>       Repeatable BuildKit secret
  --label <value>        Repeatable image label
  --target <name>        Dockerfile target stage
  --pull                 Refresh referenced base images while building
  --help                 Show this help text
USAGE
}

name=""
image=""
dockerfile="Dockerfile"
context="."
platforms="linux/amd64,linux/arm64"
target=""
pull="false"
build_args=()
build_contexts=()
secrets=()
labels=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --name) [[ $# -ge 2 ]] || { echo "error: --name requires a value" >&2; exit 1; }; name="$2"; shift 2 ;;
    --image) [[ $# -ge 2 ]] || { echo "error: --image requires a value" >&2; exit 1; }; image="$2"; shift 2 ;;
    --file) [[ $# -ge 2 ]] || { echo "error: --file requires a value" >&2; exit 1; }; dockerfile="$2"; shift 2 ;;
    --context) [[ $# -ge 2 ]] || { echo "error: --context requires a value" >&2; exit 1; }; context="$2"; shift 2 ;;
    --platforms) [[ $# -ge 2 ]] || { echo "error: --platforms requires a value" >&2; exit 1; }; platforms="$2"; shift 2 ;;
    --build-arg) [[ $# -ge 2 ]] || { echo "error: --build-arg requires a value" >&2; exit 1; }; build_args+=("$2"); shift 2 ;;
    --build-context) [[ $# -ge 2 ]] || { echo "error: --build-context requires a value" >&2; exit 1; }; build_contexts+=("$2"); shift 2 ;;
    --secret) [[ $# -ge 2 ]] || { echo "error: --secret requires a value" >&2; exit 1; }; secrets+=("$2"); shift 2 ;;
    --label) [[ $# -ge 2 ]] || { echo "error: --label requires a value" >&2; exit 1; }; labels+=("$2"); shift 2 ;;
    --target) [[ $# -ge 2 ]] || { echo "error: --target requires a value" >&2; exit 1; }; target="$2"; shift 2 ;;
    --pull) pull="true"; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "error: unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done

[[ "${name}" =~ ^[a-z0-9][a-z0-9._-]*$ ]] || { echo "error: --name must use lowercase artifact-safe characters" >&2; exit 1; }
[[ -n "${image}" ]] || { echo "error: --image is required" >&2; exit 1; }
[[ -n "${RELEASE_VERSION:-}" ]] || { echo "error: RELEASE_VERSION is required" >&2; exit 1; }
[[ -n "${RELEASE_ARTIFACT_DIR:-}" ]] || { echo "error: RELEASE_ARTIFACT_DIR is required" >&2; exit 1; }
[[ -f "${RELEASE_ARTIFACT_DIR}/staging.json" ]] || { echo "error: release staging area is not initialized" >&2; exit 1; }
[[ -f "${dockerfile}" ]] || { echo "error: Dockerfile not found: ${dockerfile}" >&2; exit 1; }
[[ -d "${context}" ]] || { echo "error: build context not found: ${context}" >&2; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "error: docker is required" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "error: python3 is required" >&2; exit 1; }
docker buildx version >/dev/null 2>&1 || { echo "error: docker buildx is required" >&2; exit 1; }
build_timeout="${RELEASE_CONTAINER_BUILD_TIMEOUT_SECONDS:-1200}"
save_timeout="${RELEASE_CONTAINER_SAVE_TIMEOUT_SECONDS:-350}"
[[ "${build_timeout}" =~ ^[1-9][0-9]*$ && "${save_timeout}" =~ ^[1-9][0-9]*$ ]] || { echo "error: container timeouts must be positive integers" >&2; exit 1; }

artifact_root="${RELEASE_ARTIFACT_DIR}/payloads/containers/${name}"
rm -rf "${artifact_root}"
mkdir -p "${artifact_root}"
metadata_rows="$(mktemp)"
trap 'rm -f "${metadata_rows}"' EXIT

IFS=',' read -r -a platform_list <<<"${platforms}"
[[ "${#platform_list[@]}" -gt 0 ]] || { echo "error: at least one platform is required" >&2; exit 1; }

for platform in "${platform_list[@]}"; do
  [[ "${platform}" =~ ^linux/(amd64|arm64)$ ]] || { echo "error: unsupported release platform: ${platform}" >&2; exit 1; }
  platform_token="${platform//\//-}"
  version_token="$(printf '%s' "${RELEASE_VERSION}" | tr -c 'A-Za-z0-9_.-' '-')"
  local_ref="mprlab-release.local/${name}:${version_token}-${platform_token}"
  archive="${artifact_root}/${platform_token}.tar"
  build_command=(docker buildx build --platform "${platform}" --load --file "${dockerfile}" --tag "${local_ref}")
  [[ "${pull}" == "true" ]] && build_command+=(--pull)
  [[ -n "${target}" ]] && build_command+=(--target "${target}")
  for value in "${build_args[@]}"; do build_command+=(--build-arg "${value}"); done
  for value in "${build_contexts[@]}"; do build_command+=(--build-context "${value}"); done
  for value in "${secrets[@]}"; do build_command+=(--secret "${value}"); done
  for value in "${labels[@]}"; do build_command+=(--label "${value}"); done
  build_command+=("${context}")

  echo "==> [release] Building ${name} for ${platform}"
  timeout -k "${build_timeout}s" -s SIGKILL "${build_timeout}s" "${build_command[@]}"
  image_id="$(docker image inspect "${local_ref}" --format '{{.Id}}')"
  timeout -k "${save_timeout}s" -s SIGKILL "${save_timeout}s" docker save --output "${archive}" "${local_ref}"
  archive_sha256="$(shasum -a 256 "${archive}" | awk '{print $1}')"
  printf '%s\t%s\t%s\t%s\t%s\n' "${platform}" "${platform_token}" "${local_ref}" "${image_id}" "${archive_sha256}" >>"${metadata_rows}"
done

python3 - "${artifact_root}/container.json" "${name}" "${image}" "${RELEASE_VERSION}" "${RELEASE_ARTIFACT_DIR}" "${metadata_rows}" <<'PY'
import json
import pathlib
import sys

output, name, image, version, artifact_dir, rows_path = sys.argv[1:]
artifact_root = pathlib.Path(artifact_dir).resolve()
platforms = []
for row in pathlib.Path(rows_path).read_text(encoding="utf-8").splitlines():
    platform, token, local_ref, image_id, sha256 = row.split("\t")
    archive = pathlib.Path(output).parent / f"{token}.tar"
    platforms.append(
        {
            "platform": platform,
            "token": token,
            "local_ref": local_ref,
            "image_id": image_id,
            "archive": archive.resolve().relative_to(artifact_root).as_posix(),
            "sha256": sha256,
        }
    )
document = {
    "schema_version": 1,
    "artifact_kind": "mprlab.container",
    "name": name,
    "image": image,
    "version": version,
    "platforms": platforms,
}
pathlib.Path(output).write_text(json.dumps(document, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY

echo "Prepared container artifact ${name} for ${platforms}."
