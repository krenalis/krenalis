# Copyright 2026 Open2b. All rights reserved.
# Use of this source code is governed by an Elastic License 2.0
# that can be found in the LICENSE file.

"""Prepare a test-only Snowflake session overlay using the production gate."""

import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True
from chunk_fault import prepare_chunk_fault
from transport import RESULT_TAGS


RESULT_OBSERVER = '''
	tag, _ := ctx.Value(queryTag).(string)
	switch tag {
	case ''' + ",\n\t\t".join(", ".join(json.dumps(tag) for tag in RESULT_TAGS[i:i + 2])
                            for i in range(0, len(RESULT_TAGS), 2)) + ''':
		format := "unknown"
		switch data.QueryResultFormat {
		case "arrow", "json":
			format = data.QueryResultFormat
		default:
			// Unknown metadata cannot attest a result format.
		}
		statement := "other"
		if data.StatementTypeID == statementTypeIDSelect {
			statement = "select"
		}
		fmt.Printf("Snowflake result format observed tag=%s format=%s statement=%s\\n", tag, format, statement)
		fmt.Printf("Snowflake result chunks observed tag=%s chunks=%d statement=%s\\n", tag, len(data.Chunks), statement)
	default:
		// Only the closed set of campaign application reads is observed.
	}
'''


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    parser.add_argument("--timezone", required=True, choices=["UTC", "America/Los_Angeles"])
    parser.add_argument("--profile", default="standard", choices=["standard", "timeout", "cancellation"])
    parser.add_argument("--result-format", choices=["ARROW", "JSON"])
    parser.add_argument("--chunk-fault", action="store_true", help="generate driver hooks for -tags krenalis_chunkfault")
    parser.add_argument("--http", action="store_true", help="run the HTTP server inside the isolated test build")
    parser.add_argument("--http-assets", type=Path, help="generated Admin assets outside the checkout")
    options = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    if options.chunk_fault and (options.result_format != "ARROW" or options.profile != "standard" or options.http):
        parser.error("chunk fault verification requires ARROW, the standard session profile and no HTTP overlay")
    if options.http != (options.http_assets is not None):
        parser.error("--http and --http-assets must be provided together")
    assets = options.http_assets.resolve() if options.http else None
    if options.http:
        if options.result_format != "ARROW" or options.profile != "standard":
            parser.error("HTTP verification requires ARROW and the standard session profile")
        if assets.is_relative_to(root) or not (assets / "index.js.br").is_file():
            parser.error("use generated HTTP assets outside the checkout")
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
            if text.count("const profileFilterVerified = true") != 1:
                parser.error("requires an open production gate; gate overlays are not allowed")
            continue
        else:
            qualifier = "" if name.startswith("test/") else "snowflaketester."
            edits = [
                ('"CLIENT_TELEMETRY_ENABLED": falseStrPtr,',
                 '"CLIENT_TELEMETRY_ENABLED": falseStrPtr,\n' +
                 '\t\t\t"TIMEZONE": new(' + json.dumps(options.timezone) + '),\n' +
                 '\t\t\t"TIMESTAMP_TYPE_MAPPING": new("TIMESTAMP_NTZ"),\n' +
                 '\t\t\t"TIMESTAMP_INPUT_FORMAT": new("AUTO"),\n' +
                 '\t\t\t"STATEMENT_TIMEOUT_IN_SECONDS": new("' + timeout + '"),' +
                 ('\n\t\t\t"GO_QUERY_RESULT_FORMAT": new(' + json.dumps(options.result_format) + '),'
                  if options.result_format is not None else '') +
                 ('\n\t\t\t"USE_CACHED_RESULT": new("false"),' if options.profile != "standard" else '')),
                ('gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, cfg)',
                 qualifier + 'SessionConnector{Connector: gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, cfg), ' +
                 'Timezone: ' + json.dumps(options.timezone) +
                 (', ResultFormat: ' + json.dumps(options.result_format) if options.result_format is not None else '') +
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

    if options.result_format is not None:
        for name, old, new in [
            ("vendor/github.com/snowflakedb/gosnowflake/v2/connection_util.go",
             "data execResponseData) chunkDownloader {\n",
             "data execResponseData) chunkDownloader {\n" + RESULT_OBSERVER),
            ("test/snowflaketester/result_format.go", 'const requestedResultFormat = ""',
             'const requestedResultFormat = ' + json.dumps(options.result_format)),
        ]:
            source = root / name
            text = source.read_text()
            if text.count(old) != 1:
                parser.error("expected one overlay anchor in " + name)
            replacements[str(source)] = (directory / name, text.replace(old, new))

    if options.chunk_fault:
        try:
            prepare_chunk_fault(root, directory, replacements)
        except ValueError as error:
            parser.error(str(error))

    if options.http:
        for name, edits in [
            ("test/krenalistester/krenalistester.go", [
                ("const launchKrenalisExternally = true", "const launchKrenalisExternally = false"),
                ("const needToGenerateAssets = !launchKrenalisExternally", "const needToGenerateAssets = false"),
                ('assets := os.DirFS(filepath.Join(k.repo, "admin", "assets"))',
                 'assets := os.DirFS(' + json.dumps(str(assets)) + ')'),
            ]),
            ("core/profiles_query.go", [
                ('"github.com/krenalis/krenalis/core/internal/unified"',
                 '"github.com/krenalis/krenalis/core/internal/unified"\n' +
                 '\t"github.com/krenalis/krenalis/test/snowflaketester"'),
                ('\t\tprofiles, schema, total, err := this.profiles(ctx, query.Properties, filter,',
                 '\t\tctx = snowflaketester.ServiceContext(ctx, snowflaketester.ServiceRecordTag)\n' +
                 '\t\tprofiles, schema, total, err := this.profiles(ctx, query.Properties, filter,'),
                ('\tresult, err := this.store.UnifiedProfiles(ctx, &root, unified.Bindings{}, at, unified.Page{',
                 '\tctx = snowflaketester.ServiceContext(ctx, snowflaketester.ServiceQueryTag)\n' +
                 '\tresult, err := this.store.UnifiedProfiles(ctx, &root, unified.Bindings{}, at, unified.Page{'),
            ]),
            ("core/profiles_explanation.go", [
                ('"github.com/krenalis/krenalis/core/internal/unified"',
                 '"github.com/krenalis/krenalis/core/internal/unified"\n' +
                 '\t"github.com/krenalis/krenalis/test/snowflaketester"'),
                ('\tresult, err := this.store.ExplainUnifiedProfile(ctx, &root, unified.Bindings{}, instant, profile)',
                 '\tctx = snowflaketester.ServiceContext(ctx, snowflaketester.ServiceExplanationTag)\n' +
                 '\tresult, err := this.store.ExplainUnifiedProfile(ctx, &root, unified.Bindings{}, instant, profile)'),
            ]),
        ]:
            source = root / name
            text = source.read_text()
            for old, new in edits:
                if text.count(old) != 1:
                    parser.error("expected one HTTP overlay anchor in " + name)
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
    print(json.dumps({"overlay": str(overlay), "timezone": options.timezone, "profile": options.profile,
                      "result_format": options.result_format, "chunk_fault": options.chunk_fault, "http": options.http,
                      "http_assets": str(assets) if assets is not None else None}))


if __name__ == "__main__":
    main()
