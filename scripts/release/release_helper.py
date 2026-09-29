#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.10"
# dependencies = []
# ///
"""Deterministic helper for the Git Release skill."""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import re
import shutil
import subprocess
import tempfile
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


SEMVER_TAG_RE = re.compile(r"^v?(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)(?:[-+][0-9A-Za-z.-]+)?$")
CALVER_TAG_RE = re.compile(
    r"^v?(?P<year>[1-9]\d)\.(?P<month_day>(?:0|[1-9]\d*))\.(?P<hhmmss>(?:0|[1-9]\d*))$"
)
# Older releases used YYYY.M.D.minutes; keep recognizing them for ordering.
LEGACY_CALVER_MINUTE_TAG_RE = re.compile(
    r"^v?(?P<year>\d{4})\.(?P<month>\d{1,2})\.(?P<day>\d{1,2})\.(?P<minutes>\d{1,4})$"
)
# Older releases also used YYYY.M.D.H[.m[.s]]; keep recognizing them for ordering.
LEGACY_CALVER_TAG_RE = re.compile(
    r"^v?(?P<year>\d{4})\.(?P<month>\d{1,2})\.(?P<day>\d{1,2})"
    r"(?:\.(?P<hour>\d{1,2})(?:\.(?P<minute>\d{1,2})(?:\.(?P<second>\d{1,2}))?)?)?$"
)
RELEASE_HEADING_RE = re.compile(
    r"^##\s+\[?(?:v?(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)(?:[-+][0-9A-Za-z.-]+)?|v?\d{4}\.\d{1,2}\.\d{1,2}(?:\.\d{1,4}(?:\.\d{1,2}(?:\.\d{1,2})?)?)?)\]?(?:[^\n]*)?$",
    re.MULTILINE,
)


class HelperError(Exception):
    def __init__(self, message: str, details: dict[str, Any] | None = None) -> None:
        super().__init__(message)
        self.details = details or {}


def run(command: list[str], cwd: Path | None = None, check: bool = True) -> subprocess.CompletedProcess[str]:
    proc = subprocess.run(
        command,
        cwd=str(cwd) if cwd else None,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if check and proc.returncode != 0:
        raise HelperError(
            f"command failed: {' '.join(command)}",
            {
                "command": command,
                "returncode": proc.returncode,
                "stdout": proc.stdout.strip(),
                "stderr": proc.stderr.strip(),
            },
        )
    return proc


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def emit(payload: dict[str, Any]) -> None:
    print(json.dumps(payload, indent=2, sort_keys=True))


def fail(message: str, details: dict[str, Any] | None = None) -> None:
    emit({"ok": False, "error": message, "details": details or {}})
    raise SystemExit(1)


def require_tools(names: list[str]) -> list[str]:
    return [name for name in names if shutil.which(name) is None]


def repo_root() -> Path:
    return Path(run(["git", "rev-parse", "--show-toplevel"]).stdout.strip())


def gh_json(command: list[str], cwd: Path) -> Any:
    return json.loads(run(command, cwd=cwd).stdout or "null")


def resolve_default_branch(cwd: Path, override: str | None = None) -> str:
    if override:
        return override

    gh_proc = run(["gh", "repo", "view", "--json", "defaultBranchRef"], cwd=cwd, check=False)
    if gh_proc.returncode == 0:
        data = json.loads(gh_proc.stdout)
        name = (data.get("defaultBranchRef") or {}).get("name")
        if name:
            return name

    remote_proc = run(["git", "remote", "show", "origin"], cwd=cwd)
    for line in remote_proc.stdout.splitlines():
        if "HEAD branch:" in line:
            return line.rsplit(":", 1)[1].strip()

    raise HelperError("could not resolve default branch")


def resolve_default_branch_local(cwd: Path, override: str | None = None) -> str:
    if override:
        return override

    remote_head = run(
        ["git", "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"],
        cwd=cwd,
        check=False,
    )
    if remote_head.returncode != 0:
        raise HelperError(
            "could not resolve the default branch from local refs",
            {"required_ref": "refs/remotes/origin/HEAD"},
        )
    ref_name = remote_head.stdout.strip()
    if not ref_name.startswith("origin/") or ref_name == "origin/":
        raise HelperError("local origin/HEAD ref is invalid", {"ref": ref_name})
    return ref_name.removeprefix("origin/")


def all_tags(cwd: Path) -> list[str]:
    return run(["git", "tag", "--sort=-version:refname"], cwd=cwd).stdout.splitlines()


def calver_match(tag: str) -> re.Match[str] | None:
    match = CALVER_TAG_RE.match(tag)
    if not match:
        return None
    month_day = int(match.group("month_day"))
    month = month_day // 100
    day = month_day % 100
    time_value = int(match.group("hhmmss"))
    if time_value > 235959:
        return None
    hhmmss = f"{time_value:06d}"
    hour = int(hhmmss[:2])
    minute = int(hhmmss[2:4])
    second = int(hhmmss[4:6])
    try:
        dt.date(2000 + int(match.group("year")), month, day)
    except ValueError:
        return None
    if not (0 <= hour <= 23 and 0 <= minute <= 59 and 0 <= second <= 59):
        return None
    return match


def legacy_calver_minute_match(tag: str) -> re.Match[str] | None:
    match = LEGACY_CALVER_MINUTE_TAG_RE.match(tag)
    if not match:
        return None
    try:
        dt.date(int(match.group("year")), int(match.group("month")), int(match.group("day")))
    except ValueError:
        return None
    if not 0 <= int(match.group("minutes")) <= 1439:
        return None
    return match


def legacy_calver_match(tag: str) -> re.Match[str] | None:
    match = LEGACY_CALVER_TAG_RE.match(tag)
    if not match:
        return None
    try:
        dt.date(int(match.group("year")), int(match.group("month")), int(match.group("day")))
    except ValueError:
        return None
    for name, upper in (("hour", 23), ("minute", 59), ("second", 59)):
        value = match.group(name)
        if value is not None and not 0 <= int(value) <= upper:
            return None
    return match


def tag_scheme(tag: str) -> str | None:
    if calver_match(tag) or legacy_calver_minute_match(tag) or legacy_calver_match(tag):
        return "calver"
    if SEMVER_TAG_RE.match(tag):
        return "semver"
    return None


def parse_release_timestamp(value: str | None, release_date: str | None = None) -> dt.datetime:
    if not value:
        if release_date:
            return dt.datetime.combine(parse_release_date(release_date), dt.time())
        return dt.datetime.now().astimezone()
    try:
        normalized = value.replace("Z", "+00:00")
        return dt.datetime.fromisoformat(normalized)
    except ValueError as exc:
        try:
            return dt.datetime.combine(dt.date.fromisoformat(value), dt.time())
        except ValueError:
            raise HelperError(
                "release timestamp must use ISO format such as YYYY-MM-DDTHH:MM:SS or YYYY-MM-DD",
                {"release_timestamp": value},
            ) from exc


def parse_release_date(value: str) -> dt.date:
    try:
        return dt.date.fromisoformat(value)
    except ValueError as exc:
        raise HelperError("release date must use YYYY-MM-DD format", {"release_date": value}) from exc


def calver_sort_key(tag: str) -> tuple[int, int, int, int, int]:
    match = calver_match(tag)
    if match:
        month_day = int(match.group("month_day"))
        hhmmss = f"{int(match.group('hhmmss')):06d}"
        seconds = (int(hhmmss[:2]) * 3600) + (int(hhmmss[2:4]) * 60) + int(hhmmss[4:6])
        return (
            2000 + int(match.group("year")),
            month_day // 100,
            month_day % 100,
            seconds,
            0,
        )

    match = legacy_calver_minute_match(tag)
    if match:
        return (
            int(match.group("year")),
            int(match.group("month")),
            int(match.group("day")),
            int(match.group("minutes")) * 60,
            -1,
        )

    match = legacy_calver_match(tag)
    if not match:
        raise ValueError(f"not a CalVer tag: {tag}")
    hour = int(match.group("hour")) if match.group("hour") is not None else 0
    minute = int(match.group("minute")) if match.group("minute") is not None else 0
    second = int(match.group("second")) if match.group("second") is not None else 0
    precision_rank = 0 if match.group("second") is not None else -1
    return (
        int(match.group("year")),
        int(match.group("month")),
        int(match.group("day")),
        (hour * 3600) + (minute * 60) + second,
        precision_rank,
    )


def calver_year(timestamp: dt.datetime) -> int:
    year = timestamp.year % 100
    if not 10 <= year <= 99:
        raise HelperError(
            "CalVer YY component must be a two-digit SemVer-compatible integer",
            {"release_year": timestamp.year, "yy": year},
        )
    return year


def calver_month_day(timestamp: dt.datetime) -> int:
    return (timestamp.month * 100) + timestamp.day


def calver_hhmmss(timestamp: dt.datetime) -> int:
    return (timestamp.hour * 10000) + (timestamp.minute * 100) + timestamp.second


def calver_seconds_since_midnight(timestamp: dt.datetime) -> int:
    return (timestamp.hour * 3600) + (timestamp.minute * 60) + timestamp.second


def calver_from_timestamp(timestamp: dt.datetime) -> str:
    parts = [calver_year(timestamp), calver_month_day(timestamp), calver_hhmmss(timestamp)]
    return ".".join(str(part) for part in parts)


def calver_candidate(tags: list[str], release_timestamp: dt.datetime) -> dict[str, Any]:
    existing = set(tags)
    chosen = calver_from_timestamp(release_timestamp)
    collision_chain = [chosen] if chosen in existing or f"v{chosen}" in existing else []
    calver_tags = [tag for tag in tags if tag_scheme(tag) == "calver"]
    latest_calver = sorted(calver_tags, key=calver_sort_key, reverse=True)[0] if calver_tags else None
    candidate_key = calver_sort_key(chosen)
    latest_key = calver_sort_key(latest_calver) if latest_calver else None
    errors: list[str] = []
    if chosen in existing or f"v{chosen}" in existing:
        errors.append("CalVer second candidate already exists")
    if latest_key and candidate_key <= latest_key:
        errors.append("CalVer timestamp is not later than the latest CalVer tag")

    return {
        "ok": not errors,
        "candidate": chosen,
        "precision": "second",
        "year": calver_year(release_timestamp),
        "month_day": calver_month_day(release_timestamp),
        "hhmmss": calver_hhmmss(release_timestamp),
        "seconds_since_midnight": calver_seconds_since_midnight(release_timestamp),
        "release_timestamp": release_timestamp.isoformat(),
        "latest_calver_tag": latest_calver,
        "collision_chain": collision_chain,
        "errors": errors,
    }


def version_info(cwd: Path, release_timestamp: dt.datetime) -> dict[str, Any]:
    tags = all_tags(cwd)
    semver_tags = [tag for tag in tags if tag_scheme(tag) == "semver"]
    calver_tags = sorted((tag for tag in tags if tag_scheme(tag) == "calver"), key=calver_sort_key, reverse=True)
    version_tags = [tag for tag in tags if tag_scheme(tag)]
    calver = calver_candidate(tags, release_timestamp)

    if semver_tags and calver_tags:
        scheme_guess = "mixed"
    elif calver_tags:
        scheme_guess = "calver"
    elif semver_tags:
        scheme_guess = "semver"
    else:
        scheme_guess = "none"

    latest_by_guess = None
    if scheme_guess == "calver":
        latest_by_guess = calver_tags[0]
    elif scheme_guess == "semver":
        latest_by_guess = semver_tags[0]
    elif version_tags:
        latest_by_guess = version_tags[0]

    return {
        "scheme_guess": scheme_guess,
        "latest_tag": latest_by_guess,
        "latest_any_version_tag": version_tags[0] if version_tags else None,
        "latest_semver_tag": semver_tags[0] if semver_tags else None,
        "latest_calver_tag": calver_tags[0] if calver_tags else None,
        "version_tags": version_tags[:20],
        "next_calver": calver["candidate"],
        "calver_candidate": calver,
        "release_date": release_timestamp.date().isoformat(),
        "release_timestamp": release_timestamp.isoformat(),
        "calver_format": "YY.MDD.HHMMSS",
    }


def detect_validation_candidates(cwd: Path) -> list[str]:
    candidates: list[str] = []

    makefile = cwd / "Makefile"
    if makefile.exists() and re.search(r"^ci\s*:", makefile.read_text(encoding="utf-8", errors="replace"), re.MULTILINE):
        candidates.append("make ci")

    package_json = cwd / "package.json"
    if package_json.exists():
        try:
            scripts = json.loads(package_json.read_text(encoding="utf-8")).get("scripts", {})
        except json.JSONDecodeError:
            scripts = {}
        runner = "npm"
        if (cwd / "pnpm-lock.yaml").exists():
            runner = "pnpm"
        elif (cwd / "yarn.lock").exists():
            runner = "yarn"
        for script_name in ("ci", "test"):
            if script_name in scripts:
                candidates.append(f"{runner} run {script_name}")

    if (cwd / "pyproject.toml").exists() or (cwd / "pytest.ini").exists():
        if not candidates:
            candidates.append("pytest")

    return candidates


def command_preflight(args: argparse.Namespace) -> int:
    missing = require_tools(["git"] if args.local else ["git", "gh", "gix"])
    if missing:
        fail("required tools are missing", {"missing_tools": missing})

    cwd = repo_root()
    default_branch = (
        resolve_default_branch_local(cwd, args.default_branch)
        if args.local
        else resolve_default_branch(cwd, args.default_branch)
    )
    versions = version_info(cwd, parse_release_timestamp(args.release_timestamp, args.release_date))
    status_lines = run(["git", "status", "--short"], cwd=cwd).stdout.splitlines()
    current_branch = run(["git", "branch", "--show-current"], cwd=cwd).stdout.strip()
    open_prs = []
    if not args.local:
        open_prs = gh_json(
            ["gh", "pr", "list", "--base", default_branch, "--state", "open", "--json", "number,title,headRefName,url"],
            cwd,
        )
    payload = {
        "ok": not status_lines and not open_prs and current_branch == default_branch,
        "scope": "local" if args.local else "remote",
        "repo_root": str(cwd),
        "default_branch": default_branch,
        "current_branch": current_branch,
        "dirty_status": status_lines,
        "open_prs": open_prs,
        "latest_tag": versions["latest_tag"],
        "version_info": versions,
        "validation_candidates": detect_validation_candidates(cwd),
    }
    emit(payload)
    return 0 if payload["ok"] else 1


def command_generate_notes(args: argparse.Namespace) -> int:
    cwd = repo_root()
    release_date = parse_release_date(args.release_date).isoformat()
    revision = "HEAD"
    if args.since_tag:
        boundary = run(["git", "rev-parse", "--verify", f"{args.since_tag}^{{commit}}"], cwd=cwd, check=False)
        if boundary.returncode != 0:
            fail("changelog boundary tag does not resolve locally", {"since_tag": args.since_tag})
        revision = f"{args.since_tag}..HEAD"

    log_result = run(["git", "log", "--format=%s", revision], cwd=cwd)
    subjects = [line.strip() for line in log_result.stdout.splitlines() if line.strip()]
    if not subjects:
        fail("no local commits are available for release notes", {"revision": revision})

    print(f"## [{args.version}] - {release_date}")
    print()
    for subject in subjects:
        print(f"- {subject}")
    return 0


def release_artifact_dir(cwd: Path, override: str | None = None) -> Path:
    if override:
        return Path(override).expanduser().resolve()
    raw_path = run(["git", "rev-parse", "--git-path", "mprlab-release"], cwd=cwd).stdout.strip()
    artifact_path = Path(raw_path)
    if not artifact_path.is_absolute():
        artifact_path = cwd / artifact_path
    return artifact_path.resolve()


def resolve_commit(cwd: Path, revision: str, label: str) -> str:
    result = run(["git", "rev-parse", "--verify", f"{revision}^{{commit}}"], cwd=cwd, check=False)
    if result.returncode != 0:
        raise HelperError(f"{label} does not resolve to a commit", {label: revision})
    return result.stdout.strip()


def command_initialize_release_artifact(args: argparse.Namespace) -> int:
    cwd = repo_root()
    artifact_path = release_artifact_dir(cwd, args.artifact_dir)
    if artifact_path.exists():
        shutil.rmtree(artifact_path)
    (artifact_path / "payloads").mkdir(parents=True)
    staging = {
        "schema_version": 1,
        "artifact_kind": "mprlab.release.staging",
        "version": args.version,
        "source_commit": resolve_commit(cwd, args.source_commit, "source_commit"),
        "release_timestamp": parse_release_timestamp(args.release_timestamp).isoformat(),
    }
    (artifact_path / "staging.json").write_text(
        json.dumps(staging, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )
    emit({"ok": True, "artifact_dir": str(artifact_path), "staging": staging})
    return 0


def inventory_payloads(artifact_path: Path) -> list[dict[str, Any]]:
    payload_root = artifact_path / "payloads"
    if not payload_root.is_dir():
        return []

    payloads: list[dict[str, Any]] = []
    for path in sorted(payload_root.rglob("*")):
        if path.is_symlink():
            raise HelperError("prepared release payloads must not contain symlinks", {"path": str(path)})
        if not path.is_file():
            continue
        relative_path = path.relative_to(artifact_path).as_posix()
        payloads.append(
            {
                "path": relative_path,
                "size": path.stat().st_size,
                "sha256": sha256_file(path),
            }
        )
    return payloads


def verify_payloads(artifact_path: Path, payloads: Any) -> list[dict[str, Any]]:
    if not isinstance(payloads, list):
        raise HelperError("prepared release payload inventory is invalid")

    expected_paths: set[str] = set()
    verified: list[dict[str, Any]] = []
    for entry in payloads:
        if not isinstance(entry, dict):
            raise HelperError("prepared release payload entry is invalid", {"entry": entry})
        relative_path = entry.get("path")
        if not isinstance(relative_path, str) or not relative_path.startswith("payloads/"):
            raise HelperError("prepared release payload path is invalid", {"path": relative_path})
        path = (artifact_path / relative_path).resolve()
        if artifact_path not in path.parents or not path.is_file() or path.is_symlink():
            raise HelperError("prepared release payload is missing or unsafe", {"path": relative_path})
        actual_size = path.stat().st_size
        actual_sha256 = sha256_file(path)
        if entry.get("size") != actual_size or entry.get("sha256") != actual_sha256:
            raise HelperError(
                "prepared release payload does not match the manifest",
                {
                    "path": relative_path,
                    "expected_size": entry.get("size"),
                    "actual_size": actual_size,
                    "expected_sha256": entry.get("sha256"),
                    "actual_sha256": actual_sha256,
                },
            )
        if relative_path in expected_paths:
            raise HelperError("prepared release payload path is duplicated", {"path": relative_path})
        expected_paths.add(relative_path)
        verified.append(entry)

    actual_paths = {entry["path"] for entry in inventory_payloads(artifact_path)}
    if actual_paths != expected_paths:
        raise HelperError(
            "prepared release payload inventory is incomplete",
            {"expected_paths": sorted(expected_paths), "actual_paths": sorted(actual_paths)},
        )
    return verified


def command_write_release_artifact(args: argparse.Namespace) -> int:
    cwd = repo_root()
    release_commit = resolve_commit(cwd, args.release_commit, "release_commit")
    source_commit = resolve_commit(cwd, args.source_commit, "source_commit")
    head_commit = resolve_commit(cwd, "HEAD", "head")
    tag_commit = resolve_commit(cwd, args.version, "version")
    if release_commit != head_commit:
        fail("release commit must be HEAD", {"release_commit": release_commit, "head": head_commit})
    if tag_commit != release_commit:
        fail("local release tag must point at the release commit", {"version": args.version, "tag_commit": tag_commit})

    parent_result = run(["git", "rev-parse", "--verify", f"{release_commit}^"], cwd=cwd, check=False)
    parent_commit = parent_result.stdout.strip() if parent_result.returncode == 0 else ""
    if parent_commit != source_commit:
        fail(
            "release commit must directly follow the prepared source commit",
            {"source_commit": source_commit, "release_parent": parent_commit},
        )

    changed_files = run(
        ["git", "diff-tree", "--no-commit-id", "--name-only", "-r", release_commit], cwd=cwd
    ).stdout.splitlines()
    if changed_files != ["CHANGELOG.md"]:
        fail("release commit must contain only CHANGELOG.md", {"changed_files": changed_files})

    notes_source = Path(args.notes_file)
    notes = notes_source.read_text(encoding="utf-8").strip()
    if not notes:
        fail("release notes file is empty", {"notes_file": str(notes_source)})

    artifact_path = release_artifact_dir(cwd, args.artifact_dir)
    staging_path = artifact_path / "staging.json"
    if not staging_path.is_file():
        fail("prepared release staging area is missing", {"artifact_dir": str(artifact_path)})
    staging = json.loads(staging_path.read_text(encoding="utf-8"))
    expected_staging = {
        "artifact_kind": "mprlab.release.staging",
        "version": args.version,
        "source_commit": source_commit,
    }
    for key, expected_value in expected_staging.items():
        if staging.get(key) != expected_value:
            fail(
                "prepared release staging area does not match the release",
                {"field": key, "expected": expected_value, "actual": staging.get(key)},
            )

    notes_path = artifact_path / "notes.md"
    notes_path.write_text(notes + "\n", encoding="utf-8")
    payloads = inventory_payloads(artifact_path)
    manifest = {
        "schema_version": 2,
        "artifact_kind": "mprlab.release",
        "version": args.version,
        "source_commit": source_commit,
        "release_commit": release_commit,
        "default_branch": args.default_branch,
        "release_timestamp": parse_release_timestamp(args.release_timestamp).isoformat(),
        "notes_sha256": sha256_file(notes_path),
        "payloads": payloads,
    }
    manifest_path = artifact_path / "manifest.json"
    manifest_path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    staging_path.unlink()
    emit({"ok": True, "artifact_dir": str(artifact_path), "manifest": manifest})
    return 0


def load_release_artifact(cwd: Path, override: str | None = None) -> tuple[Path, dict[str, Any], Path]:
    artifact_path = release_artifact_dir(cwd, override)
    manifest_path = artifact_path / "manifest.json"
    notes_path = artifact_path / "notes.md"
    if not manifest_path.is_file() or not notes_path.is_file():
        raise HelperError(
            "prepared release artifact is missing; run make release",
            {"artifact_dir": str(artifact_path)},
        )
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    if manifest.get("schema_version") != 2 or manifest.get("artifact_kind") != "mprlab.release":
        raise HelperError("prepared release manifest has an invalid contract", {"manifest": str(manifest_path)})
    actual_notes_sha256 = sha256_file(notes_path)
    if manifest.get("notes_sha256") != actual_notes_sha256:
        raise HelperError(
            "prepared release notes do not match the manifest",
            {"expected": manifest.get("notes_sha256"), "actual": actual_notes_sha256},
        )
    verify_payloads(artifact_path, manifest.get("payloads"))
    return artifact_path, manifest, notes_path


def command_verify_release_artifact(args: argparse.Namespace) -> int:
    cwd = repo_root()
    artifact_path, manifest, _ = load_release_artifact(cwd, args.artifact_dir)
    emit({"ok": True, "artifact_dir": str(artifact_path), "manifest": manifest})
    return 0


def release_asset_paths(artifact_path: Path, manifest: dict[str, Any]) -> list[Path]:
    prefix = "payloads/release-assets/"
    assets = [artifact_path / "manifest.json"] + [
        artifact_path / entry["path"]
        for entry in manifest.get("payloads", [])
        if entry["path"].startswith(prefix)
    ]
    names = [path.name for path in assets]
    if len(names) != len(set(names)):
        raise HelperError("GitHub Release asset names must be unique", {"asset_names": names})
    return assets


def publish_release_assets(cwd: Path, version: str, assets: list[Path]) -> list[dict[str, Any]]:
    if not assets:
        return []

    run(["gh", "release", "upload", version, *[str(path) for path in assets], "--clobber"], cwd=cwd)
    published: list[dict[str, Any]] = []
    with tempfile.TemporaryDirectory(prefix="mprlab-release-assets-") as temporary_directory:
        download_root = Path(temporary_directory)
        for asset in assets:
            asset_dir = download_root / asset.name
            asset_dir.mkdir()
            run(
                ["gh", "release", "download", version, "--pattern", asset.name, "--dir", str(asset_dir)],
                cwd=cwd,
            )
            downloaded = asset_dir / asset.name
            expected_sha256 = sha256_file(asset)
            actual_sha256 = sha256_file(downloaded)
            if actual_sha256 != expected_sha256:
                raise HelperError(
                    "published GitHub Release asset does not match the prepared payload",
                    {
                        "asset": asset.name,
                        "expected_sha256": expected_sha256,
                        "actual_sha256": actual_sha256,
                    },
                )
            published.append(
                {
                    "name": asset.name,
                    "sha256": actual_sha256,
                    "size": downloaded.stat().st_size,
                }
            )
    return published


def command_publish_prepared_release(args: argparse.Namespace) -> int:
    missing = require_tools(["git", "gh"])
    if missing:
        fail("required tools are missing", {"missing_tools": missing})

    cwd = repo_root()
    artifact_path, manifest, notes_path = load_release_artifact(cwd, args.artifact_dir)
    version = str(manifest.get("version") or "")
    default_branch = str(manifest.get("default_branch") or "")
    release_commit = str(manifest.get("release_commit") or "")
    source_commit = str(manifest.get("source_commit") or "")
    release_assets = release_asset_paths(artifact_path, manifest)
    if not all((version, default_branch, release_commit, source_commit)):
        fail("prepared release manifest is incomplete", {"artifact_dir": str(artifact_path)})

    current_branch = run(["git", "branch", "--show-current"], cwd=cwd).stdout.strip()
    dirty_status = run(["git", "status", "--short"], cwd=cwd).stdout.splitlines()
    head_commit = resolve_commit(cwd, "HEAD", "head")
    tag_commit = resolve_commit(cwd, version, "version")
    errors: list[str] = []
    if current_branch != default_branch:
        errors.append(f"current branch is {current_branch or '<detached>'}; expected {default_branch}")
    if dirty_status:
        errors.append("worktree is dirty")
    if head_commit != release_commit:
        errors.append("HEAD does not match the prepared release commit")
    if tag_commit != release_commit:
        errors.append("local release tag does not match the prepared release commit")
    if errors:
        fail("prepared release is not publishable", {"errors": errors, "dirty_status": dirty_status})

    remote_ref = f"refs/remotes/{args.remote}/{default_branch}"
    run(
        [
            "git",
            "fetch",
            "--prune",
            args.remote,
            f"+refs/heads/{default_branch}:{remote_ref}",
        ],
        cwd=cwd,
    )
    remote_branch_commit = resolve_commit(cwd, remote_ref, "remote_branch")
    if remote_branch_commit not in (source_commit, release_commit):
        fail(
            "remote default branch changed after make release; reconcile and run make release again",
            {
                "remote_branch": remote_branch_commit,
                "prepared_source_commit": source_commit,
                "prepared_release_commit": release_commit,
            },
        )

    open_prs = gh_json(
        ["gh", "pr", "list", "--base", default_branch, "--state", "open", "--json", "number,title,headRefName,url"],
        cwd,
    )
    if open_prs:
        fail("open pull requests target the default branch", {"open_prs": open_prs})

    remote_tag_commit = ls_remote_tag_commit(cwd, version)
    if remote_tag_commit and remote_tag_commit != release_commit:
        fail(
            "remote release tag points at a different commit",
            {"version": version, "remote_tag_commit": remote_tag_commit, "release_commit": release_commit},
        )

    plan = {
        "push_branch": remote_branch_commit != release_commit,
        "push_tag": not remote_tag_commit,
        "publish_github_release": True,
        "release_assets": [path.name for path in release_assets],
    }
    if args.dry_run:
        emit(
            {
                "ok": True,
                "dry_run": True,
                "artifact_dir": str(artifact_path),
                "version": version,
                "release_commit": release_commit,
                "remote": args.remote,
                "plan": plan,
            }
        )
        return 0

    if plan["push_branch"]:
        run(["git", "push", args.remote, f"HEAD:refs/heads/{default_branch}"], cwd=cwd)
    if plan["push_tag"]:
        run(["git", "push", args.remote, f"refs/tags/{version}:refs/tags/{version}"], cwd=cwd)

    publish_args = argparse.Namespace(version=version, notes_file=str(notes_path), title=None)
    if command_publish_release(publish_args) != 0:
        return 1
    published_assets = publish_release_assets(cwd, version, release_assets)
    verify_args = argparse.Namespace(
        version=version,
        release_commit=release_commit,
        notes_file=str(notes_path),
        default_branch=default_branch,
        watch_run=[],
        skip_pages=True,
        expect_pages_text=[],
    )
    verify_result = command_verify_release(verify_args)
    if verify_result == 0 and published_assets:
        emit({"ok": True, "published_release_assets": published_assets})
    return verify_result


def normalize_markdown(text: str) -> str:
    return "\n".join(line.rstrip() for line in text.strip().splitlines()).strip()


def command_insert_changelog(args: argparse.Namespace) -> int:
    cwd = repo_root()
    notes_path = Path(args.notes_file)
    notes = notes_path.read_text(encoding="utf-8").strip()
    if not notes:
        fail("release notes file is empty", {"notes_file": str(notes_path)})

    changelog = cwd / args.changelog
    if changelog.exists():
        existing = changelog.read_text(encoding="utf-8")
    else:
        existing = "# Changelog\n\n"

    first_heading = next((line.strip() for line in notes.splitlines() if line.startswith("## ")), None)
    if first_heading and re.search(rf"^{re.escape(first_heading)}$", existing, re.MULTILINE):
        if normalize_markdown(notes) in normalize_markdown(existing):
            emit({"ok": True, "changed": False, "changelog": str(changelog), "reason": "release notes already present"})
            return 0
        fail("changelog already contains a matching release heading with different content", {"heading": first_heading})

    section = notes.rstrip() + "\n\n"
    match = RELEASE_HEADING_RE.search(existing)
    if match:
        updated = existing[: match.start()] + section + existing[match.start() :]
    else:
        h1 = re.search(r"^# .*$", existing, re.MULTILINE)
        if h1:
            insert_at = h1.end()
            while insert_at < len(existing) and existing[insert_at] == "\n":
                insert_at += 1
            updated = existing[:insert_at].rstrip() + "\n\n" + section + existing[insert_at:].lstrip()
        else:
            updated = section + existing.lstrip()

    changelog.write_text(updated, encoding="utf-8")
    emit({"ok": True, "changed": updated != existing, "changelog": str(changelog)})
    return 0


def command_publish_release(args: argparse.Namespace) -> int:
    missing = require_tools(["git", "gh"])
    if missing:
        fail("required tools are missing", {"missing_tools": missing})

    cwd = repo_root()
    notes_path = Path(args.notes_file)
    expected_notes = normalize_markdown(notes_path.read_text(encoding="utf-8"))
    if not expected_notes:
        fail("release notes file is empty", {"notes_file": str(notes_path)})

    title = args.title or f"Release {args.version}"
    view_command = [
        "gh",
        "release",
        "view",
        args.version,
        "--json",
        "tagName,name,body,publishedAt,isDraft,isPrerelease,targetCommitish,url",
    ]
    existing_proc = run(view_command, cwd=cwd, check=False)
    action = "none"
    command: list[str] | None = None

    if existing_proc.returncode != 0:
        action = "created"
        command = [
            "gh",
            "release",
            "create",
            args.version,
            "--verify-tag",
            "--title",
            title,
            "--notes-file",
            str(notes_path),
            "--latest",
        ]
    else:
        existing = json.loads(existing_proc.stdout)
        actual_notes = normalize_markdown(existing.get("body") or "")
        needs_edit = (
            existing.get("tagName") != args.version
            or existing.get("name") != title
            or existing.get("isDraft")
            or actual_notes != expected_notes
        )
        if needs_edit:
            action = "updated"
            command = [
                "gh",
                "release",
                "edit",
                args.version,
                "--verify-tag",
                "--title",
                title,
                "--notes-file",
                str(notes_path),
                "--draft=false",
                "--latest",
            ]

    if command:
        run(command, cwd=cwd)

    refreshed = gh_json(view_command, cwd)
    errors: list[str] = []
    if refreshed.get("tagName") != args.version:
        errors.append("GitHub Release object has the wrong tagName")
    if refreshed.get("isDraft"):
        errors.append("GitHub Release object is still a draft")
    if not refreshed.get("publishedAt"):
        errors.append("GitHub Release object has no publishedAt timestamp")
    if normalize_markdown(refreshed.get("body") or "") != expected_notes:
        errors.append("GitHub Release body does not match generated release notes")

    payload = {"ok": not errors, "action": action, "release": refreshed, "errors": errors}
    emit(payload)
    return 0 if not errors else 1


def ls_remote_tag_commit(cwd: Path, version: str) -> str:
    peeled = run(["git", "ls-remote", "--tags", "origin", f"refs/tags/{version}^{{}}"], cwd=cwd).stdout.strip()
    if peeled:
        return peeled.split()[0]
    direct = run(["git", "ls-remote", "--tags", "origin", f"refs/tags/{version}"], cwd=cwd).stdout.strip()
    return direct.split()[0] if direct else ""


def fetch_url(url: str, head_only: bool = False) -> dict[str, Any]:
    method = "HEAD" if head_only else "GET"
    request = urllib.request.Request(url, method=method, headers={"User-Agent": "gitrelease-helper/1"})
    try:
        with urllib.request.urlopen(request, timeout=20) as response:
            body = "" if head_only else response.read(1_000_000).decode("utf-8", errors="replace")
            return {"ok": True, "status": response.status, "url": response.geturl(), "body": body}
    except urllib.error.HTTPError as exc:
        return {"ok": False, "status": exc.code, "error": str(exc), "url": url}
    except urllib.error.URLError as exc:
        return {"ok": False, "error": str(exc), "url": url}


def optional_gh_json(command: list[str], cwd: Path) -> dict[str, Any]:
    proc = run(command, cwd=cwd, check=False)
    if proc.returncode == 0:
        return {"ok": True, "data": json.loads(proc.stdout or "null")}
    return {"ok": False, "returncode": proc.returncode, "stderr": proc.stderr.strip(), "stdout": proc.stdout.strip()}


def collect_pages(cwd: Path, expected_texts: list[str]) -> tuple[dict[str, Any], list[str]]:
    errors: list[str] = []
    pages = optional_gh_json(["gh", "api", "repos/{owner}/{repo}/pages"], cwd)
    if not pages["ok"]:
        stderr = pages.get("stderr", "")
        if "404" in stderr or "Not Found" in stderr:
            return {"configured": False, "lookup": pages}, []
        errors.append("GitHub Pages configuration lookup failed")
        return {"configured": None, "lookup": pages}, errors

    data = pages["data"] or {}
    html_url = data.get("html_url")
    result: dict[str, Any] = {
        "configured": True,
        "config": data,
        "latest_build": optional_gh_json(
            ["gh", "api", "repos/{owner}/{repo}/pages/builds/latest", "--jq", "{status,error,commit,created_at,updated_at,url}"],
            cwd,
        ),
        "latest_deployment": optional_gh_json(
            [
                "gh",
                "api",
                "repos/{owner}/{repo}/deployments?environment=github-pages",
                "--jq",
                ".[0] | {id,sha,ref,created_at,statuses_url}",
            ],
            cwd,
        ),
    }
    if not html_url:
        errors.append("GitHub Pages is configured but has no html_url")
        return result, errors

    head = fetch_url(html_url, head_only=True)
    if not head["ok"]:
        head = fetch_url(html_url, head_only=False)
    result["site_probe"] = {key: value for key, value in head.items() if key != "body"}
    if not head["ok"]:
        errors.append("GitHub Pages URL is not reachable")

    if expected_texts:
        page = fetch_url(html_url, head_only=False)
        body = page.get("body", "") if page["ok"] else ""
        missing = [text for text in expected_texts if text not in body]
        result["expected_text_check"] = {"ok": not missing, "missing": missing}
        if missing:
            errors.append("GitHub Pages URL does not contain expected release text")

    return result, errors


def collect_runs(cwd: Path, default_branch: str, release_commit: str) -> dict[str, Any]:
    return {
        "for_release_commit": optional_gh_json(
            [
                "gh",
                "run",
                "list",
                "--commit",
                release_commit,
                "--json",
                "databaseId,name,event,status,conclusion,headSha,url",
                "--limit",
                "20",
            ],
            cwd,
        ),
        "release_events": optional_gh_json(
            [
                "gh",
                "run",
                "list",
                "--event",
                "release",
                "--json",
                "databaseId,name,event,status,conclusion,headSha,url",
                "--limit",
                "20",
            ],
            cwd,
        ),
        "default_branch_push_events": optional_gh_json(
            [
                "gh",
                "run",
                "list",
                "--event",
                "push",
                "--branch",
                default_branch,
                "--json",
                "databaseId,name,event,status,conclusion,headSha,url",
                "--limit",
                "20",
            ],
            cwd,
        ),
    }


def command_verify_release(args: argparse.Namespace) -> int:
    missing = require_tools(["git", "gh"])
    if missing:
        fail("required tools are missing", {"missing_tools": missing})

    cwd = repo_root()
    default_branch = resolve_default_branch(cwd, args.default_branch)
    release_commit = run(["git", "rev-parse", args.release_commit], cwd=cwd).stdout.strip()
    errors: list[str] = []

    local_tag_proc = run(["git", "rev-list", "-n", "1", args.version], cwd=cwd, check=False)
    local_tag_commit = local_tag_proc.stdout.strip() if local_tag_proc.returncode == 0 else ""
    remote_tag_commit = ls_remote_tag_commit(cwd, args.version)
    if local_tag_commit != release_commit:
        errors.append("local tag does not point at release commit")
    if remote_tag_commit != release_commit:
        errors.append("remote tag does not point at release commit")

    release_proc = run(
        [
            "gh",
            "release",
            "view",
            args.version,
            "--json",
            "tagName,name,body,publishedAt,isDraft,isPrerelease,targetCommitish,url",
        ],
        cwd=cwd,
        check=False,
    )
    release: dict[str, Any] | None = None
    if release_proc.returncode != 0:
        errors.append("GitHub Release object is missing or unreadable")
    else:
        release = json.loads(release_proc.stdout)
        if release.get("tagName") != args.version:
            errors.append("GitHub Release object has the wrong tagName")
        if release.get("isDraft"):
            errors.append("GitHub Release object is still a draft")
        if not release.get("publishedAt"):
            errors.append("GitHub Release object has no publishedAt timestamp")
        if args.notes_file:
            expected_notes = normalize_markdown(Path(args.notes_file).read_text(encoding="utf-8"))
            actual_notes = normalize_markdown(release.get("body") or "")
            if expected_notes != actual_notes:
                errors.append("GitHub Release body does not match generated release notes")

    watched_runs: list[dict[str, Any]] = []
    for run_id in args.watch_run:
        proc = run(["gh", "run", "watch", str(run_id), "--exit-status"], cwd=cwd, check=False)
        watched_runs.append(
            {
                "run_id": run_id,
                "returncode": proc.returncode,
                "stdout": proc.stdout.strip(),
                "stderr": proc.stderr.strip(),
            }
        )
        if proc.returncode != 0:
            errors.append(f"watched GitHub Actions run failed or did not complete: {run_id}")

    pages, page_errors = ({"skipped": True}, [])
    if not args.skip_pages:
        pages, page_errors = collect_pages(cwd, args.expect_pages_text)
        errors.extend(page_errors)

    payload = {
        "ok": not errors,
        "repo_root": str(cwd),
        "default_branch": default_branch,
        "version": args.version,
        "release_commit": release_commit,
        "local_tag_commit": local_tag_commit,
        "remote_tag_commit": remote_tag_commit,
        "release": release,
        "runs": collect_runs(cwd, default_branch, release_commit),
        "watched_runs": watched_runs,
        "pages": pages,
        "final_status": run(["git", "status", "--short"], cwd=cwd).stdout.splitlines(),
        "errors": errors,
    }
    emit(payload)
    return 0 if not errors else 1


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Deterministic helper for the Git Release skill.")
    subparsers = parser.add_subparsers(dest="command", required=True)

    preflight = subparsers.add_parser("preflight", help="Check deterministic release preconditions.")
    preflight.add_argument("--default-branch")
    preflight.add_argument("--release-date", help="Release date in YYYY-MM-DD format. Used as midnight if no timestamp is provided.")
    preflight.add_argument(
        "--release-timestamp",
        help="Release timestamp in ISO format for CalVer candidate generation, for example 2026-04-29T06:17:41 -> 26.429.61741.",
    )
    preflight.add_argument(
        "--local",
        action="store_true",
        help="Use only local Git state. Do not query GitHub or a remote repository.",
    )
    preflight.set_defaults(func=command_preflight)

    notes = subparsers.add_parser("generate-notes", help="Generate deterministic release notes from local Git history.")
    notes.add_argument("--version", required=True)
    notes.add_argument("--release-date", required=True)
    notes.add_argument("--since-tag")
    notes.set_defaults(func=command_generate_notes)

    changelog = subparsers.add_parser("insert-changelog", help="Insert generated release notes into CHANGELOG.md.")
    changelog.add_argument("--notes-file", required=True)
    changelog.add_argument("--changelog", default="CHANGELOG.md")
    changelog.set_defaults(func=command_insert_changelog)

    publish = subparsers.add_parser("publish-release", help="Create or update the GitHub Release object.")
    publish.add_argument("--version", required=True)
    publish.add_argument("--notes-file", required=True)
    publish.add_argument("--title")
    publish.set_defaults(func=command_publish_release)

    initialize_artifact = subparsers.add_parser(
        "initialize-release-artifact",
        help="Create an empty local staging area for release payloads.",
    )
    initialize_artifact.add_argument("--version", required=True)
    initialize_artifact.add_argument("--source-commit", required=True)
    initialize_artifact.add_argument("--release-timestamp", required=True)
    initialize_artifact.add_argument("--artifact-dir")
    initialize_artifact.set_defaults(func=command_initialize_release_artifact)

    artifact = subparsers.add_parser(
        "write-release-artifact",
        help="Write the prepared local release manifest and notes under the repository Git directory.",
    )
    artifact.add_argument("--version", required=True)
    artifact.add_argument("--source-commit", required=True)
    artifact.add_argument("--release-commit", required=True)
    artifact.add_argument("--notes-file", required=True)
    artifact.add_argument("--default-branch", required=True)
    artifact.add_argument("--release-timestamp", required=True)
    artifact.add_argument("--artifact-dir")
    artifact.set_defaults(func=command_write_release_artifact)

    verify_artifact = subparsers.add_parser(
        "verify-release-artifact",
        help="Verify the prepared local release manifest, notes, and payload hashes.",
    )
    verify_artifact.add_argument("--artifact-dir")
    verify_artifact.set_defaults(func=command_verify_release_artifact)

    publish_prepared = subparsers.add_parser(
        "publish-prepared-release",
        help="Push the prepared branch/tag and publish the matching GitHub Release object.",
    )
    publish_prepared.add_argument("--remote", default="origin")
    publish_prepared.add_argument("--artifact-dir")
    publish_prepared.add_argument("--dry-run", action="store_true")
    publish_prepared.set_defaults(func=command_publish_prepared_release)

    verify = subparsers.add_parser("verify-release", help="Verify remote tag, GitHub Release, runs, and Pages.")
    verify.add_argument("--version", required=True)
    verify.add_argument("--release-commit", required=True)
    verify.add_argument("--notes-file")
    verify.add_argument("--default-branch")
    verify.add_argument("--watch-run", action="append", default=[])
    verify.add_argument("--skip-pages", action="store_true")
    verify.add_argument("--expect-pages-text", action="append", default=[])
    verify.set_defaults(func=command_verify_release)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    try:
        return args.func(args)
    except HelperError as exc:
        fail(str(exc), exc.details)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
