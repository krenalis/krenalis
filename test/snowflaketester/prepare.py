# Copyright 2026 Open2b. All rights reserved.
# Use of this source code is governed by an Elastic License 2.0
# that can be found in the LICENSE file.

"""Prepare a test-only Snowflake gate and session overlay from current sources."""

import argparse
import json
import os
from pathlib import Path
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    parser.add_argument("--timezone", required=True, choices=["UTC", "America/Los_Angeles"])
    parser.add_argument("--profile", default="standard", choices=["standard", "timeout", "cancellation"])
    options = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    directory = options.directory.resolve()
    if directory.is_relative_to(root):
        parser.error("write overlays outside the checkout")
    if directory.exists() and (not directory.is_dir() or any(directory.iterdir())):
        parser.error("use a fresh or empty overlay directory")

    replacements = {}
    timeout = "10" if options.profile == "timeout" else "60"
    for name in ["warehouses/snowflake/profile_filter.go", "warehouses/snowflake/warehouse.go",
                 "connectors/snowflake/snowflake.go", "test/snowflaketester/connection.go"]:
        source = root / name
        text = source.read_text()
        if name.endswith("profile_filter.go"):
            edits = [("const profileFilterVerified = false", "const profileFilterVerified = true")]
        else:
            qualifier = "" if name.startswith("test/") else "snowflaketester."
            edits = [
                ('"CLIENT_TELEMETRY_ENABLED": falseStrPtr,',
                 '"CLIENT_TELEMETRY_ENABLED": falseStrPtr,\n' +
                 '\t\t\t"TIMEZONE": new(' + json.dumps(options.timezone) + '),\n' +
                 '\t\t\t"TIMESTAMP_TYPE_MAPPING": new("TIMESTAMP_NTZ"),\n' +
                 '\t\t\t"TIMESTAMP_INPUT_FORMAT": new("AUTO"),\n' +
                 '\t\t\t"STATEMENT_TIMEOUT_IN_SECONDS": new("' + timeout + '"),' +
                 ('\n\t\t\t"USE_CACHED_RESULT": new("false"),' if options.profile != "standard" else '')),
                ('gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, cfg)',
                 qualifier + 'SessionConnector{Connector: gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, cfg), ' +
                 'Timezone: ' + json.dumps(options.timezone) +
                 (', Profile: ' + json.dumps(options.profile) if options.profile != "standard" else '') + '}'),
            ]
            if qualifier:
                edits.append(('"github.com/krenalis/krenalis/tools/json"',
                              '"github.com/krenalis/krenalis/test/snowflaketester"\n' +
                              '\t"github.com/krenalis/krenalis/tools/json"'))
        for old, new in edits:
            if text.count(old) != 1:
                parser.error("expected one overlay anchor in " + name)
            text = text.replace(old, new)
        replacements[str(source)] = (directory / name, text)

    directory.mkdir(mode=0o700, parents=True, exist_ok=True)
    for target, text in replacements.values():
        target.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        with os.fdopen(os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), "w") as output:
            output.write(text)
    subprocess.run(["gofmt", "-w", *(str(target) for target, _ in replacements.values())], check=True)
    overlay = directory / "overlay.json"
    with os.fdopen(os.open(overlay, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), "w") as output:
        json.dump({"Replace": {source: str(target) for source, (target, _) in replacements.items()}}, output)
        output.write("\n")
    print(json.dumps({"overlay": str(overlay), "timezone": options.timezone, "profile": options.profile}))


if __name__ == "__main__":
    main()
