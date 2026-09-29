#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
release_scripts="${repository_root}/scripts/release"
release_helper="${release_scripts}/release_helper.py"
prepare_release="${release_scripts}/prepare_release.sh"
prepare_pages="${release_scripts}/prepare_pages_artifact.sh"
deploy_pages="${release_scripts}/deploy_pages_artifact.sh"
release_version="v1.2.0"
temporary_root="$(mktemp -d)"
trap 'rm -rf "${temporary_root}"' EXIT

remote_repository="${temporary_root}/origin.git"
fixture_repository="${temporary_root}/repository"
git init --bare "${remote_repository}" >/dev/null
git clone "${remote_repository}" "${fixture_repository}" >/dev/null
git -C "${fixture_repository}" config user.name "Pages Release Test"
git -C "${fixture_repository}" config user.email "pages-release-test@example.invalid"
mkdir -p "${fixture_repository}/site"
printf '%s\n' '<!doctype html><title>Pages release fixture</title>' >"${fixture_repository}/site/index.html"
cat >"${fixture_repository}/Makefile" <<MAKEFILE
ci:
	@true

pages-artifact:
	@"${prepare_pages}" --source site
MAKEFILE
git -C "${fixture_repository}" add Makefile site/index.html
git -C "${fixture_repository}" commit -m "Add Pages fixture" >/dev/null
git -C "${fixture_repository}" branch -M master
git -C "${fixture_repository}" push -u origin master >/dev/null
git --git-dir="${remote_repository}" symbolic-ref HEAD refs/heads/master
git -C "${fixture_repository}" remote set-head origin -a >/dev/null

(
  cd "${fixture_repository}"
  RELEASE_HELPER="${release_helper}" \
    RELEASE_ARTIFACT_TARGETS=pages-artifact \
    "${prepare_release}" --version "${release_version}" >/dev/null
)

source_commit="$(git -C "${fixture_repository}" rev-parse HEAD^)"
release_commit="$(git -C "${fixture_repository}" rev-parse HEAD)"
[[ "${source_commit}" != "${release_commit}" ]] || {
  echo "error: release preparation did not preserve distinct source and release commits" >&2
  exit 1
}

artifact_directory="$(git -C "${fixture_repository}" rev-parse --git-path mprlab-release)"
[[ "${artifact_directory}" == /* ]] || artifact_directory="${fixture_repository}/${artifact_directory}"
manifest_path="${artifact_directory}/manifest.json"
archive_path="${artifact_directory}/payloads/release-assets/pages.tar.gz"
public_marker_path="${temporary_root}/public-marker.json"

python3 - "${manifest_path}" "${archive_path}" "${public_marker_path}" "${release_version}" "${source_commit}" "${release_commit}" <<'PYTHON'
import json
import pathlib
import sys
import tarfile

manifest_path, archive_path, marker_path, version, source_commit, release_commit = sys.argv[1:]
manifest = json.loads(pathlib.Path(manifest_path).read_text(encoding="utf-8"))
if manifest["source_commit"] != source_commit:
    raise SystemExit("release manifest has the wrong source commit")
if manifest["release_commit"] != release_commit:
    raise SystemExit("release manifest has the wrong release commit")

with tarfile.open(archive_path, "r:gz") as pages_archive:
    members = {
        member.name.removeprefix("./"): member
        for member in pages_archive.getmembers()
    }
    nojekyll_member = members.get(".nojekyll")
    if nojekyll_member is None or nojekyll_member.size != 0:
        raise SystemExit("Pages archive must contain an empty .nojekyll")
    marker_member = members.get(".mprlab-release.json")
    if marker_member is None:
        raise SystemExit("Pages archive has no release marker")
    marker_stream = pages_archive.extractfile(marker_member)
    if marker_stream is None:
        raise SystemExit("Pages release marker is unreadable")
    marker = json.load(marker_stream)

if marker.get("release_version") != version:
    raise SystemExit("Pages release marker has the wrong version")
if marker.get("source_commit") != source_commit:
    raise SystemExit("Pages release marker has the wrong source commit")
pathlib.Path(marker_path).write_text(json.dumps(marker), encoding="utf-8")
PYTHON

git -C "${fixture_repository}" push origin HEAD:refs/heads/master >/dev/null
git -C "${fixture_repository}" push origin "refs/tags/${release_version}:refs/tags/${release_version}" >/dev/null
fake_bin="${temporary_root}/fake-bin"
mkdir -p "${fake_bin}"
cat >"${fake_bin}/gh" <<'SHELL'
#!/bin/sh
set -eu
destination=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--dir" ]; then
    destination="$2"
    shift 2
  else
    shift
  fi
done
[ -n "$destination" ]
cp "$FAKE_RELEASE_DIR/manifest.json" "$destination/manifest.json"
cp "$FAKE_RELEASE_DIR/payloads/release-assets/pages.tar.gz" "$destination/pages.tar.gz"
SHELL
cat >"${fake_bin}/curl" <<'SHELL'
#!/bin/sh
set -eu
cat "$FAKE_PAGES_MARKER"
SHELL
chmod +x "${fake_bin}/gh" "${fake_bin}/curl"

deploy_environment_path="${fake_bin}:${PATH}"
accepted_output="$(
  cd "${fixture_repository}"
  PATH="${deploy_environment_path}" \
    FAKE_RELEASE_DIR="${artifact_directory}" \
    FAKE_PAGES_MARKER="${public_marker_path}" \
    PAGES_VERIFY_ATTEMPTS=1 \
    PAGES_VERIFY_DELAY_SECONDS=0 \
    "${deploy_pages}" \
      --version "${release_version}" \
      --url https://pages.example.invalid \
      --skip-configure
)"
[[ "${accepted_output}" == *"Verified https://pages.example.invalid at source ${source_commit}."* ]] || {
  echo "error: public Pages marker for the source commit was not accepted" >&2
  exit 1
}
[[ "${accepted_output}" != *"${release_commit}"* ]] || {
  echo "error: deployment reported the release commit as Pages source" >&2
  exit 1
}

python3 - "${public_marker_path}" "${release_commit}" <<'PYTHON'
import json
import pathlib
import sys

marker_path, release_commit = sys.argv[1:]
marker_file = pathlib.Path(marker_path)
marker = json.loads(marker_file.read_text(encoding="utf-8"))
marker["source_commit"] = release_commit
marker_file.write_text(json.dumps(marker), encoding="utf-8")
PYTHON

set +e
rejected_output="$(
  cd "${fixture_repository}"
  PATH="${deploy_environment_path}" \
    FAKE_RELEASE_DIR="${artifact_directory}" \
    FAKE_PAGES_MARKER="${public_marker_path}" \
    PAGES_VERIFY_ATTEMPTS=1 \
    PAGES_VERIFY_DELAY_SECONDS=0 \
    "${deploy_pages}" \
      --version "${release_version}" \
      --url https://pages.example.invalid \
      --skip-configure 2>&1
)"
rejected_status=$?
set -e
[[ "${rejected_status}" -ne 0 ]] || {
  echo "error: public Pages marker for the release commit was accepted" >&2
  exit 1
}
[[ "${rejected_output}" == *"source ${source_commit}"* ]] || {
  echo "error: release-commit marker rejection did not name the expected source" >&2
  exit 1
}

echo "Repository-owned Pages release contract passed."

