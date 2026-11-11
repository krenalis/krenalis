# Copyright 2026 Open2b. All rights reserved.
# Use of this source code is governed by an Elastic License 2.0
# that can be found in the LICENSE file.

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
RUNNER = Path(__file__).with_name("run.py")
PACKAGE = "github.com/krenalis/krenalis/warehouses/snowflake"
SCHEMA = "KRENALIS_TEST_SCHEMA_123_" + "abcdef01" * 4
SESSION = ("Snowflake test session parameters timezone={timezone} timestamp_type_mapping=TIMESTAMP_NTZ "
           "timestamp_input_format=AUTO statement_timeout_in_seconds=60\n")


class RunTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="krenalis-run-test-")
        self.addCleanup(temporary.cleanup)
        self.directory = Path(temporary.name)
        binary = self.directory / "bin"
        binary.mkdir()
        go = binary / "go"
        go.write_text(r'''#!/usr/bin/env python3
import json, os, pathlib, sys
if sys.argv[1:] == ['version']:
    print('go version synthetic-test')
    raise SystemExit(0)
if os.environ.get('FAKE_MUTATE'):
    path = pathlib.Path(os.environ['FAKE_MUTATE'])
    path.write_text(path.read_text() + '// changed during test\n')
if os.environ.get('FAKE_OUTPUT_BYTES'):
    length = int(os.environ['FAKE_OUTPUT_BYTES']) - 1
    for offset in range(0, length, 1024):
        print(json.dumps({'Action': 'output', 'Output': 'x' * min(1024, length - offset)}))
    print(json.dumps({'Action': 'output', 'Output': '\n'}))
for event in json.loads(os.environ['FAKE_EVENTS']):
    if isinstance(event, str):
        print(event)
    else:
        text = json.dumps(event).replace('&', r'\u0026').replace('<', r'\u003c').replace('>', r'\u003e')
        print(text)
raise SystemExit(int(os.environ.get('FAKE_EXIT', '0')))
''')
        go.chmod(0o700)
        self.environment = {key: value for key, value in os.environ.items()
                            if not key.startswith("KRENALIS_SNOWFLAKE_TESTER_")}
        self.environment.update(PATH=str(binary) + os.pathsep + os.environ["PATH"],
                                KRENALIS_SKIP_SNOWFLAKE_TESTS="true", PYTHONDONTWRITEBYTECODE="1")
        self.number = 0

    def audit(self, directory):
        result = subprocess.run([sys.executable, str(RUNNER), "--audit", str(directory)],
                                cwd=ROOT, env=self.environment, capture_output=True, text=True)
        return result, json.loads(result.stdout)

    def block(self, events, *, values=None, arguments=None, require_schemas=False, go_exit=0, mutate=None, output_bytes=None, require_session_timezone=None):
        self.number += 1
        path = self.directory / str(self.number) / "run.jsonl"
        environment = dict(self.environment, FAKE_EVENTS=json.dumps(events), FAKE_EXIT=str(go_exit))
        if values:
            environment.update({"KRENALIS_SNOWFLAKE_TESTER_" + key: value for key, value in values.items()})
        if mutate:
            environment["FAKE_MUTATE"] = str(mutate)
        if output_bytes is not None:
            environment["FAKE_OUTPUT_BYTES"] = str(output_bytes)
        command = [sys.executable, str(RUNNER)]
        if require_schemas:
            command.append("--require-schemas")
        if require_session_timezone is not None:
            command.extend(["--require-session-timezone", require_session_timezone])
        command.extend([str(path), "--", *(arguments or ["./warehouses/snowflake"])])
        result = subprocess.run(command, cwd=ROOT, env=environment, capture_output=True, text=True)
        self.assertEqual(result.stderr, "")
        manifest = json.loads(path.with_suffix(".manifest.json").read_text())
        return result, path, manifest

    def event(self, action, output=None):
        event = {"Action": action, "Package": PACKAGE, "Test": "TestSynthetic", "Time": "2026-11-11T00:00:00Z"}
        if output is not None:
            event["Output"] = output
        return event

    def test_go_json_escaped_password(self):
        password = "fake&pw<x>"
        result, path, manifest = self.block(
            [self.event("output", "password: " + password + "\n"), self.event("pass")],
            values={"PASSWORD": password},
        )
        self.assertEqual(result.returncode, 0)
        self.assertEqual(json.loads(path.read_text().splitlines()[0])["Output"],
                         "password: <SNOWFLAKE_TEST_VALUE>\n")
        self.assertNotIn(password, path.read_text())
        self.assertNotIn(r"fake\u0026pw\u003cx\u003e", path.read_text())
        self.assertEqual(manifest["counts"], {"pass": 1})

    def test_fragmented_output_masks_complete_value(self):
        password = "fake&pw<x>"
        output = "x" * 4090 + password + " tail\n"
        events = [self.event("output", output[offset:offset + 1024]) for offset in range(0, len(output), 1024)]
        events[1]["Time"] = "2026-11-11T00:00:01Z"
        result, path, manifest = self.block([*events, self.event("pass")], values={"PASSWORD": password})
        self.assertEqual(result.returncode, 0)
        decoded = [json.loads(line) for line in path.read_text().splitlines()]
        self.assertEqual(len(decoded), 2)
        self.assertEqual(decoded[0], dict(events[0], Output="x" * 4090 + "<SNOWFLAKE_TEST_VALUE> tail\n"))
        self.assertFalse(manifest["output_limit_exceeded"])

    def test_fragmented_schema_records_are_tracked_before_redaction(self):
        events = []
        for action in ["created", "dropped"]:
            output = "x" * 4090 + "Snowflake test schema " + action + " schema=" + SCHEMA + "\n"
            events.extend(self.event("output", output[offset:offset + 1024])
                          for offset in range(0, len(output), 1024))
        result, path, manifest = self.block([*events, self.event("pass")],
                                            values={"DATABASE": "krenalis", "ROLE": "test"}, require_schemas=True)
        self.assertEqual(result.returncode, 0)
        self.assertEqual(len(path.read_text().splitlines()), 3)
        self.assertEqual(manifest["schemas"], {"required": True, "created": {SCHEMA.upper(): 1},
                                              "dropped": {SCHEMA.upper(): 1}})
        audit, evidence = self.audit(path.parent)
        self.assertEqual(audit.returncode, 0)
        self.assertEqual((evidence["schemas_created"], evidence["schemas_dropped"]), (1, 1))

    def test_unterminated_output_flushes_at_eof_and_before_other_events(self):
        events = [self.event("output", "fake&"), self.event("output", "pw<x>")]
        for following in [[], [self.event("pass")], ["other line", self.event("pass")]]:
            with self.subTest(following=following):
                result, path, _ = self.block(events + following, values={"PASSWORD": "fake&pw<x>"})
                self.assertEqual(result.returncode, 0 if following else 3)
                lines = path.read_text().splitlines()
                self.assertEqual(json.loads(lines[0])["Output"], "<SNOWFLAKE_TEST_VALUE>")
                self.assertEqual(len(lines), 1 + len(following))
                if following and isinstance(following[0], str):
                    self.assertEqual(lines[1], following[0])

    def test_fragment_keys_keep_output_streams_separate(self):
        for field in ["Package", "ImportPath", "Test", "Action"]:
            with self.subTest(field=field):
                first = self.event("output", "first")
                second = dict(self.event("output", "second\n"), **{field: "other"})
                result, path, _ = self.block([first, second, self.event("pass")])
                self.assertEqual(result.returncode, 0)
                self.assertEqual([json.loads(line) for line in path.read_text().splitlines()],
                                 [first, second, self.event("pass")])

    def test_output_reassembly_limit_fails_without_interrupting_cleanup(self):
        limit = 4 * 1024 * 1024
        events = [self.event("output", "Snowflake test schema created schema=" + SCHEMA + "\n"),
                  self.event("output", "Snowflake test schema dropped schema=" + SCHEMA + "\n"),
                  self.event("pass")]
        for length, expected in [(limit, 0), (limit + 1, 5)]:
            with self.subTest(length=length):
                result, path, manifest = self.block(events, output_bytes=length, require_schemas=True)
                self.assertEqual(result.returncode, expected)
                self.assertEqual(manifest["exit"], expected)
                self.assertEqual(manifest["go_exit"], 0)
                self.assertEqual(manifest["state"], "complete")
                self.assertEqual(manifest["output_limit_exceeded"], length > limit)
                self.assertEqual(manifest["redaction"]["max_output_bytes"], limit)
                self.assertEqual(manifest["schemas"]["created"], {SCHEMA.upper(): 1})
                self.assertEqual(manifest["schemas"]["dropped"], {SCHEMA.upper(): 1})
                decoded = [json.loads(line) for line in path.read_text().splitlines()]
                self.assertEqual(len(decoded), 4)
                if length > limit:
                    self.assertEqual(decoded[0]["Output"], "<OUTPUT_EXCEEDS_LIMIT>\n")
                    self.assertLess(path.stat().st_size, 4096)
                else:
                    self.assertEqual(len(decoded[0]["Output"].encode("utf-8")), limit)
                audit, evidence = self.audit(path.parent)
                self.assertEqual(audit.returncode, int(length > limit))
                self.assertEqual((evidence["schemas_created"], evidence["schemas_dropped"]), (1, 1))

    def test_identifier_case_and_account_spelling(self):
        values = {"USER": "fake_user", "DATABASE": "fake_db", "ROLE": "fake_role",
                  "WAREHOUSE": "fake_wh", "ACCOUNT": "fake.org"}
        output = "FAKE_USER fake_DB FAKE_ROLE fake_WH FAKE.ORG fake-org\n"
        result, path, _ = self.block([self.event("output", output), self.event("pass")], values=values)
        self.assertEqual(result.returncode, 0)
        decoded = json.loads(path.read_text().splitlines()[0])["Output"]
        self.assertEqual(decoded, " ".join(["<SNOWFLAKE_TEST_VALUE>"] * 6) + "\n")

    def test_metadata_and_schema_tracking_survive_common_values(self):
        events = [self.event("output", "Snowflake test schema created schema=" + SCHEMA + "\n"),
                  self.event("output", "Snowflake test schema dropped schema=" + SCHEMA.upper() + "\n"),
                  self.event("pass")]
        result, path, manifest = self.block(events, values={"DATABASE": "krenalis", "ROLE": "test"},
                                            require_schemas=True)
        self.assertEqual(result.returncode, 0)
        decoded = [json.loads(line) for line in path.read_text().splitlines()]
        for original, redacted in zip(events, decoded):
            self.assertEqual({key: value for key, value in redacted.items() if key != "Output"},
                             {key: value for key, value in original.items() if key != "Output"})
        self.assertNotIn("Snowflake test schema created", decoded[0]["Output"])
        self.assertEqual(manifest["schemas"], {"required": True, "created": {SCHEMA.upper(): 1},
                                              "dropped": {SCHEMA.upper(): 1}})
        self.assertEqual(manifest["redaction"]["short_value_keys"], ["KRENALIS_SNOWFLAKE_TESTER_ROLE"])
        audit, evidence = self.audit(path.parent)
        self.assertEqual(audit.returncode, 0)
        self.assertEqual((evidence["schemas_created"], evidence["schemas_dropped"]), (1, 1))
        self.assertEqual(evidence["schemas_without_drop_record"], {})

    def test_non_json_output(self):
        result, path, _ = self.block(["FAKE_USER: fake&pw<x>", r"password: fake\u0026pw\u003cx\u003e",
                                     self.event("pass")],
                                     values={"USER": "fake_user", "PASSWORD": "fake&pw<x>"})
        self.assertEqual(result.returncode, 0)
        self.assertEqual(path.read_text().splitlines()[0], "<SNOWFLAKE_TEST_VALUE>: <SNOWFLAKE_TEST_VALUE>")
        self.assertEqual(path.read_text().splitlines()[1], "password: <SNOWFLAKE_TEST_VALUE>")

    def test_process_failures_skips_and_empty_selections(self):
        for events, go_exit, expected in [
            ([self.event("fail")], 1, 1),
            ([self.event("skip"), self.event("pass")], 0, 3),
            ([], 0, 3),
        ]:
            with self.subTest(expected=expected, events=events):
                result, _, manifest = self.block(events, go_exit=go_exit)
                self.assertEqual(result.returncode, expected)
                self.assertEqual(manifest["exit"], expected)
                self.assertEqual(manifest["go_exit"], go_exit)

    def test_schema_evidence_is_required_for_remote_blocks(self):
        for output, expected_created in [(None, 0), ("Snowflake test schema created schema=" + SCHEMA, 1)]:
            with self.subTest(created=expected_created):
                events = [self.event("pass")]
                if output:
                    events.insert(0, self.event("output", output))
                result, path, _ = self.block(events, require_schemas=True)
                self.assertEqual(result.returncode, 4)
                audit, evidence = self.audit(path.parent)
                self.assertEqual(audit.returncode, 1)
                self.assertEqual(evidence["schemas_created"], expected_created)
                self.assertEqual(evidence["schemas_dropped"], 0)

    def test_remote_environment_requires_schema_records(self):
        self.environment["KRENALIS_SKIP_SNOWFLAKE_TESTS"] = "false"
        result, _, manifest = self.block([self.event("pass")])
        self.assertEqual(result.returncode, 4)
        self.assertTrue(manifest["schemas"]["required"])

    def test_session_timezone_is_recorded_before_redaction(self):
        for timezone in ["UTC", "America/Los_Angeles"]:
            with self.subTest(timezone=timezone):
                record = SESSION.format(timezone=timezone)
                output = "x" * 4090 + record
                events = [self.event("output", output[offset:offset + 1024])
                          for offset in range(0, len(output), 1024)]
                result, path, manifest = self.block([*events, self.event("output", record), self.event("pass")],
                    values={"DATABASE": timezone, "ROLE": "test"}, require_session_timezone=timezone)
                self.assertEqual(result.returncode, 0)
                self.assertEqual(manifest["sessions"], {"required_timezone": timezone,
                    "verified": {timezone: 2}, "invalid_records": 0})
                self.assertNotIn("timezone=" + timezone, path.read_text())
                self.assertEqual(manifest["counts"], {"pass": 1})

    def test_session_timezone_requirement_rejects_missing_or_different_records(self):
        utc, other = SESSION.format(timezone="UTC"), SESSION.format(timezone="America/Los_Angeles")
        unknown = SESSION.format(timezone="sensitive" * 1024)
        for outputs, verified, invalid in [
            ([], {}, 0),
            ([utc], {}, 0),  # A raw non-event line is not verification evidence.
            ([self.event("output", other)], {"America/Los_Angeles": 1}, 0),
            ([self.event("output", utc + other)], {"UTC": 1, "America/Los_Angeles": 1}, 0),
            ([self.event("output", utc + SESSION.format(timezone="Europe/Rome"))], {"UTC": 1}, 1),
            ([self.event("output", utc + unknown)], {"UTC": 1}, 1),
            ([self.event("output", utc + utc.replace("=60", "=600"))], {"UTC": 1}, 1),
            ([self.event("output", "Snowflake test session parameters\n")], {}, 1),
        ]:
            with self.subTest(verified=verified, invalid=invalid):
                result, _, manifest = self.block([*outputs, self.event("pass")], require_session_timezone="UTC")
                self.assertEqual(result.returncode, 6)
                self.assertEqual(manifest["go_exit"], 0)
                self.assertEqual(manifest["exit"], 6)
                self.assertEqual(manifest["sessions"], {"required_timezone": "UTC", "verified": verified,
                                                       "invalid_records": invalid})
                self.assertNotIn("sensitive" * 1024, json.dumps(manifest))

    def test_session_timezone_collection_is_optional(self):
        for output, verified in [("no session records\n", {}),
                                 (SESSION.format(timezone="UTC") + SESSION.format(timezone="America/Los_Angeles"),
                                  {"UTC": 1, "America/Los_Angeles": 1})]:
            with self.subTest(verified=verified):
                result, _, manifest = self.block([self.event("output", output), self.event("pass")])
                self.assertEqual(result.returncode, 0)
                self.assertEqual(manifest["sessions"], {"required_timezone": None, "verified": verified,
                                                       "invalid_records": 0})

    def test_session_timezone_preserves_existing_failure_codes(self):
        for events, options, expected in [
            ([self.event("fail")], {"go_exit": 1}, 1),
            ([self.event("skip"), self.event("pass")], {}, 3),
            ([self.event("pass")], {"require_schemas": True}, 4),
            ([self.event("pass")], {"output_bytes": 4 * 1024 * 1024 + 1}, 5),
        ]:
            with self.subTest(expected=expected):
                result, _, manifest = self.block(events, require_session_timezone="UTC", **options)
                self.assertEqual(result.returncode, expected)
                self.assertEqual(manifest["exit"], expected)
                self.assertEqual(manifest["sessions"]["verified"], {})

    def test_session_timezone_options_reject_invalid_uses(self):
        for options in [["--require-session-timezone", "Europe/Rome"],
                        ["--audit", "--require-session-timezone", "UTC"]]:
            with self.subTest(options=options):
                path = self.directory / "invalid.jsonl"
                result = subprocess.run([sys.executable, str(RUNNER), *options, str(path)], cwd=ROOT,
                                        env=self.environment, capture_output=True, text=True)
                self.assertEqual(result.returncode, 2)
                self.assertFalse(path.exists())
                self.assertFalse(path.with_suffix(".manifest.json").exists())

    def test_audit_rejects_absent_or_old_evidence(self):
        directory = self.directory / "audit"
        directory.mkdir()
        result, evidence = self.audit(directory)
        self.assertEqual(result.returncode, 1)
        self.assertEqual(evidence["manifests"], 0)
        (directory / "old.manifest.json").write_text(json.dumps({"state": "complete", "counts": {"pass": 1}}))
        result, evidence = self.audit(directory)
        self.assertEqual(result.returncode, 1)
        self.assertEqual(evidence["schemas_created"], 0)
        self.assertTrue(evidence["issues"])

    def test_audit_rejects_incomplete_or_invalid_evidence(self):
        directory = self.directory / "audit"
        directory.mkdir()
        path = directory / "run.manifest.json"
        for manifest in [
            {"state": "running", "schemas": {"required": True, "created": {SCHEMA: 1}, "dropped": {SCHEMA: 1}}},
            {"state": "complete", "schemas": {"required": True, "created": {SCHEMA: -1}, "dropped": {}}},
            {"state": "complete", "schemas": {"required": True, "created": {"unexpected": 1}, "dropped": {}}},
            {"state": "complete", "schemas": {"required": True, "created": {}, "dropped": {SCHEMA: 1}}},
        ]:
            with self.subTest(manifest=manifest):
                path.write_text(json.dumps(manifest))
                result, _ = self.audit(directory)
                self.assertEqual(result.returncode, 1)
        path.write_text("invalid JSON")
        result, evidence = self.audit(directory)
        self.assertEqual(result.returncode, 1)
        self.assertEqual((evidence["schemas_created"], evidence["schemas_dropped"]), (0, 0))

    def test_all_overlay_spellings_and_changed_replacements(self):
        replacement = self.directory / "replacement.go"
        replacement.write_text("package snowflake\n")
        overlay = self.directory / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {str(ROOT / "warehouses/snowflake/profile_filter.go"):
                                                   str(replacement)}}))
        for flag in ["-overlay", "--overlay", "-overlay=", "--overlay="]:
            with self.subTest(flag=flag):
                arguments = [flag + str(overlay)] if flag.endswith("=") else [flag, str(overlay)]
                arguments.append("./warehouses/snowflake")
                result, _, manifest = self.block([self.event("pass")], arguments=arguments)
                self.assertEqual(result.returncode, 0)
                self.assertIn(str(overlay), manifest["before"]["overlays"])
        result, _, manifest = self.block([self.event("pass")],
                                         arguments=["--overlay", str(overlay), "./warehouses/snowflake"],
                                         mutate=replacement)
        self.assertEqual(result.returncode, 2)
        self.assertEqual(manifest["go_exit"], 0)
        self.assertFalse(manifest["source_stable"])

    def test_evidence_paths_and_permissions(self):
        result, path, _ = self.block([self.event("pass")])
        self.assertEqual(result.returncode, 0)
        manifest_path = path.with_suffix(".manifest.json")
        self.assertEqual(path.stat().st_mode & 0o777, 0o600)
        self.assertEqual(manifest_path.stat().st_mode & 0o777, 0o600)
        original = (path.read_bytes(), manifest_path.read_bytes())
        repeated = subprocess.run([sys.executable, str(RUNNER), str(path), "--", "./warehouses/snowflake"],
                                  cwd=ROOT, env=self.environment, text=True, capture_output=True)
        self.assertEqual(repeated.returncode, 2)
        self.assertEqual((path.read_bytes(), manifest_path.read_bytes()), original)
        inside = ROOT / "test/snowflaketester" / (self.directory.name + ".jsonl")
        for owned in [inside, inside.with_suffix(".manifest.json"), inside.with_suffix(".manifest.tmp")]:
            self.addCleanup(owned.unlink, missing_ok=True)
        rejected = subprocess.run([sys.executable, str(RUNNER), str(inside), "--", "./warehouses/snowflake"],
                                  cwd=ROOT, env=self.environment, text=True, capture_output=True)
        self.assertEqual(rejected.returncode, 2)
        self.assertFalse(inside.exists())


if __name__ == "__main__":
    unittest.main()
