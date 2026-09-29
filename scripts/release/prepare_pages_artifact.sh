#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  prepare_pages_artifact.sh --source <directory> [options]

Packages a static site as pages.tar.gz in the active local release artifact.
The archive includes .mprlab-release.json for deploy-time verification.

Options:
  --domain <domain>      Write a CNAME file with this domain
  --exclude <pattern>    Repeatable rsync exclusion pattern
  --help                 Show this help text
USAGE
}

source_directory=""
domain=""
excludes=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --source) [[ $# -ge 2 ]] || { echo "error: --source requires a value" >&2; exit 1; }; source_directory="$2"; shift 2 ;;
    --domain) [[ $# -ge 2 ]] || { echo "error: --domain requires a value" >&2; exit 1; }; domain="$2"; shift 2 ;;
    --exclude) [[ $# -ge 2 ]] || { echo "error: --exclude requires a value" >&2; exit 1; }; excludes+=("$2"); shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) echo "error: unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done

[[ -d "${source_directory}" ]] || { echo "error: Pages source directory not found: ${source_directory}" >&2; exit 1; }
[[ -n "${RELEASE_VERSION:-}" ]] || { echo "error: RELEASE_VERSION is required" >&2; exit 1; }
[[ -n "${RELEASE_ARTIFACT_DIR:-}" ]] || { echo "error: RELEASE_ARTIFACT_DIR is required" >&2; exit 1; }
staging_manifest="${RELEASE_ARTIFACT_DIR}/staging.json"
[[ -f "${staging_manifest}" ]] || { echo "error: release staging area is not initialized" >&2; exit 1; }
command -v rsync >/dev/null 2>&1 || { echo "error: rsync is required" >&2; exit 1; }

temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT
site_directory="${temporary_directory}/site"
mkdir -p "${site_directory}"
rsync_args=(-a)
for pattern in "${excludes[@]}"; do rsync_args+=(--exclude "${pattern}"); done
rsync "${rsync_args[@]}" "${source_directory}/" "${site_directory}/"
if find "${site_directory}" -type l -print -quit | grep -q .; then
  echo "error: Pages source must not contain symlinks" >&2
  exit 1
fi
: >"${site_directory}/.nojekyll"
if [[ -n "${domain}" ]]; then
  printf '%s\n' "${domain}" >"${site_directory}/CNAME"
fi
python3 - "${staging_manifest}" "${site_directory}/.mprlab-release.json" <<'PY'
import json
import pathlib
import sys

staging = json.load(open(sys.argv[1], encoding="utf-8"))
marker = {
    "schema_version": 1,
    "release_version": staging["version"],
    "source_commit": staging["source_commit"],
    "release_timestamp": staging["release_timestamp"],
}
pathlib.Path(sys.argv[2]).write_text(json.dumps(marker, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY

asset_directory="${RELEASE_ARTIFACT_DIR}/payloads/release-assets"
mkdir -p "${asset_directory}"
archive="${asset_directory}/pages.tar.gz"
rm -f "${archive}"
COPYFILE_DISABLE=1 tar -czf "${archive}" -C "${site_directory}" .
echo "Prepared ${archive}."
