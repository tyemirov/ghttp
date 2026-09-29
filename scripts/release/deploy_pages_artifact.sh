#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  deploy_pages_artifact.sh --url <public-url> [options]

Downloads manifest.json and pages.tar.gz from a published GitHub Release,
verifies them against the remote tag, and replaces the configured Pages branch.

Options:
  --remote <name>       Git remote. Default: origin
  --branch <name>       Pages branch. Default: gh-pages
  --version <tag>       Published release tag. Default: exact v* tag at HEAD
  --url <url>           Public Pages URL used for post-deploy verification
  --skip-configure      Do not create/update the Pages branch source setting
  --skip-verify         Do not verify the public release marker
  --help                Show this help text
USAGE
}

remote="origin"
branch="gh-pages"
version=""
url=""
configure="true"
verify="true"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --remote) [[ $# -ge 2 ]] || { echo "error: --remote requires a value" >&2; exit 1; }; remote="$2"; shift 2 ;;
    --branch) [[ $# -ge 2 ]] || { echo "error: --branch requires a value" >&2; exit 1; }; branch="$2"; shift 2 ;;
    --version) [[ $# -ge 2 ]] || { echo "error: --version requires a value" >&2; exit 1; }; version="$2"; shift 2 ;;
    --url) [[ $# -ge 2 ]] || { echo "error: --url requires a value" >&2; exit 1; }; url="$2"; shift 2 ;;
    --skip-configure) configure="false"; shift ;;
    --skip-verify) verify="false"; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "error: unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done

[[ -n "${url}" || "${verify}" == "false" ]] || { echo "error: --url is required unless --skip-verify is set" >&2; exit 1; }
for command_name in git gh python3 tar; do command -v "${command_name}" >/dev/null 2>&1 || { echo "error: ${command_name} is required" >&2; exit 1; }; done

repo_root="$(git rev-parse --show-toplevel)"
if [[ -z "${version}" ]]; then
  version="$(git tag --points-at HEAD --list 'v*' --sort=-version:refname | head -n 1)"
fi
[[ -n "${version}" ]] || { echo "error: no exact release tag at HEAD; pass --version after make publish" >&2; exit 1; }

temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT
download_directory="${temporary_directory}/download"
site_directory="${temporary_directory}/site"
checkout_directory="${temporary_directory}/checkout"
mkdir -p "${download_directory}" "${site_directory}"
gh release download "${version}" --pattern manifest.json --pattern pages.tar.gz --dir "${download_directory}"
archive="${download_directory}/pages.tar.gz"
readarray -t release_values < <(python3 - "${download_directory}/manifest.json" "${version}" <<'PY'
import json
import sys

manifest = json.load(open(sys.argv[1], encoding="utf-8"))
if manifest.get("schema_version") != 2 or manifest.get("artifact_kind") != "mprlab.release":
    raise SystemExit("published release manifest has an invalid contract")
if manifest.get("version") != sys.argv[2]:
    raise SystemExit("published release manifest has the wrong version")
asset = next((item for item in manifest["payloads"] if item["path"] == "payloads/release-assets/pages.tar.gz"), None)
if asset is None:
    raise SystemExit("published release has no Pages payload; run make release and make publish")
print(manifest["release_commit"])
print(manifest["source_commit"])
print(asset["sha256"])
PY
)
release_commit="${release_values[0]}"
source_commit="${release_values[1]}"
expected_sha256="${release_values[2]}"
remote_tag_commit="$(git ls-remote --tags "${remote}" "refs/tags/${version}^{}" | awk 'NR == 1 {print $1}')"
if [[ -z "${remote_tag_commit}" ]]; then
  remote_tag_commit="$(git ls-remote --tags "${remote}" "refs/tags/${version}" | awk 'NR == 1 {print $1}')"
fi
[[ "${remote_tag_commit}" == "${release_commit}" ]] || { echo "error: published release manifest does not match remote tag ${version}" >&2; exit 1; }
actual_sha256="$(shasum -a 256 "${archive}" | awk '{print $1}')"
[[ "${actual_sha256}" == "${expected_sha256}" ]] || { echo "error: published Pages asset does not match make release" >&2; exit 1; }
python3 - "${archive}" <<'PY'
import pathlib
import sys
import tarfile

with tarfile.open(sys.argv[1], "r:gz") as archive:
    for member in archive.getmembers():
        path = pathlib.PurePosixPath(member.name)
        if path.is_absolute() or ".." in path.parts or member.issym() or member.islnk():
            raise SystemExit(f"unsafe Pages archive member: {member.name}")
PY
tar -xzf "${archive}" -C "${site_directory}"
python3 - "${site_directory}/.mprlab-release.json" "${version}" "${source_commit}" <<'PY'
import json
import sys

marker_path, version, source_commit = sys.argv[1:]
try:
    marker = json.load(open(marker_path, encoding="utf-8"))
except (OSError, json.JSONDecodeError):
    raise SystemExit(f"published Pages marker is invalid for source {source_commit}")
if marker.get("schema_version") != 1:
    raise SystemExit(f"published Pages marker has an invalid schema for source {source_commit}")
if marker.get("release_version") != version:
    raise SystemExit(f"published Pages marker has the wrong version for source {source_commit}")
if marker.get("source_commit") != source_commit:
    raise SystemExit(f"published Pages marker has the wrong source; expected source {source_commit}")
PY

remote_url="$(git remote get-url "${remote}")"
git clone --no-checkout "${remote_url}" "${checkout_directory}" >/dev/null
if git -C "${checkout_directory}" show-ref --verify --quiet "refs/remotes/origin/${branch}"; then
  git -C "${checkout_directory}" checkout -B "${branch}" "origin/${branch}" >/dev/null
else
  git -C "${checkout_directory}" checkout --orphan "${branch}" >/dev/null
fi
find "${checkout_directory}" -mindepth 1 -maxdepth 1 ! -name .git -exec rm -rf {} +
cp -R "${site_directory}"/. "${checkout_directory}/"
git -C "${checkout_directory}" add -A
if ! git -C "${checkout_directory}" diff --cached --quiet; then
  git -C "${checkout_directory}" -c user.name="MPR Lab Pages Deployer" -c user.email="pages-deployer@mprlab.invalid" commit -m "Deploy Pages for ${version}" >/dev/null
  git -C "${checkout_directory}" push origin "HEAD:refs/heads/${branch}"
else
  echo "Pages branch already contains ${version} from source ${source_commit}."
fi

if [[ "${configure}" == "true" ]]; then
  if gh api repos/{owner}/{repo}/pages >/dev/null 2>&1; then
    gh api --method PUT repos/{owner}/{repo}/pages -f build_type=legacy -f "source[branch]=${branch}" -f 'source[path]=/' -F https_enforced=true >/dev/null
  else
    gh api --method POST repos/{owner}/{repo}/pages -f build_type=legacy -f "source[branch]=${branch}" -f 'source[path]=/' -F https_enforced=true >/dev/null
  fi
  gh api --method POST repos/{owner}/{repo}/pages/builds >/dev/null
fi

if [[ "${verify}" == "true" ]]; then
  marker_url="${url%/}/.mprlab-release.json"
  attempts="${PAGES_VERIFY_ATTEMPTS:-12}"
  delay_seconds="${PAGES_VERIFY_DELAY_SECONDS:-5}"
  for ((attempt = 1; attempt <= attempts; attempt += 1)); do
    marker="$(curl --fail --silent --show-error "${marker_url}" 2>/dev/null || true)"
    if python3 - "${version}" "${source_commit}" "${marker}" >/dev/null 2>&1 <<'PY'
import json
import sys

data = json.loads(sys.argv[3])
if data.get("schema_version") != 1:
    raise SystemExit(1)
if data.get("release_version") != sys.argv[1]:
    raise SystemExit(1)
if data.get("source_commit") != sys.argv[2]:
    raise SystemExit(1)
PY
    then
      echo "Verified ${url} at source ${source_commit}."
      exit 0
    fi
    sleep "${delay_seconds}"
  done
  echo "error: Pages marker did not reach source ${source_commit}: ${marker_url}" >&2
  exit 1
fi

echo "Deployed Pages release ${version} from source ${source_commit}."
