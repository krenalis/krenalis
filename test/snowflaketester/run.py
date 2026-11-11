# Copyright 2026 Open2b. All rights reserved.
# Use of this source code is governed by an Elastic License 2.0
# that can be found in the LICENSE file.

"""Run a Go test block with redacted output and source provenance; never source .env files."""

import argparse
import collections
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys


SCHEMA_NAME = r"KRENALIS_TEST_SCHEMA_[0-9]+_[0-9a-f]{32}"
SCHEMA_EVENT = re.compile(r"Snowflake test schema (created|dropped) schema=(" + SCHEMA_NAME + r")\b", re.IGNORECASE)


def audit_schemas(directory):
    paths = sorted(directory.glob("*.manifest.json"))
    created, dropped = collections.Counter(), collections.Counter()
    issues = [] if paths else ["no manifests found"]
    for path in paths:
        try:
            manifest = json.loads(path.read_text())
        except (OSError, ValueError):
            issues.append(path.name + ": unreadable manifest")
            continue
        schemas = manifest.get("schemas") if isinstance(manifest, dict) else None
        if not isinstance(schemas, dict) or type(schemas.get("required")) is not bool:
            issues.append(path.name + ": missing schema evidence")
            continue
        if manifest.get("state") != "complete":
            issues.append(path.name + ": incomplete run")
        for action, total in [("created", created), ("dropped", dropped)]:
            records = schemas.get(action)
            if not isinstance(records, dict) or any(not isinstance(name, str) or
                    re.fullmatch(SCHEMA_NAME, name, re.IGNORECASE) is None or
                    type(count) is not int or count <= 0 for name, count in records.items()):
                issues.append(path.name + ": invalid schema evidence")
                continue
            for name, count in records.items():
                total[name.upper()] += count
        if schemas["required"] and not schemas.get("created"):
            issues.append(path.name + ": required schema records are absent")
    remaining, unmatched_drops = dict(created - dropped), dict(dropped - created)
    print(json.dumps({"manifests": len(paths), "schemas_created": sum(created.values()),
                      "schemas_dropped": sum(dropped.values()), "schemas_without_drop_record": remaining,
                      "drops_without_create_record": unmatched_drops, "issues": issues}))
    return int(bool(issues or remaining or unmatched_drops))


def source_snapshot(root, arguments):
    def git(*args):
        return subprocess.check_output(["git", *args], cwd=root).decode()

    tracked = git("ls-files", "-z").split("\0")
    changed = git("diff", "--name-only", "-z", "HEAD").split("\0")
    untracked = git("ls-files", "--others", "--exclude-standard", "-z").split("\0")
    files = {name for name in tracked if name.endswith("_test.go")}
    files.update({"go.mod", "go.sum", "vendor/modules.txt", "test/snowflaketester/run.py"})
    for name in changed + untracked:
        if name.endswith(".go") or name.startswith(("core/", "cmd/", "warehouses/", "connectors/", "test/", "tools/", "vendor/")):
            if Path(name).suffix in (".go", ".sql", ".json", ".txt", ".py", ".s", ".h", ".c", ".md", ".csv"):
                files.add(name)
    hashes = {}
    for name in sorted(files):
        path = root / name
        if path.is_file():
            with path.open("rb") as source:
                hashes[name] = hashlib.file_digest(source, "sha256").hexdigest()
        else:
            hashes[name] = None
    overlays = {}
    for i, argument in enumerate(arguments):
        if argument in ("-overlay", "--overlay"):
            if i + 1 == len(arguments):
                raise ValueError("-overlay requires a file")
            path = Path(arguments[i + 1])
        elif argument.startswith(("-overlay=", "--overlay=")):
            path = Path(argument.split("=", 1)[1])
        else:
            continue
        overlay = path.read_bytes()
        replacements = json.loads(overlay).get("Replace", {})
        overlays[str(path)] = {"sha256": hashlib.sha256(overlay).hexdigest(), "replacements": {
            source: {"path": target, "sha256": hashlib.sha256(Path(target).read_bytes()).hexdigest()
                     if target else None} for source, target in replacements.items()}}
    return {"head": git("rev-parse", "HEAD").strip(), "status": git("status", "--porcelain"),
            "files_sha256": hashes, "overlays": overlays}


def write_manifest(path, value):
    temporary = path.with_suffix(".tmp")
    with os.fdopen(os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), "w") as output:
        json.dump(value, output, indent=2)
        output.write("\n")
    temporary.replace(path)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--audit", action="store_true", help="audit schema evidence in a manifest directory")
    parser.add_argument("--require-schemas", action="store_true", help="require schema creation and complete cleanup")
    parser.add_argument("path", type=Path)
    parser.add_argument("go_args", nargs=argparse.REMAINDER)
    options = parser.parse_args()
    arguments = options.go_args
    if options.audit:
        if arguments or options.require_schemas:
            parser.error("--audit takes only a manifest directory")
        return audit_schemas(options.path)
    if arguments[:1] == ["--"]:
        arguments = arguments[1:]
    if not arguments:
        parser.error("a Go test package and arguments are required")
    root = Path(__file__).resolve().parents[2]
    if Path.cwd() != root:
        parser.error("run from the repository root")
    log_path = options.path.resolve()
    if log_path.is_relative_to(root):
        parser.error("write evidence outside the checkout")
    manifest_path = log_path.with_suffix(".manifest.json")
    if log_path.exists() or manifest_path.exists():
        parser.error("use a fresh log name; existing evidence is never overwritten")
    command = ["go", "test", *arguments, "-mod=vendor", "-parallel=1", "-p=1", "-failfast", "-count=1", "-json"]
    version = subprocess.check_output(["go", "version"], text=True).strip()
    driver = next(line for line in (root / "vendor/modules.txt").read_text().splitlines()
                  if line.startswith("# github.com/snowflakedb/gosnowflake/v2 v"))
    masks = set()
    short_value_keys = []
    for key, value in os.environ.items():
        if key.startswith("KRENALIS_SNOWFLAKE_TESTER_") and value:
            escaped = json.dumps(value)[1:-1]
            masks.update((value, escaped, escaped.replace("&", r"\u0026").replace("<", r"\u003c").replace(">", r"\u003e")))
            if key.endswith("_ACCOUNT"):
                masks.add(value.replace(".", "-"))
            if len(value) <= 4:
                short_value_keys.append(key)
    mask = re.compile("|".join(re.escape(value) for value in sorted(masks, key=len, reverse=True)),
                      re.IGNORECASE) if masks else None
    manifest = {"started_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                "command": command, "go_version": version, "driver": driver,
                "before": source_snapshot(root, arguments), "state": "running",
                "schemas": {"required": options.require_schemas or
                            os.environ.get("KRENALIS_SKIP_SNOWFLAKE_TESTS") == "false",
                            "created": {}, "dropped": {}},
                "redaction": {"case_insensitive": True, "output_only": True,
                              "short_value_keys": sorted(short_value_keys)}}
    log_path.parent.mkdir(parents=True, exist_ok=True)
    write_manifest(manifest_path, manifest)
    counts = collections.Counter()
    with os.fdopen(os.open(log_path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), "w") as log:
        process = subprocess.Popen(command, cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                   text=True, errors="replace")
        for line in process.stdout:
            try:
                event = json.loads(line)
            except json.JSONDecodeError:
                event = None
            if isinstance(event, dict) and "Action" in event:
                if event.get("Test") and event.get("Action") in ("pass", "skip", "fail"):
                    counts[event["Action"]] += 1
                output = event.get("Output")
                if isinstance(output, str):
                    records = SCHEMA_EVENT.findall(output)
                    for action, name in records:
                        names = manifest["schemas"][action.lower()]
                        name = name.upper()
                        names[name] = names.get(name, 0) + 1
                    if records:
                        write_manifest(manifest_path, manifest)
                    if mask is not None:
                        event["Output"] = mask.sub("<SNOWFLAKE_TEST_VALUE>", output)
                line = json.dumps(event) + "\n"
            elif mask is not None:
                line = mask.sub("<SNOWFLAKE_TEST_VALUE>", line)
            log.write(line)
            log.flush()
        code = process.wait()
    manifest.update(state="complete", go_exit=code, counts=dict(counts),
                    after=source_snapshot(root, arguments),
                    finished_at=datetime.datetime.now(datetime.timezone.utc).isoformat())
    manifest["source_stable"] = all(manifest["before"][key] == manifest["after"][key]
                                    for key in ("head", "files_sha256", "overlays"))
    if code == 0 and not manifest["source_stable"]:
        code = 2
    if code == 0 and (counts["skip"] or not counts["pass"]):
        code = 3
    schemas = manifest["schemas"]
    if code == 0 and (schemas["created"] != schemas["dropped"] or
                      schemas["required"] and not schemas["created"]):
        code = 4
    manifest["exit"] = code
    write_manifest(manifest_path, manifest)
    print(json.dumps({"log": str(log_path), "manifest": str(manifest_path), "exit": manifest["exit"],
                      "source_stable": manifest["source_stable"], "counts": dict(counts)}))
    return manifest["exit"]


if __name__ == "__main__":
    sys.exit(main())
