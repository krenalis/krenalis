# Copyright 2026 Open2b. All rights reserved.
# Use of this source code is governed by an Elastic License 2.0
# that can be found in the LICENSE file.

from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import generate_catalog
from generate_catalog import native_type


class NativeChoicesTest(unittest.TestCase):
    def test_reviewed_mixed_fields(self):
        for event, path, expected in [
            ("identity.identified", "traits.company.id", "types.String()"),
            ("identity.group_associated", "traits.employees", "types.String()"),
            ("commerce.checkout_step_completed", "properties.value", "types.JSON()"),
            ("commerce.payment_info_entered", "properties.value", "types.JSON()"),
        ]:
            with self.subTest(event=event, path=path):
                node = {"path": path, "description": "Reworded independently of its type."}
                self.assertEqual(native_type(node, event, lambda *_: ["string", "number"], 0), expected)

    def test_unreviewed_mixed_field(self):
        with self.assertRaisesRegex(ValueError, "Unreviewed mixed kinds"):
            native_type({"path": "properties.new_field"}, "custom.event", lambda *_: ["string", "number"], 0)

    def test_changed_source_kinds(self):
        with self.assertRaisesRegex(ValueError, "Source kinds changed"):
            native_type({"path": "traits.company.id"}, "identity.identified", lambda *_: ["boolean"], 0)

    def test_names_alone_do_not_assign_semantics(self):
        for path in ["properties.total", "properties.price", "properties.conversation_duration"]:
            with self.subTest(path=path):
                self.assertEqual(
                    native_type({"path": path}, "custom.event", lambda *_: ["number"], 0),
                    "types.Float(64).Real()",
                )

    def test_check_detects_drift_without_writing(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "catalog.go"
            original = b"// stale generated catalog\n"
            output.write_bytes(original)
            with patch.object(generate_catalog, "OUTPUT", output), patch("sys.argv", ["generate_catalog.py", "--check"]):
                with self.assertRaisesRegex(SystemExit, "differs from the generator"):
                    generate_catalog.main()
            self.assertEqual(output.read_bytes(), original)


if __name__ == "__main__":
    unittest.main()
