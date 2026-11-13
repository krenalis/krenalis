# Copyright 2026 Open2b. All rights reserved.
# Use of this source code is governed by an Elastic License 2.0
# that can be found in the LICENSE file.

import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest

from prepare import RESULT_OBSERVER
from chunk_fault import DRIVER, prepare_chunk_fault
from transport import RESULT_TAGS


ROOT = Path(__file__).resolve().parents[2]
PREPARE = Path(__file__).with_name("prepare.py")


class PrepareTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="krenalis-prepare-test-")
        self.addCleanup(temporary.cleanup)
        self.directory = Path(temporary.name)
        self.environment = {key: value for key, value in os.environ.items()
                            if not key.startswith("KRENALIS_SNOWFLAKE_TESTER_")}

    def prepare(self, directory, timezone, profile="standard", result_format=None, http=False, http_assets=None, chunk_fault=False):
        options = ["--result-format", result_format] if result_format is not None else []
        if chunk_fault:
            options.append("--chunk-fault")
        if http:
            options.append("--http")
        if http_assets is not None:
            options.extend(["--http-assets", str(http_assets)])
        return subprocess.run([sys.executable, "-B", str(PREPARE), str(directory), "--timezone", timezone, "--profile", profile, *options],
                              cwd=ROOT, env=self.environment, capture_output=True, text=True)

    def test_entry_points_do_not_write_bytecode(self):
        directory = self.directory / "scripts"
        directory.mkdir()
        for name in ("prepare.py", "run.py", "transport.py", "chunk_fault.py"):
            (directory / name).write_bytes(PREPARE.with_name(name).read_bytes())
        environment = dict(self.environment)
        environment.pop("PYTHONDONTWRITEBYTECODE", None)
        environment.pop("PYTHONPYCACHEPREFIX", None)
        for name in ("prepare.py", "run.py"):
            with self.subTest(script=name):
                result = subprocess.run([sys.executable, str(directory / name), "--help"],
                                        env=environment, capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertFalse(list(directory.rglob("*.pyc")))

    def test_gate_and_all_connection_paths(self):
        for timezone in ["UTC", "America/Los_Angeles"]:
            with self.subTest(timezone=timezone):
                directory = (self.directory / timezone.replace("/", "-")).resolve()
                result = self.prepare(directory, timezone)
                self.assertEqual(result.returncode, 0, result.stderr)
                overlay = directory / "overlay.json"
                replacements = json.loads(overlay.read_text())["Replace"]
                self.assertEqual(len(replacements), 4)
                for original, target in replacements.items():
                    original, target = Path(original), Path(target)
                    source = original.read_text()
                    text = target.read_text()
                    self.assertTrue(target.is_relative_to(directory))
                    self.assertEqual(target.stat().st_mode & 0o777, 0o600)
                    if original.name == "profile_filter.go":
                        self.assertEqual(text, source.replace("const profileFilterVerified = false",
                                                             "const profileFilterVerified = true"))
                    else:
                        for name, value in [("TIMEZONE", timezone), ("TIMESTAMP_TYPE_MAPPING", "TIMESTAMP_NTZ"),
                                            ("TIMESTAMP_INPUT_FORMAT", "AUTO"), ("STATEMENT_TIMEOUT_IN_SECONDS", "60")]:
                            self.assertEqual(len(re.findall('"' + name + r'":\s+new\("' + value + r'"\)', text)), 1)
                        self.assertEqual(text.count("SessionConnector{"), 1)
                        self.assertEqual(text.count('Timezone: "' + timezone + '"'), 1)
                        if original.parent.name == "snowflaketester":
                            self.assertNotIn('"github.com/krenalis/krenalis/test/snowflaketester"', text)
                        else:
                            self.assertEqual(text.count('"github.com/krenalis/krenalis/test/snowflaketester"'), 1)
                        self.assertNotIn("SessionConnector{", source)
                        self.assertNotIn('"STATEMENT_TIMEOUT_IN_SECONDS"', source)
                formatted = subprocess.run(["gofmt", "-l", *replacements.values()], capture_output=True, text=True)
                self.assertEqual((formatted.returncode, formatted.stdout, formatted.stderr), (0, "", ""))
                self.assertEqual(overlay.stat().st_mode & 0o777, 0o600)

    def test_interruption_profiles(self):
        for profile, timeout in [("timeout", "10"), ("cancellation", "60")]:
            with self.subTest(profile=profile):
                directory = self.directory / profile
                result = self.prepare(directory, "UTC", profile)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(json.loads(result.stdout)["profile"], profile)
                replacements = json.loads((directory / "overlay.json").read_text())["Replace"]
                for original, target in replacements.items():
                    if Path(original).name == "profile_filter.go":
                        continue
                    text = Path(target).read_text()
                    self.assertRegex(text, r'"STATEMENT_TIMEOUT_IN_SECONDS":\s+new\("' + timeout + r'"\)')
                    self.assertRegex(text, r'"USE_CACHED_RESULT":\s+new\("false"\)')
                    self.assertIn('Profile: "' + profile + '"', text)
        rejected = self.prepare(self.directory / "invalid-profile", "UTC", "unknown")
        self.assertEqual(rejected.returncode, 2)

    def test_result_format_overlay(self):
        driver = ROOT / "vendor/github.com/snowflakedb/gosnowflake/v2/connection_util.go"
        tags = re.findall(r'"(krenalis-transport/[^"\n]+)"',
                          (ROOT / "test/snowflaketester/result_format.go").read_text())
        self.assertEqual(sorted(tags), sorted(RESULT_TAGS))
        for requested in ["ARROW", "JSON"]:
            with self.subTest(format=requested):
                directory = self.directory / requested
                result = self.prepare(directory, "UTC", result_format=requested)
                self.assertEqual(result.returncode, 0, result.stderr)
                replacements = json.loads((directory / "overlay.json").read_text())["Replace"]
                self.assertEqual(len(replacements), 6)
                instrumented = Path(replacements[str(driver)]).read_text()
                self.assertEqual(instrumented.replace(RESULT_OBSERVER, ""), driver.read_text())
                for name in ["warehouses/snowflake/warehouse.go", "connectors/snowflake/snowflake.go",
                             "test/snowflaketester/connection.go"]:
                    text = Path(replacements[str(ROOT / name)]).read_text()
                    self.assertRegex(text, r'"GO_QUERY_RESULT_FORMAT":\s+new\("' + requested + r'"\)')
                    self.assertIn('ResultFormat: "' + requested + '"', text)
                selection = Path(replacements[str(ROOT / "test/snowflaketester/result_format.go")]).read_text()
                self.assertIn('const requestedResultFormat = "' + requested + '"', selection)
        rejected = self.prepare(self.directory / "invalid-format", "UTC", result_format="unknown")
        self.assertEqual(rejected.returncode, 2)

    def test_observer_reads_response_metadata_only_for_fixture_tags(self):
        directory = self.directory / "observer"
        result = self.prepare(directory, "UTC", result_format="ARROW")
        self.assertEqual(result.returncode, 0, result.stderr)
        overlay = directory / "overlay.json"
        replacements = json.loads(overlay.read_text())["Replace"]
        probe = directory / "observer_test.go"
        probe.write_text('''package gosnowflake
import (
    "context"
    "testing"
)
func TestTransportOverlayObservation(t *testing.T) {
    tags := []string{
''' + "".join("        " + json.dumps(tag) + ",\n" for tag in RESULT_TAGS) + '''    }
    tags = append(tags, "", "krenalis-session-verification", "krenalis-transport/unknown")
    for _, tag := range tags {
        for _, format := range []string{"arrow", "json", "untrusted"} {
            for _, statement := range []int64{statementTypeIDSelect, statementTypeIDSelect + 1} {
                ctx := context.WithValue(t.Context(), queryTag, tag)
                downloader := populateChunkDownloader(ctx, nil, execResponseData{QueryResultFormat: format, StatementTypeID: statement})
                if string(downloader.getQueryResultFormat()) != format {
                    t.Fatalf("expected original format %s, got %s", format, downloader.getQueryResultFormat())
                }
            }
        }
    }
}
''')
        replacements[str(ROOT / "vendor/github.com/snowflakedb/gosnowflake/v2/transport_overlay_test.go")] = str(probe)
        overlay.write_text(json.dumps({"Replace": replacements}))
        command = ["go", "test", "-mod=vendor", "-overlay", str(overlay),
                   "./vendor/github.com/snowflakedb/gosnowflake/v2", "-run", "^TestTransportOverlayObservation$",
                   "-count=1", "-json"]
        checked = subprocess.run(command, cwd=ROOT, env=self.environment, capture_output=True, text=True, timeout=180)
        self.assertEqual(checked.returncode, 0, checked.stdout + checked.stderr)
        output = "".join(json.loads(line).get("Output", "") for line in checked.stdout.splitlines())
        records = re.findall(r'Snowflake result format observed tag=(\S+) format=(\S+) statement=(\S+)', output)
        self.assertEqual(sorted(records), sorted((tag, actual, statement) for tag in RESULT_TAGS
                                                for actual in ("arrow", "json", "unknown")
                                                for statement in ("select", "other")))
        self.assertNotIn("format=untrusted", output)
        chunks = re.findall(r'Snowflake result chunks observed tag=(\S+) chunks=(\S+) statement=(\S+)', output)
        self.assertEqual(sorted(chunks), sorted((tag, "0", statement) for tag in RESULT_TAGS
                                               for _ in ("arrow", "json", "unknown")
                                               for statement in ("select", "other")))

    def test_chunk_fault_requires_arrow_and_a_standard_non_http_build(self):
        for index, options in enumerate([{}, {"result_format": "JSON"},
                {"result_format": "ARROW", "profile": "timeout"},
                {"result_format": "ARROW", "http": True}]):
            with self.subTest(options=options):
                directory = self.directory / str(index)
                result = self.prepare(directory, "UTC", chunk_fault=True, **options)
                self.assertEqual(result.returncode, 2)
                self.assertFalse(directory.exists())

    def test_chunk_fault_anchors_fail_closed(self):
        for name in ("connection_util.go", "chunk_downloader.go", "rows.go"):
            with self.subTest(source=name):
                root = self.directory / name
                vendor = root / DRIVER
                vendor.mkdir(parents=True)
                for source in ("connection_util.go", "chunk_downloader.go", "rows.go"):
                    text = (ROOT / DRIVER / source).read_text() if source != name else "changed vendor"
                    (vendor / source).write_text(text)
                with self.assertRaisesRegex(ValueError, "expected one chunk fault overlay anchor"):
                    prepare_chunk_fault(root, self.directory / "output", {})
                self.assertFalse((self.directory / "output").exists())

    def test_chunk_fault_uses_real_downloader_retries_without_a_canceled_caller(self):
        directory = self.directory / "chunk-fault"
        result = self.prepare(directory, "UTC", result_format="ARROW", chunk_fault=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(json.loads(result.stdout)["chunk_fault"])
        overlay = directory / "overlay.json"
        replacements = json.loads(overlay.read_text())["Replace"]
        self.assertEqual(len(replacements), 9)
        self.assertFalse((ROOT / DRIVER / "chunk_fault.go").exists())
        probe = directory / "chunk_fault_overlay_test.go"
        probe.write_text('''package gosnowflake
import (
    "database/sql/driver"
    "errors"
    "sync"
    "testing"
    "testing/synctest"
    "github.com/snowflakedb/gosnowflake/v2/internal/query"
)
func TestChunkFaultOverlay(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {

        for _, tag := range []string{"", "krenalis-session-verification", "krenalis-transport/record-explanation"} {
            ctx, _ := WithChunkFault(WithQueryTag(t.Context(), tag))
            if chunkFaultObservation(ctx) != nil {
                t.Fatalf("expected unmarked read for tag %q, got an active fault", tag)
            }
        }
        if chunkFaultObservation(WithQueryTag(t.Context(), "krenalis-transport/record-query")) != nil {
            t.Fatal("expected an unmarked query without a fault, got an active fault")
        }
        ctx, observed := WithChunkFault(WithQueryTag(t.Context(), "krenalis-transport/record-query"))
        scd := populateChunkDownloader(ctx, nil, execResponseData{
            Chunks: []query.ExecResponseChunk{{}, {}}, QueryResultFormat: "arrow",
        }).(*snowflakeChunkDownloader)
        scd.ChunksError = make(chan *chunkError, 1)
        scd.DoneDownloadCond = sync.NewCond(&sync.Mutex{})
        scd.FuncDownloadHelper = downloadChunkHelper
        scd.FuncDownload = downloadChunk
        scd.CurrentIndex = -1
        scd.CurrentChunkSize = 1
        scd.CurrentChunk = []chunkRowType{{ArrowRow: []snowflakeValue{int64(1)}}}

        go downloadChunk(ctx, scd, 1)
        synctest.Wait()
        if observed.Attempts.Load() != 0 {
            t.Fatalf("expected no fault before delivery, got %d attempts", observed.Attempts.Load())
        }
        rows := &snowflakeRows{ctx: ctx, ChunkDownloader: scd}
        err := rows.Next(make([]driver.Value, 1))
        if err != nil {
            t.Fatalf("expected a valid row before the fault, got %v", err)
        }
        synctest.Wait()
        for attempt := int64(1); attempt <= observed.ExpectedAttempts(); attempt++ {
            err := scd.checkErrorRetry()
            if attempt == observed.ExpectedAttempts() {
                if err != nil {
                    if !errors.Is(err, ErrChunkFault) {
                        t.Fatalf("expected injected failure, got %v", err)
                    }
                }
                if err == nil {
                    t.Fatal("expected exhausted retries, got success")
                }
            } else if err != nil {
                t.Fatalf("expected retry %d, got %v", attempt, err)
            }
            synctest.Wait()
        }
        if ctx.Err() != nil || observed.Rows.Load() != 1 || observed.Chunks.Load() != 2 ||
            observed.Attempts.Load() != observed.ExpectedAttempts() ||
            int64(scd.ChunksErrorCounter) != observed.ExpectedAttempts()-1 {
            t.Fatalf("expected live caller, one row and %d attempts, got rows=%d chunks=%d attempts=%d retries=%d context=%v",
                observed.ExpectedAttempts(), observed.Rows.Load(), observed.Chunks.Load(), observed.Attempts.Load(),
                scd.ChunksErrorCounter, ctx.Err())
        }

    })
}
''')
        replacements[str(ROOT / DRIVER / "chunk_fault_overlay_test.go")] = str(probe)
        overlay.write_text(json.dumps({"Replace": replacements}))
        checked = subprocess.run(["go", "test", "-mod=vendor", "-race", "-tags", "krenalis_chunkfault",
            "-overlay", str(overlay), "./vendor/github.com/snowflakedb/gosnowflake/v2",
            "-run", "^TestChunkFaultOverlay$", "-count=1", "-timeout=30s"], cwd=ROOT,
            env=self.environment, capture_output=True, text=True, timeout=180)
        self.assertEqual(checked.returncode, 0, checked.stdout + checked.stderr)

    def test_http_overlay_tags_core_reads_and_internal_server(self):
        assets = (self.directory / "assets").resolve()
        assets.mkdir()
        (assets / "index.js.br").write_bytes(b"asset-path-fixture")
        directory = self.directory / "http"
        result = self.prepare(directory, "UTC", result_format="ARROW", http=True, http_assets=assets)
        self.assertEqual(result.returncode, 0, result.stderr)
        replacements = json.loads((directory / "overlay.json").read_text())["Replace"]
        self.assertEqual(len(replacements), 9)
        original = ROOT / "test/krenalistester/krenalistester.go"
        text = Path(replacements[str(original)]).read_text()
        self.assertEqual(text.count("const launchKrenalisExternally = false"), 1)
        self.assertEqual(text.count("const needToGenerateAssets = false"), 1)
        self.assertIn("assets := os.DirFS(" + json.dumps(str(assets)) + ")", text)
        restored = text.replace("const launchKrenalisExternally = false", "const launchKrenalisExternally = true")
        restored = restored.replace("const needToGenerateAssets = false",
                                    "const needToGenerateAssets = !launchKrenalisExternally")
        restored = restored.replace("assets := os.DirFS(" + json.dumps(str(assets)) + ")",
                                    'assets := os.DirFS(filepath.Join(k.repo, "admin", "assets"))')
        self.assertEqual(restored, original.read_text())
        for name, tags in [("profiles_query.go", ["ServiceRecordTag", "ServiceQueryTag"]),
                           ("profiles_explanation.go", ["ServiceExplanationTag"])]:
            source = ROOT / "core" / name
            text = Path(replacements[str(source)]).read_text()
            for tag in tags:
                pattern = r'^[ \t]+ctx = snowflaketester.ServiceContext\(ctx, snowflaketester\.' + tag + r'\)\n'
                self.assertEqual(len(re.findall(pattern, text, re.MULTILINE)), 1)
                text = re.sub(pattern, "", text, flags=re.MULTILINE)
            text = text.replace('\t"github.com/krenalis/krenalis/test/snowflaketester"\n', "")
            self.assertEqual(text, source.read_text())
        marker = (ROOT / "test/krenalistester/server_mode.go").read_text()
        self.assertIn("return !launchKrenalisExternally", marker)
        self.assertTrue(json.loads(result.stdout)["http"])

    def test_http_overlay_requires_explicit_assets_arrow_and_standard_profile(self):
        assets = (self.directory / "assets").resolve()
        assets.mkdir()
        (assets / "index.js.br").write_bytes(b"asset-path-fixture")
        for number, options in enumerate([
                {"http": True}, {"http_assets": assets},
                {"http": True, "http_assets": assets},
                {"http": True, "http_assets": assets, "result_format": "JSON"},
                {"http": True, "http_assets": assets, "result_format": "ARROW", "profile": "timeout"},
                {"http": True, "http_assets": self.directory / "missing", "result_format": "ARROW"},
                {"http": True, "http_assets": ROOT, "result_format": "ARROW"},
        ]):
            with self.subTest(options=options):
                result = self.prepare(self.directory / str(number), "UTC", **options)
                self.assertEqual(result.returncode, 2)
                self.assertFalse((self.directory / str(number)).exists())

    def test_existing_evidence_is_never_overwritten(self):
        directory = self.directory / "overlay"
        directory.mkdir()
        result = self.prepare(directory, "UTC")
        self.assertEqual(result.returncode, 0, result.stderr)
        before = {str(path): path.read_bytes() for path in directory.rglob("*") if path.is_file()}
        rejected = self.prepare(directory, "America/Los_Angeles")
        self.assertEqual(rejected.returncode, 2)
        self.assertEqual({str(path): path.read_bytes() for path in directory.rglob("*") if path.is_file()}, before)

    def test_invalid_timezone_and_checkout_paths_are_rejected(self):
        directory = self.directory / "invalid"
        rejected = self.prepare(directory, "Europe/Rome")
        self.assertEqual(rejected.returncode, 2)
        self.assertFalse(directory.exists())
        inside = ROOT / "test/snowflaketester" / self.directory.name
        rejected = self.prepare(inside, "UTC")
        self.assertEqual(rejected.returncode, 2)
        self.assertFalse(inside.exists())


if __name__ == "__main__":
    unittest.main()
