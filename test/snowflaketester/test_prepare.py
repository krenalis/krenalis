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


ROOT = Path(__file__).resolve().parents[2]
PREPARE = Path(__file__).with_name("prepare.py")


class PrepareTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="krenalis-prepare-test-")
        self.addCleanup(temporary.cleanup)
        self.directory = Path(temporary.name)
        self.environment = {key: value for key, value in os.environ.items()
                            if not key.startswith("KRENALIS_SNOWFLAKE_TESTER_")}

    def prepare(self, directory, timezone, profile="standard"):
        return subprocess.run([sys.executable, "-B", str(PREPARE), str(directory), "--timezone", timezone, "--profile", profile],
                              cwd=ROOT, env=self.environment, capture_output=True, text=True)

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
